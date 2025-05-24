package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type KeyPair struct {
	PrivateKey string
	PublicKey  string
}

type WarpConfig struct {
	Config struct {
		Interface struct {
			Addresses struct {
				V6 string `json:"v6"`
			} `json:"addresses"`
		} `json:"interface"`
		ClientID string `json:"client_id"`
		Peers    []struct {
			PublicKey string `json:"public_key"`
		} `json:"peers"`
	} `json:"config"`
}

type WarpParams struct {
	IPv6       string
	Reserved   []int
	PublicKey  string
	PrivateKey string
}

func generatePrivateKeyBase64() (string, error) {
	var privateKey [32]byte

	_, err := rand.Read(privateKey[:])
	if err != nil {
		return "", err
	}

	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	return base64.StdEncoding.EncodeToString(privateKey[:]), nil
}

func fetchWarpConfig(privateKey string) (WarpConfig, error) {
	payload := map[string]interface{}{
		"install_id":   "",
		"fcm_token":    "",
		"tos":          time.Now().UTC().Format(time.RFC3339),
		"type":         "Android",
		"model":        "PC",
		"locale":       "en_US",
		"warp_enabled": true,
		"key":          privateKey,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return WarpConfig{}, fmt.Errorf("Error marshaling warp reg payload: %v\n", err)
	}

	apiBaseUrl := "https://api.cloudflareclient.com/v0a4005/reg"
	req, err := http.NewRequest("POST", apiBaseUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return WarpConfig{}, fmt.Errorf("Error registering warp: %v\n", err)
	}
	req.Header.Set("User-Agent", "insomnia/8.6.1")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return WarpConfig{}, fmt.Errorf("Error registering warp: %v\n", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WarpConfig{}, fmt.Errorf("Error reading warp config: %v\n", err)
	}

	var result WarpConfig
	if err := json.Unmarshal(body, &result); err != nil {
		return WarpConfig{}, fmt.Errorf("failed to parse API response: %v", err)
	}

	return result, nil
}

func base64ToDecimal(base64Str string) ([]int, error) {
	decoded, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, fmt.Errorf("Error decoding reserved: %v\n", err)
	}

	decimalArray := make([]int, len(decoded))
	for i, b := range decoded {
		decimalArray[i] = int(b)
	}
	return decimalArray, nil
}

func extractWarpParams(config WarpConfig, privateKey string) (WarpParams, error) {
	reserved, err := base64ToDecimal(config.Config.ClientID)
	if err != nil {
		return WarpParams{}, fmt.Errorf("Error extracting warp account: %v\n", err)
	}

	return WarpParams{
		IPv6:       config.Config.Interface.Addresses.V6 + "/128",
		Reserved:   reserved,
		PublicKey:  config.Config.Peers[0].PublicKey,
		PrivateKey: privateKey,
	}, nil
}

func getWarpParams() (WarpParams, error) {
	PrivateKey, err := generatePrivateKeyBase64()
	if err != nil {
		return WarpParams{}, err
	}

	config, err := fetchWarpConfig(PrivateKey)
	if err != nil {
		return WarpParams{}, err
	}
	successMessage("Registered identical warp config\n")

	warpConfig, err := extractWarpParams(config, PrivateKey)
	if err != nil {
		return WarpParams{}, err
	}

	return warpConfig, nil
}
