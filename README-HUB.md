# S-UI - Advanced Web Panel for SagerNet/Sing-Box (Multi-Node Support)

## Short Description (for Docker Hub)

```
An advanced Web Panel for SagerNet/Sing-Box with multi-node distributed architecture support.
```

---

## Full Description / Overview

```markdown
# S-UI - Advanced Web Panel for SagerNet/Sing-Box

[![Docker Image](https://img.shields.io/badge/docker-zhengxiongzhao/s--ui-blue)](https://hub.docker.com/r/zhengxiongzhao/s-ui)
[![License](https://img.shields.io/badge/license-GPL--3.0-green)](LICENSE)

> **An advanced Web Panel built for SagerNet/Sing-Box**

## 🚀 Features

- **🌐 Multi-Node Architecture** - Distributed panel management with centralized control
- **📊 Web-based Management** - Intuitive UI for proxy configuration and monitoring
- **🔒 Secure Communication** - HTTP-based panel-to-agent communication with token authentication
- **🔄 Scalable** - Add multiple agent nodes to expand your proxy infrastructure
- **⚡ Sing-Box Powered** - Built on the high-performance SagerNet/Sing-Box core

## 🏗️ Architecture

S-UI supports a distributed multi-node architecture:

```
┌─────────────────────────────────────────────────────────────┐
│                      S-UI Panel (Main Control)              │
│                                                             │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    │
│   │  Web UI     │    │  Config Mgr │    │  Node Mgr   │    │
│   └─────────────┘    └─────────────┘    └─────────────┘    │
│                          │                                  │
│                          ▼                                  │
│   ┌───────────────────────────────────────────────────┐    │
│   │              HTTP Management API                   │    │
│   └───────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                           │
          ┌────────────────┼────────────────┐
          ▼                ▼                ▼
   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
   │ Agent Node 1│  │ Agent Node 2│  │ Agent Node N│
   │  (hk1)      │  │  (us1)      │  │  (jp1)      │
   └─────────────┘  └─────────────┘  └─────────────┘
```

### Component Roles

| Component | Mode | Description |
|-----------|------|-------------|
| **Panel** | `panel` | Main control server - manages web UI, configurations, and coordinates all agent nodes |
| **Agent** | `agent` | Remote node - executes proxy tasks, reports status to panel |

## 📦 Quick Start

### Single Node Deployment

```bash
docker run -d \
  --name s-ui \
  --restart unless-stopped \
  -p 2095:2095 \
  -p 2096:2096 \
  -p 443:443/udp \
  -v ./db:/app/db \
  -v ./cert:/app/cert \
  -e SUI_MODE=panel \
  zhengxiongzhao/s-ui:latest
```

## 🔑 Access Panel

After deployment, access the Web Panel at:

```
http://<your-server-ip>:2095/app/login
```

### 📋 Default Installation Information

| Item | Value |
|------|-------|
| **Panel Port** | `2095` |
| **Panel Path** | `/app/` |
| **Subscription Port** | `2096` |
| **Subscription Path** | `/sub/` |
| **Default User/Password** | `admin` / `admin` |

> ⚠️ **Important:** Please change the default password immediately after first login!

### Multi-Node Deployment

#### 1. Deploy Panel (Main Control Server)

```bash
docker run -d \
  --name s-ui-panel \
  --restart unless-stopped \
  --network host \
  -e SUI_MODE=panel \
  -v ./db:/app/db \
  -v ./cert:/app/cert \
  zhengxiongzhao/s-ui:latest
```

#### 2. Deploy Agent Nodes

```bash
# Agent Node 1 (e.g., Hong Kong)
docker run -d \
  --name s-ui-agent-hk1 \
  --restart unless-stopped \
  -p 2097:2097 \
  -p 51443:443/udp \
  -e SUI_MODE=agent \
  -e SUI_NODE_NAME=hk1 \
  -e SUI_NODE_TOKEN=your-secure-token-here \
  -v ./agent-hk1:/app/db \
  zhengxiongzhao/s-ui:latest

# Agent Node 2 (e.g., US)
docker run -d \
  --name s-ui-agent-us1 \
  --restart unless-stopped \
  -p 2098:2097 \
  -p 51444:443/udp \
  -e SUI_MODE=agent \
  -e SUI_NODE_NAME=us1 \
  -e SUI_NODE_TOKEN=your-secure-token-here \
  -v ./agent-us1:/app/db \
  zhengxiongzhao/s-ui:latest
```

## 📋 Docker Compose

### Multi-Node Example

```yaml
services:
  # Panel (main control server)
  s-ui-panel:
    image: zhengxiongzhao/s-ui:latest
    restart: unless-stopped
    network_mode: host
    environment:
      - SUI_MODE=panel
    volumes:
      - ./test-db:/app/db
      - ./test-cert:/app/cert

  # Agent (remote node hk1)
  s-ui-agent-hk1:
    image: zhengxiongzhao/s-ui:latest
    restart: unless-stopped
    ports:
      - "2097:2097"
      - "51443:443/udp"
    environment:
      - SUI_MODE=agent
      - SUI_NODE_NAME=hk1
      - SUI_NODE_TOKEN=test-token-hk1-12345
    volumes:
      - ./test-agent-hk1:/app/db
```

## ⚙️ Environment Variables

| Variable         |                      Type                      | Default       | Description |
| ---------------- | :--------------------------------------------: | :------------ | :---------- |
| `SUI_MODE`       |             `"panel"` \| `"agent"`             | `"panel"`     | Running mode: `panel` (main control) or `agent` (remote node) |
| `SUI_NODE_TOKEN` |                    `string`                    | -             | Authentication token for agent registration (required for agent mode) |
| `SUI_NODE_NAME`  |                    `string`                    | -             | Custom name for the agent node (only used in agent mode) |
| `SUI_AGENT_LISTEN`|                   `string`                    | `"0.0.0.0"`   | Listen IP for agent API server |
| `SUI_AGENT_PORT` |                   `integer`                    | `2097`        | Listen Port for agent API server |
| `SUI_ENABLE_SUB` |                   `boolean`                    | `true` (panel) / `false` (agent) | Enable subscription service |
| `SUI_ENABLE_WEB` |                   `boolean`                    | `true` (panel) / `false` (agent) | Enable web service |
| `SUI_LOG_LEVEL`  | `"debug"` \| `"info"` \| `"warn"` \| `"error"` \| `"silent"` | `"info"` | Log level |
| `SUI_DEBUG`      |                   `boolean`                    | `false`       | Enable debug mode |
| `SUI_BIN_FOLDER` |                    `string`                    | `"bin"`       | Path to binary folder |
| `SUI_DB_FOLDER`  |                    `string`                    | `"db"`        | Path to database folder |
| `SINGBOX_API`    |                    `string`                    | -             | Sing-Box API address |

## 🔌 Ports

| Port | Protocol | Description |
|------|----------|-------------|
| `2095` | TCP | Web Panel (Default) |
| `2096` | TCP | Subscription (Default) |
| `2097` | TCP | Agent Management API |
| `443` | UDP | Proxy traffic (Sing-Box) |

## 📁 Volumes

| Path | Description |
|------|-------------|
| `/app/db` | Database, configuration and agent data |
| `/app/cert` | TLS certificates |

## 🔐 Security

- Use strong, unique tokens for each agent node
- Enable TLS for production deployments
- Restrict network access to management ports
- Regularly update to the latest version

## 📝 License

This project is licensed under the GPL-3.0 License.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

---

**Built with ❤️ for the SagerNet/Sing-Box community**
```
