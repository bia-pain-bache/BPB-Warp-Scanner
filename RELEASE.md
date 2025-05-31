# Bug fixes and Improvements

- Revised Wireguard MTU based on IP version.
- Added Custom UDP noise configuration to scanner, the default noise is like BPB Panel default UDP noise:

```text
type: Random
Packet: 50-100
Delay: 1-5 ms
Count: 5
```

However, if this is not working well on your ISP, you can set desired settings in wizard. For more information please read [Xray core UDP noises docs](https://xtls.github.io/config/outbounds/freedom.html#outboundconfigurationobject).
