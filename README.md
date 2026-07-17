# S-UI
**An Advanced Web Panel • Built on SagerNet/Sing-Box**

![](https://img.shields.io/github/v/release/zhengxiongzhao/s-ui.svg)
![S-UI Docker pull](https://img.shields.io/docker/pulls/alireza7/s-ui.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/zhengxiongzhao/s-ui)](https://goreportcard.com/report/github.com/zhengxiongzhao/s-ui)
[![Downloads](https://img.shields.io/github/downloads/zhengxiongzhao/s-ui/total.svg)](https://img.shields.io/github/downloads/zhengxiongzhao/s-ui/total.svg)
[![License](https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true)](https://www.gnu.org/licenses/gpl-3.0.en.html)

> **Disclaimer:** This project is only for personal learning and communication, please do not use it for illegal purposes, please do not use it in a production environment

**If you think this project is helpful to you, you may wish to give a**:star2:

**Want to contribute?** See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, coding conventions, testing, and the pull request process.

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/alireza7)

<a href="https://nowpayments.io/donation/alireza7" target="_blank" rel="noreferrer noopener">
   <img src="https://nowpayments.io/images/embeds/donation-button-white.svg" alt="Crypto donation button by NOWPayments">
</a>

## Quick Overview
| Features                               |      Enable?       |
| -------------------------------------- | :----------------: |
| Multi-Protocol                         | :heavy_check_mark: |
| Multi-Language                         | :heavy_check_mark: |
| Multi-Client/Inbound                   | :heavy_check_mark: |
| Advanced Traffic Routing Interface     | :heavy_check_mark: |
| Client & Traffic & System Status       | :heavy_check_mark: |
| Subscription Link (link/json/clash + info)| :heavy_check_mark: |
| Dark/Light Theme                       | :heavy_check_mark: |
| API Interface                          | :heavy_check_mark: |

## Supported Platforms
| Platform | Architecture | Status |
|----------|--------------|---------|
| Linux    | amd64, arm64, armv7, armv6, armv5, 386, s390x | ✅ Supported |
| Windows  | amd64, 386, arm64 | ✅ Supported |
| macOS    | amd64, arm64 | 🚧 Experimental |

## Screenshots

!["Main"](https://github.com/zhengxiongzhao/s-ui-frontend/raw/main/media/main.png)

[Other UI Screenshots](https://github.com/zhengxiongzhao/s-ui-frontend/blob/main/screenshots.md)

## API Documentation

[API-Documentation Wiki](https://github.com/zhengxiongzhao/s-ui/wiki/API-Documentation)

## Default Installation Information
- Panel Port: 2095
- Panel Path: /app/
- Subscription Port: 2096
- Subscription Path: /sub/
- User/Password: admin

## Install & Upgrade to Latest Version

### Linux/macOS
```sh
bash <(curl -Ls https://raw.githubusercontent.com/zhengxiongzhao/s-ui/master/install.sh)
```

### Windows
1. Download the latest Windows release from [GitHub Releases](https://github.com/zhengxiongzhao/s-ui/releases/latest)
2. Extract the ZIP file
3. Run `install-windows.bat` as Administrator
4. Follow the installation wizard

## Install legacy Version

**Step 1:** To install your desired legacy version, add the version to the end of the installation command. e.g., ver `1.0.0`:

```sh
VERSION=1.0.0 && bash <(curl -Ls https://raw.githubusercontent.com/zhengxiongzhao/s-ui/$VERSION/install.sh) $VERSION
```

## Manual installation

### Linux/macOS
1. Get the latest version of S-UI based on your OS/Architecture from GitHub: [https://github.com/zhengxiongzhao/s-ui/releases/latest](https://github.com/zhengxiongzhao/s-ui/releases/latest)
2. **OPTIONAL** Get the latest version of `s-ui.sh` [https://raw.githubusercontent.com/zhengxiongzhao/s-ui/master/s-ui.sh](https://raw.githubusercontent.com/zhengxiongzhao/s-ui/master/s-ui.sh)
3. **OPTIONAL** Copy `s-ui.sh` to /usr/bin/ and run `chmod +x /usr/bin/s-ui`.
4. Extract s-ui tar.gz file to a directory of your choice and navigate to the directory where you extracted the tar.gz file.
5. Copy *.service files to /etc/systemd/system/ and run `systemctl daemon-reload`.
6. Enable autostart and start S-UI service using `systemctl enable s-ui --now`
7. Start sing-box service using `systemctl enable sing-box --now`

### Windows
1. Get the latest Windows version from GitHub: [https://github.com/zhengxiongzhao/s-ui/releases/latest](https://github.com/zhengxiongzhao/s-ui/releases/latest)
2. Download the appropriate Windows package (e.g., `s-ui-windows-amd64.zip`)
3. Extract the ZIP file to a directory of your choice
4. Run `install-windows.bat` as Administrator
5. Follow the installation wizard
6. Access the panel at http://localhost:2095/app

## Uninstall S-UI

```sh
sudo -i

systemctl disable s-ui  --now

rm -f /etc/systemd/system/sing-box.service
systemctl daemon-reload

rm -fr /usr/local/s-ui
rm /usr/bin/s-ui
```

## Install using Docker

<details>
   <summary>Click for details</summary>

### Usage

**Step 1:** Install Docker

```shell
curl -fsSL https://get.docker.com | sh
```

**Step 2:** Install S-UI

#### Single Node Selection (Panel Mode)

```shell
docker run -itd \
    -p 2095:2095 -p 2096:2096 -p 443:443 -p 80:80 \
    -v $PWD/db/:/app/db/ \
    -v $PWD/cert/:/root/cert/ \
    --name s-ui --restart=unless-stopped \
    zhengxiongzhao/s-ui:latest
```

#### Multi-Node Distributed Architecture Selection (Panel + Agent Mode)

##### 1. Deploy Panel (Main Control Server)
```shell
docker run -d \
  --name s-ui-panel \
  --restart unless-stopped \
  --network host \
  -e SUI_MODE=panel \
  -v ./db:/app/db \
  -v ./cert:/app/cert \
  zhengxiongzhao/s-ui:latest
```

##### 2. Deploy Agent Nodes
```shell
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
```

</details>

## Manual run ( contribution )

<details>
   <summary>Click for details</summary>

### Build and run whole project
```shell
./runSUI.sh
```

### Clone the repository
```shell
# clone repository
git clone https://github.com/zhengxiongzhao/s-ui
# clone submodules
git submodule update --init --recursive
```


### - Frontend

Visit [s-ui-frontend](https://github.com/zhengxiongzhao/s-ui-frontend) for frontend code

### - Backend
> Please build frontend once before!

To build backend:
```shell
# remove old frontend compiled files
rm -fr web/html/*
# apply new frontend compiled files
cp -R frontend/dist/ web/html/
# build
go build -o sui main.go
```

To run backend (from root folder of repository):
```shell
./sui
```

</details>

## Languages

- English
- Farsi
- Vietnamese
- Chinese (Simplified)
- Chinese (Traditional)
- Russian

## Features

- **Multi-Node Architecture**: Distribute panels and agents across nodes for centralized management and lightweight clients.
- Supported protocols:
  - General:  Mixed, SOCKS, HTTP, HTTPS, Direct, Redirect, TProxy
  - V2Ray based: VLESS, VMess, Trojan, Shadowsocks
  - Other protocols: ShadowTLS, Hysteria, Hysteria2, Naive, TUIC
- Supports XTLS protocols
- An advanced interface for routing traffic, incorporating PROXY Protocol, External, and Transparent Proxy, SSL Certificate, and Port
- An advanced interface for inbound and outbound configuration
- Clients’ traffic cap and expiration date
- Displays online clients, inbounds and outbounds with traffic statistics, and system status monitoring
- Subscription service with ability to add external links and subscription
- HTTPS for secure access to the web panel and subscription service (self-provided domain + SSL certificate)
- Dark/Light theme

## Environment Variables

<details>
  <summary>Click for details</summary>

### Usage

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

</details>

## SSL Certificate

<details>
  <summary>Click for details</summary>

### Certbot

```bash
snap install core; snap refresh core
snap install --classic certbot
ln -s /snap/bin/certbot /usr/bin/certbot

certbot certonly --standalone --register-unsafely-without-email --non-interactive --agree-tos -d <Your Domain Name>
```

</details>

## Third-party Projects

Community-made projects built around S-UI. These are not affiliated with or maintained by S-UI — use them at your own discretion:

- [itning/reset-s-ui-traffic](https://github.com/itning/reset-s-ui-traffic) — periodic traffic reset for all users
- [zqh2333/s-ui-traffic-reset](https://github.com/zqh2333/s-ui-traffic-reset) — traffic reset tool

> Building something on top of S-UI (a Telegram bot, monitoring, automation, ...)? Open an issue/PR to get it listed here.

## Stargazers over Time
[![Stargazers over time](https://starchart.cc/zhengxiongzhao/s-ui.svg)](https://starchart.cc/zhengxiongzhao/s-ui)
