<h1 align="center">💦 BPB Warp Scanner</h1>

This project is developed to provide a handy Warp endpoints scanner for [BPB Panel](https://github.com/bia-pain-bache/BPB-Worker-Panel) using [Xray-core](https://github.com/XTLS/Xray-core)

## Features

- Tests network quality based on IP version to optimize scan settings
- Registers a new Warp config per scan
- Performs real delay test instead of ping to extract real endpoints
- Ability to adjust output results count
- 3 IP version modes: `IPv4`, `IPv6` and `IPv4 & IPv6`
- Setting quantity of endpoints to scan: `Quick`, `Normal`, `Deep` and `Custom` modes
- Optional mode to scan with or without UDP noise
- Full noise configuration options with `Base64`, `Hex`, `String` and `Random` modes

> [!TIP]
> For most reliable results, Please set noise configuration exactly similar to your BPB Panel v2ray noise.  
> The default noise is like BPB Panel:  
> ```text
> type: Random
> Packet: 50-100
> Delay: 1-5 ms
> Count: 5
> ```

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
