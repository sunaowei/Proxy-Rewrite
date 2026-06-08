# Proxy-Rewrite

HTTP/HTTPS 代理转发工具，支持通过 Web 界面无限配置 URL 重写规则。

## 功能

- HTTP 正向代理（支持 HTTPS CONNECT 隧道）
- URL 通配符匹配和重写转发
- Web 管理界面（规则增删改查、开关控制）
- 实时请求日志（SSE 推送）
- 亮色/暗色主题切换
- 日志过滤（按方法、状态码、关键字）
- 规则无数量限制，JSON 文件持久化

## 快速开始

```bash
# 编译
go build -o proxy-rewrite .

# 启动
./proxy-rewrite
```

启动后：
- **Web 管理界面**: http://localhost:9090
- **代理地址**: `127.0.0.1:8080`

## 配置浏览器代理

将浏览器或系统的 HTTP 代理设置为 `127.0.0.1:8080`，所有经过代理的请求会根据规则自动重写转发。

### curl 测试

```bash
# 直接通过代理请求（会被规则匹配并重写）
curl -x http://localhost:8080 http://api.example.com:8080/v1/users

# 不匹配规则的请求会透传
curl -x http://localhost:8080 http://httpbin.org/get
```

## 规则配置

规则存储在 `data/rules.json`，格式如下：

```json
{
  "proxy_port": "8080",
  "web_port": "9090",
  "rules": [
    {
      "id": "自动生成",
      "name": "规则名称",
      "match_pattern": "http://原始地址/path/*",
      "target_url": "http://目标地址/path/*",
      "enabled": true,
      "created_at": "2026-06-08T10:00:00+08:00"
    }
  ]
}
```

### 通配符匹配规则

- `*` 匹配任意字符
- 匹配时只替换协议+主机+端口+路径前缀，`*` 部分保持不变

示例：
```
匹配: http://api.example.com:8080/v1/*
目标: http://127.0.0.1:3000/v1/*

请求: http://api.example.com:8080/v1/users/list
结果: http://127.0.0.1:3000/v1/users/list
```

多通配符示例：
```
匹配: http://*.*:8080/api/*
目标: http://127.0.0.1:3000/api/*

请求: http://api.example.com:8080/api/register
结果: http://127.0.0.1:3000/api/register
```

## Web API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/rules | 获取所有规则 |
| POST | /api/rules | 新增规则 |
| PUT | /api/rules/:id | 更新规则 |
| DELETE | /api/rules/:id | 删除规则 |
| PATCH | /api/rules/:id/toggle | 切换启用/禁用 |
| GET | /api/logs | SSE 实时日志流 |

## 后台运行

```bash
# 使用 nohup
nohup ./proxy-rewrite > proxy.log 2>&1 &

# 或使用 systemd (创建 /etc/systemd/system/proxy-rewrite.service)
```

## 停止

```bash
pkill -f proxy-rewrite
```

## 文件结构

```
proxy-rewrite/
├── main.go           # 程序入口
├── config.go         # 配置模型定义
├── store.go          # 规则存储（JSON 文件读写）
├── rewriter.go       # URL 通配符匹配和重写引擎
├── proxy.go          # HTTP/HTTPS 代理服务器
├── handler.go        # Web API 路由处理
├── web/
│   └── index.html    # Web 管理界面
├── data/
│   └── rules.json    # 规则持久化存储（自动创建）
├── go.mod
└── README.md
```

## License

MIT
