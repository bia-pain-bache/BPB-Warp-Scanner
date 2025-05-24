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
	Servers       []string `json:"servers"`
	Tag           string   `json:"tag"`
	QueryStrategy string   `json:"queryStrategy"`
}

type Log struct {
	Access   string `json:"access"`
	Error    string `json:"error"`
	Loglevel string `json:"warning"`
	DnsLog   bool   `json:"dnsLog,omitempty"`
}

type httpInbound struct {
	Protocol string `json:"protocol"`
	Listen   string `json:"listen"`
	Port     int    `json:"port"`
	Tag      string `json:"tag"`
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
	Noises *[]Noise `json:"noises,omitempty"`
}

type WgOutbound struct {
	Protocol       string          `json:"protocol"`
	Settings       Settings        `json:"settings"`
	StreamSettings *StreamSettings `json:"streamSettings,omitempty"`
	DomainStrategy string          `json:"domainStrategy"`
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

func buildHttpInbound(index int) httpInbound {
	inbound := httpInbound{
		Listen:   "127.0.0.1",
		Port:     CORE_INIT_PORT + index,
		Protocol: "http",
		Tag:      fmt.Sprintf("http-in-%d", index+1),
	}

	return inbound
}

func buildWgOutbound(index int, endpoint string, isNoise bool, isIPv4 bool, params WarpParams) WgOutbound {
	domainStrategy := "ForceIPv4"
	if !isIPv4 {
		domainStrategy = "ForceIPv6"
	}

	outbound := WgOutbound{
		Protocol: "wireguard",
		Settings: Settings{
			Address: []string{
				"172.16.0.2/32",
				params.IPv6,
			},
			Mtu:         1280,
			NoKernelTun: true,
			Peers: []Peers{
				{
					Endpoint:  endpoint,
					KeepAlive: 5,
					PublicKey: params.PublicKey,
				},
			},
			Reserved:  params.Reserved,
			SecretKey: params.PrivateKey,
		},
		DomainStrategy: domainStrategy,
		Tag:            fmt.Sprintf("proxy-%d", index+1),
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

func buildRoutingRule(index int) RoutingRule {
	return RoutingRule{
		InboundTag: []string{
			fmt.Sprintf("http-in-%d", index+1),
		},
		OutboundTag: fmt.Sprintf("proxy-%d", index+1),
		Type:        "field",
	}
}

func buildConfig(endpoints []string, isNoise bool) (XrayConfig, error) {
	queryStrategy := "UseIP"
	if ipv4Mode && !ipv6Mode {
		queryStrategy = "UseIPv4"
	}
	if ipv6Mode && !ipv4Mode {
		queryStrategy = "UseIPv6"
	}

	config := XrayConfig{
		Remarks: "test",
		Log: Log{
			Access:   "core/log/access.log",
			Error:    "core/log/error.log",
			Loglevel: "warning",
			// DnsLog:   true,
		},
		Dns: Dns{
			Servers: []string{
				"8.8.8.8",
			},
			Tag:           "dns",
			QueryStrategy: queryStrategy,
		},
		Inbounds: []httpInbound{},
		Outbounds: []any{
			FreedomOutbound{
				Protocol: "freedom",
				Settings: FreedomSettings{},
				Tag:      "direct",
			},
		},
		Routing: Routing{
			DomainStrategy: "AsIs",
			Rules: []RoutingRule{
				{
					InboundTag:  []string{"dns"},
					OutboundTag: "direct",
					Type:        "field",
				},
			},
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

	params, err := getWarpParams()
	if err != nil {
		return XrayConfig{}, err
	}

	count := len(endpoints)
	isIPv4 := ipv4Mode
	for index, endpoint := range endpoints {
		inbound := buildHttpInbound(index)
		config.Inbounds = append(config.Inbounds, inbound)

		if ipv4Mode && ipv6Mode && index >= count/2 {
			isIPv4 = false
		}
		outbound := buildWgOutbound(index, endpoint, isNoise, isIPv4, params)
		config.Outbounds = append(config.Outbounds, outbound)

		routingRule := buildRoutingRule(index)
		config.Routing.Rules = append(config.Routing.Rules, routingRule)
	}

	return config, nil
}

func createXrayConfig(endpoints []string, isNoise bool) error {
	config, err := buildConfig(endpoints, isNoise)
	if err != nil {
		return fmt.Errorf("Error building Xray config: %v\n", err)
	}

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
