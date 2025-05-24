<h1 align="center">💦 BPB Warp Scanner</h1>

This project is developed to provide a handy Warp endpoints scanner for [BPB Panel](https://github.com/bia-pain-bache/BPB-Worker-Panel) using [Xray-core](https://github.com/XTLS/Xray-core)

## Features

- Performs real delay test instead of ping to extract real endpoints
- Ability to adjust output results count
- 3 IP version modes: `IPv4`, `IPv6` and `IPv4 & IPv6`
- Setting quantity of endpoints to scan: `Quick`, `Normal` and `Deep` modes
- Optional mode to scan with or without UDP noise

## 💡 How to use

> [!IMPORTANT]
> Please disconnect your VPN before scanning.
>
> In windwos you should totally exit v2rayN from taskbar, clearing system proxy is not enough.

### Windows - Darwin

Based on your operating system architecture, [download the ZIP file](https://github.com/bia-pain-bache/BPB-Warp-Scanner/releases/latest), unzip it, and run the `BPB-Warp-Scanner`.

### Android (Termux) - Linux

> [!WARNING]
> You should install Termux from [Github](https://github.com/termux/termux-app/releases/latest), Google play version has bugs.

Android users who have Termux installed on their device and Linux users can use this bash:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/bia-pain-bache/BPB-Warp-Scanner/main/install.sh)
```
