package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type LogEntry struct {
	Time      string `json:"time"`
	Original  string `json:"original"`
	Rewritten string `json:"rewritten"`
	RuleName  string `json:"rule_name"`
	Method    string `json:"method"`
	Status    int    `json:"status"`
}

type LogBroadcaster struct {
	clients map[chan LogEntry]struct{}
}

func NewLogBroadcaster() *LogBroadcaster {
	return &LogBroadcaster{clients: make(map[chan LogEntry]struct{})}
}

func (lb *LogBroadcaster) Subscribe() chan LogEntry {
	ch := make(chan LogEntry, 100)
	lb.clients[ch] = struct{}{}
	return ch
}

func (lb *LogBroadcaster) Unsubscribe(ch chan LogEntry) {
	delete(lb.clients, ch)
	close(ch)
}

func (lb *LogBroadcaster) Broadcast(entry LogEntry) {
	for ch := range lb.clients {
		select {
		case ch <- entry:
		default:
		}
	}
}

type ProxyServer struct {
	store       *Store
	broadcaster *LogBroadcaster
	listener    net.Listener
}

func NewProxyServer(store *Store, broadcaster *LogBroadcaster) *ProxyServer {
	return &ProxyServer{
		store:       store,
		broadcaster: broadcaster,
	}
}

func (p *ProxyServer) Start(port string) error {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	p.listener = ln
	log.Printf("Proxy server listening on :%s", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				continue
			}
			return err
		}
		go p.handleConn(conn)
	}
}

func (p *ProxyServer) handleConn(conn net.Conn) {
	defer conn.Close()

	br := bufio.NewReader(conn)
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}

	if req.Method == http.MethodConnect {
		p.handleCONNECT(conn, req)
	} else {
		p.handleHTTP(conn, req)
	}
}

func (p *ProxyServer) handleHTTP(conn net.Conn, req *http.Request) {
	entry := LogEntry{
		Time:     time.Now().Format("15:04:05"),
		Original: req.URL.String(),
		Method:   req.Method,
	}

	if !req.URL.IsAbs() {
		resp := &http.Response{
			StatusCode: 400,
			ProtoMajor: 1,
			ProtoMinor: 1,
			Body:       io.NopCloser(strings.NewReader("Bad Request: non-proxy request\r\n")),
		}
		resp.Write(conn)
		entry.Status = 400
		p.broadcaster.Broadcast(entry)
		return
	}

	rules := p.store.GetRules()
	result := applyRewrite(rules, req.URL.String())

	if result.Matched {
		entry.Rewritten = result.NewURL
		entry.RuleName = result.Rule.Name

		newURL, err := url.Parse(result.NewURL)
		if err == nil {
			req.URL.Scheme = newURL.Scheme
			req.URL.Host = newURL.Host
			req.URL.Path = newURL.Path
			req.Host = newURL.Host
		}
		log.Printf("[REWRITE] %s %s -> %s (rule: %s)", entry.Method, entry.Original, entry.Rewritten, entry.RuleName)
	}

	req.RequestURI = ""

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		entry.Status = http.StatusBadGateway
		p.broadcaster.Broadcast(entry)
		errResp := &http.Response{
			StatusCode: 502,
			ProtoMajor: 1,
			ProtoMinor: 1,
			Body:       io.NopCloser(strings.NewReader("Bad Gateway: " + err.Error() + "\r\n")),
		}
		errResp.Write(conn)
		return
	}
	defer resp.Body.Close()

	entry.Status = resp.StatusCode
	p.broadcaster.Broadcast(entry)

	resp.Write(conn)
}

func (p *ProxyServer) handleCONNECT(conn net.Conn, req *http.Request) {
	host := req.Host
	if !strings.Contains(host, ":") {
		host = host + ":443"
	}

	entry := LogEntry{
		Time:     time.Now().Format("15:04:05"),
		Original: host,
		Method:   "CONNECT",
	}

	destConn, err := net.DialTimeout("tcp", host, 10*time.Second)
	if err != nil {
		entry.Status = http.StatusBadGateway
		p.broadcaster.Broadcast(entry)
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}

	entry.Status = http.StatusOK
	p.broadcaster.Broadcast(entry)
	log.Printf("[CONNECT] %s", host)

	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	done := make(chan struct{})
	go func() { transfer(destConn, conn); close(done) }()
	go func() { transfer(conn, destConn) }()
	<-done
}

func transfer(dst io.WriteCloser, src io.ReadCloser) {
	defer dst.Close()
	defer src.Close()
	io.Copy(dst, src)
}

func (p *ProxyServer) Stop() error {
	if p.listener != nil {
		return p.listener.Close()
	}
	return nil
}
