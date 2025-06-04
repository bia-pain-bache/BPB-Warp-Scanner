package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var httpClient *http.Client

func setDns() {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Second * 3,
				}
				return d.DialContext(ctx, "udp", "8.8.8.8:53")
			},
		},
	}

	httpClient = &http.Client{
		Transport: &http.Transport{
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   3 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
		},
		Timeout: 5 * time.Second,
	}
}

func checkNetworkStats() {
	fmt.Printf("\n%s Determining network quality to adjust scan options, please wait...\n", prompt)
	const (
		testTargetURL     = "http://www.gstatic.com/generate_204"
		initialTestCount  = 100
		goodLatencyMs     = 300
		moderateLatencyMs = 400
		poorLatencyMs     = 500
		acceptableLoss    = 10.0
		highLoss          = 20.0
		moderateJitterMs  = 20.0
		highJitterMs      = 40.0
	)

	var totalLatency int64
	successCount := 0
	var wg sync.WaitGroup
	latencyResults := make(chan int64, initialTestCount)

	for range initialTestCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			resp, err := httpClient.Head(testTargetURL)
			latency := time.Since(start).Milliseconds()
			if err == nil && resp.StatusCode == http.StatusNoContent {
				if resp.Body != nil {
					resp.Body.Close()
				}
				latencyResults <- latency
			} else {
				latencyResults <- -1
			}
		}()
		time.Sleep(150 * time.Millisecond)
	}

	wg.Wait()
	close(latencyResults)

	successfulLatencies := make([]int64, 0, successCount)
	for res := range latencyResults {
		if res >= 0 {
			successCount++
			totalLatency += res
			successfulLatencies = append(successfulLatencies, res)
		}
	}

	if successCount == 0 {
		failMessage("Initial network quality test failed. Could not reach test server.")
		fmt.Printf("%s Using default scan settings.\n", prompt)
		return
	}

	avgLatency := totalLatency / int64(successCount)
	lossRate := float64(initialTestCount-successCount) / float64(initialTestCount) * 100
	var avgJitter float64

	if successCount > 1 {
		var totalJitter int64
		for i := range len(successfulLatencies) - 1 {
			diff := successfulLatencies[i+1] - successfulLatencies[i]
			if diff < 0 {
				diff = -diff
			}
			totalJitter += diff
		}
		avgJitter = float64(totalJitter) / float64(successCount-1)
		fmt.Printf("\n%s Avg Latency: %dms - Jitter: %.1fms - Loss: %.1f%%\n", prompt, avgLatency, avgJitter, lossRate)
	} else {
		fmt.Printf("\n%s Avg Latency: %dms - Jitter not calculated (unsuccessful tests) - Loss: %.1f%%\n", prompt, avgLatency, lossRate)
	}

	if avgLatency >= int64(poorLatencyMs) || lossRate >= highLoss || (successCount > 1 && avgJitter >= highJitterMs) {
		scanConfig.ScanRetries = 7
		successMessage("Network appears slow/unstable.")
	} else if avgLatency >= int64(moderateLatencyMs) || lossRate >= acceptableLoss || (successCount > 1 && avgJitter >= moderateJitterMs) {
		scanConfig.ScanRetries = 5
		successMessage("Network is moderate or some packet loss detected.")
	} else {
		successMessage("Network quality seems good. Using default scan settings.")
	}
}

func scanEndpoints() ([]ScanResult, error) {
	err := createXrayConfig()
	if err != nil {
		return nil, err
	}

	cmd, err := runXrayCore()
	if err != nil {
		log.Print(err)
		return nil, err
	}

	var wg sync.WaitGroup
	results := make(chan ScanResult, len(scanConfig.Endpoints))
	transports := make([]*http.Transport, len(scanConfig.Endpoints))

	for i, endpoint := range scanConfig.Endpoints {
		wg.Add(1)
		go func(endpoint string, portIdx int) {
			defer wg.Done()
			time.Sleep(time.Duration(portIdx*100) * time.Millisecond)
			proxyURL := must(url.Parse(fmt.Sprintf("http://127.0.0.1:%d", 1080+portIdx)))
			transport := &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
			transports[portIdx] = transport

			currentRetries := scanConfig.ScanRetries
			var successCount int
			var totalLatency int64

			var innerWg sync.WaitGroup
			latencies := make(chan int64, currentRetries)

			for t := range currentRetries {
				innerWg.Add(1)
				go func(delay int) {
					defer innerWg.Done()
					time.Sleep(time.Duration(delay) * time.Millisecond)
					client := &http.Client{
						Timeout:   2 * time.Second,
						Transport: transport,
					}

					start := time.Now()
					resp, err := client.Head("http://www.gstatic.com/generate_204")
					latency := time.Since(start).Milliseconds()
					if err == nil && resp.StatusCode == 204 {
						if resp.Body != nil {
							resp.Body.Close()
						}
						latencies <- latency
					} else {
						latencies <- -1
					}
				}(t * 200)
			}
			innerWg.Wait()
			close(latencies)

			for l := range latencies {
				if l >= 0 {
					successCount++
					totalLatency += l
				}
			}

			if successCount == 0 {
				log.Printf("[%d] %s -> %s\n", i+1, fmtStr(endpoint, ORANGE, false), fmtStr("Failed", RED, true))
			} else {
				avgLatency := totalLatency / int64(successCount)
				lossRate := float64(currentRetries-successCount) / float64(currentRetries) * 100
				results <- ScanResult{Endpoint: endpoint, Loss: lossRate, Latency: avgLatency}
				log.Printf("[%d] %s -> %s - %s %.1f %% - %s %d ms\n",
					i+1,
					fmtStr(endpoint, ORANGE, false),
					fmtStr("Success", GREEN, true),
					fmtStr("Loss rate:", "", true),
					lossRate,
					fmtStr("Avg. Latency:", "", true),
					avgLatency,
				)
			}
		}(endpoint, i)
	}
	wg.Wait()
	close(results)

	var allResults []ScanResult
	for r := range results {
		allResults = append(allResults, r)
	}

	if err := cmd.Process.Kill(); err != nil {
		return nil, fmt.Errorf("error killing Xray core: %w", err)
	}

	cmd.Wait()

	return allResults, nil
}
