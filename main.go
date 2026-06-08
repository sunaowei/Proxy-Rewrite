package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func main() {
	store, err := NewStore()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	broadcaster := NewLogBroadcaster()

	proxy := NewProxyServer(store, broadcaster)
	handler := NewHandler(store, broadcaster)

	webMux := http.NewServeMux()
	handler.RegisterRoutes(webMux)

	webMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			http.ServeFile(w, r, "web/index.html")
			return
		}
		http.ServeFile(w, r, "web"+r.URL.Path)
	})

	go func() {
		webAddr := ":" + store.WebPort()
		log.Printf("Web UI listening on http://localhost%s", webAddr)
		if err := http.ListenAndServe(webAddr, webMux); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Web server error: %v", err)
		}
	}()

	go func() {
		if err := proxy.Start(store.ProxyPort()); err != nil {
			log.Printf("Proxy server stopped: %v", err)
		}
	}()

	fmt.Println("========================================")
	fmt.Printf("  Proxy Rewrite Tool\n")
	fmt.Printf("  Proxy:  http://localhost:%s\n", store.ProxyPort())
	fmt.Printf("  Web UI: http://localhost:%s\n", store.WebPort())
	fmt.Println("========================================")
	fmt.Println("\nConfigure your browser/system proxy to use:")
	fmt.Printf("  HTTP Proxy: 127.0.0.1:%s\n", store.ProxyPort())
	fmt.Println()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("Shutting down...")
	proxy.Stop()
}
