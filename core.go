package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Dns struct {
	Servers []string `json:"servers"`
	Tag     string   `json:"tag"`
}

type Log struct {
	Access   string `json:"access"`
	Error    string `json:"error"`
	Loglevel string `json:"warning"`
}

type httpInbound struct {
	Listen   string `json:"listen"`
	Port     int    `json:"port"`
	Tag      string `json:"tag"`
	Protocol string `json:"protocol"`
}

type dokodemoSettings struct {
	Address string `json:"address"`
	Network string `json:"network"`
	Port    int    `json:"port"`
}
type dokodemoInbound struct {
	Listen   string           `json:"listen"`
	Port     int              `json:"port"`
	Tag      string           `json:"tag"`
	Protocol string           `json:"protocol"`
	Settings dokodemoSettings `json:"settings"`
}

type Peers struct {
	Endpoint  string `json:"endpoint"`
	KeepAlive int    `json:"keepAlive"`
	PublicKey string `json:"publicKey"`
}

type Settings struct {
	Address     []string `json:"address"`
	Mtu         int      `json:"mtu"`
	NoKernelTun bool     `json:"noKernelTun"`
	Reserved    []int    `json:"reserved"`
	SecretKey   string   `json:"secretKey"`
	Peers       []Peers  `json:"peers"`
}

type Sockopt struct {
	DialerProxy string `json:"dialerProxy"`
}

type StreamSettings struct {
	Sockopt struct {
		DialerProxy string `json:"dialerProxy"`
	} `json:"sockopt"`
}

type FreedomOutbound struct {
	Protocol string `json:"protocol"`
	Settings any    `json:"settings"`
	Tag      string `json:"tag"`
}

type Noise struct {
	Type   string `json:"type"`
	Packet string `json:"packet"`
	Delay  string `json:"delay"`
}

type FreedomSettings struct {
	Noises *[]Noise `json:"noises"`
}

type WgOutbound struct {
	Protocol       string          `json:"protocol"`
	Settings       Settings        `json:"settings"`
	StreamSettings *StreamSettings `json:"streamSettings,omitempty"`
	Tag            string          `json:"tag"`
}

type RoutingRule struct {
	InboundTag  []string `json:"inboundTag"`
	OutboundTag string   `json:"outboundTag"`
	Type        string   `json:"type"`
}

type Routing struct {
	DomainStrategy string        `json:"domainStrategy"`
	Rules          []RoutingRule `json:"rules"`
}

type XrayConfig struct {
	Remarks   string        `json:"remarks"`
	Log       Log           `json:"log"`
	Dns       Dns           `json:"dns"`
	Inbounds  []httpInbound `json:"inbounds"`
	Outbounds []any         `json:"outbounds"`
	Routing   Routing       `json:"routing"`
}

var xrayConfig = filepath.Join(CORE_DIR, "config.json")

func buildXrayInbound(index int) httpInbound {
	inbound := httpInbound{
		Listen:   "127.0.0.1",
		Port:     1080 + index,
		Protocol: "http",
		Tag:      fmt.Sprintf("http-in-%d", index+1),
	}

	return inbound
}

func buildXrayWgOutbound(index int, endpoint string, isNoise bool) WgOutbound {
	outbound := WgOutbound{
		Protocol: "wireguard",
		Settings: Settings{
			Address: []string{
				"172.16.0.2/32",
				"2606:4700:110:844c:42a:316b:f0a4:c524/128",
			},
			Mtu:         1280,
			NoKernelTun: true,
			Peers: []Peers{
				{
					Endpoint:  endpoint,
					KeepAlive: 5,
					PublicKey: "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo=",
				},
			},
			Reserved: []int{
				120,
				63,
				135,
			},
			SecretKey: "aBLe8/f8yno5xxXZKGLvUwLs6iWLOH5BSZf3AWH7yWk=",
		},
		Tag: fmt.Sprintf("proxy-%d", index+1),
	}

	if isNoise {
		outbound.StreamSettings = &StreamSettings{
			Sockopt: Sockopt{
				DialerProxy: "udp-noise",
			},
		}
	}

	return outbound
}

func buildXrayRoutingRule(index int) RoutingRule {
	return RoutingRule{
		InboundTag: []string{
			fmt.Sprintf("http-in-%d", index+1),
		},
		OutboundTag: fmt.Sprintf("proxy-%d", index+1),
		Type:        "field",
	}
}

func buildXrayConfig(endpoints []string, isNoise bool) XrayConfig {
	config := XrayConfig{
		Remarks: "test",
		Log: Log{
			Access:   "core/log/access.log",
			Error:    "core/log/error.log",
			Loglevel: "warning",
		},
		Dns: Dns{
			Servers: []string{
				"1.1.1.1",
			},
			Tag: "dns",
		},
		Inbounds:  []httpInbound{},
		Outbounds: []any{},
		Routing: Routing{
			DomainStrategy: "AsIs",
			Rules:          []RoutingRule{},
		},
	}

	if isNoise {
		udpNoiseOutbound := FreedomOutbound{
			Protocol: "freedom",
			Settings: FreedomSettings{
				Noises: &[]Noise{
					{
						Delay:  "1-1",
						Packet: "50-100",
						Type:   "rand",
					},
					{
						Delay:  "1-1",
						Packet: "50-100",
						Type:   "rand",
					},
					{
						Delay:  "1-1",
						Packet: "50-100",
						Type:   "rand",
					},
					{
						Delay:  "1-1",
						Packet: "50-100",
						Type:   "rand",
					},
					{
						Delay:  "1-1",
						Packet: "50-100",
						Type:   "rand",
					},
				},
			},
			Tag: "udp-noise",
		}

		config.Outbounds = append(config.Outbounds, udpNoiseOutbound)
	}

	for index, endpoint := range endpoints {
		inbound := buildXrayInbound(index)
		config.Inbounds = append(config.Inbounds, inbound)
		outbound := buildXrayWgOutbound(index, endpoint, isNoise)
		config.Outbounds = append(config.Outbounds, outbound)
		routingRule := buildXrayRoutingRule(index)
		config.Routing.Rules = append(config.Routing.Rules, routingRule)
	}

	return config
}

func createXrayConfig(endpoints []string, isNoise bool) error {
	config := buildXrayConfig(endpoints, isNoise)
	jsonBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON marshal error: %v\n", err)
	}

	file, err := os.Create(xrayConfig)
	if err != nil {
		return fmt.Errorf("Error creating config.json: %v\n", err)
	}
	defer file.Close()

	if _, err := file.Write(jsonBytes); err != nil {
		return fmt.Errorf("Error writing config.json: %v\n", err)
	}

	return nil
}

func runXrayCore() (*exec.Cmd, error) {
	cmd := exec.Command(xrayPath, "-c", xrayConfig)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("Error starting XRay core: %v\n", err)
	}

	fmt.Println("Waiting for XRay core to initialize...")
	time.Sleep(1000 * time.Millisecond)
	return cmd, nil
}
