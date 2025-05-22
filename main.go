package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

const (
	RED      = "1"
	GREEN    = "2"
	ORANGE   = "208"
	BLUE     = "39"
	CORE_DIR = "core"
)

var (
	VERSION  = "dev"
	title    = fmtStr("●", BLUE, true)
	ask      = fmtStr("-", "", true)
	info     = fmtStr("+", "", true)
	warning  = fmtStr("Warning", RED, true)
	xrayPath string
	ipv4Mode bool
	ipv6Mode bool
)

type ScanResult struct {
	Endpoint string
	Loss     float64
	Latency  int64
}

func fmtStr(str string, color string, isBold bool) string {
	style := lipgloss.NewStyle().Bold(isBold)

	if color != "" {
		style = style.Foreground(lipgloss.Color(color))
	}

	return style.Render(str)
}

func renderHeader() {
	fmt.Printf(`
■■■■■■■  ■■■■■■■  ■■■■■■■ 
■■   ■■  ■■   ■■  ■■   ■■
■■■■■■■  ■■■■■■■  ■■■■■■■ 
■■   ■■  ■■       ■■   ■■
■■■■■■■  ■■       ■■■■■■■  %s %s
`,
		fmtStr("Warp Scanner", BLUE, true),
		fmtStr(VERSION, GREEN, false),
	)
}

func generateEndpoints(count int) []string {
	ports := []int{
		500, 854, 859, 864, 878, 880, 890, 891, 894, 903,
		908, 928, 934, 939, 942, 943, 945, 946, 955, 968,
		987, 988, 1002, 1010, 1014, 1018, 1070, 1074, 1180, 1387,
		1701, 1843, 2371, 2408, 2506, 3138, 3476, 3581, 3854, 4177,
		4198, 4233, 4500, 5279, 5956, 7103, 7152, 7156, 7281, 7559, 8319, 8742, 8854, 8886,
	}

	ipv4Prefixes := []string{
		"188.114.96.", "188.114.97.", "188.114.98.", "188.114.99.",
		"162.159.192.", "162.159.193.", "162.159.195.",
	}
	ipv6Prefixes := []string{
		"2606:4700:d0::", "2606:4700:d1::",
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	endpoints := make([]string, 0, count)
	seen := make(map[string]bool)

	ipv4Count, ipv6Count := 0, 0
	if ipv4Mode && ipv6Mode {
		ipv4Count = count / 2
		ipv6Count = count - ipv4Count
	} else if ipv4Mode {
		ipv4Count = count
	} else if ipv6Mode {
		ipv6Count = count
	}

	for len(endpoints) < ipv4Count {
		prefix := ipv4Prefixes[rand.Intn(len(ipv4Prefixes))]
		ip := fmt.Sprintf("%s%d", prefix, rand.Intn(256))
		endpoint := fmt.Sprintf("%s:%d", ip, ports[rand.Intn(len(ports))])
		if !seen[endpoint] {
			seen[endpoint] = true
			endpoints = append(endpoints, endpoint)
		}
	}

	for len(endpoints) < ipv4Count+ipv6Count {
		prefix := ipv6Prefixes[rand.Intn(len(ipv6Prefixes))]
		ip := fmt.Sprintf("[%s%x:%x:%x:%x]", prefix,
			rand.Intn(65536), rand.Intn(65536),
			rand.Intn(65536), rand.Intn(65536))
		endpoint := fmt.Sprintf("%s:%d", ip, ports[rand.Intn(len(ports))])
		if !seen[endpoint] {
			seen[endpoint] = true
			endpoints = append(endpoints, endpoint)
		}
	}

	message := fmt.Sprintf("Generated %d endpoints to test\n", len(endpoints))
	successMessage(message)
	return endpoints
}

func must[T any](v T, _ error) T { return v }

func writeLines(path string, lines []string) error {
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

func renderEndpoints(results []ScanResult) {
	message := fmt.Sprintf("Top %d Endpoints:\n", len(results))
	successMessage(message)

	var tableRows [][]string
	for _, r := range results {
		tableRows = append(tableRows, []string{
			r.Endpoint,
			fmt.Sprintf("%.2f %%", r.Loss),
			fmt.Sprintf("%d ms", r.Latency),
		})
	}

	table := table.New().
		Border(lipgloss.MarkdownBorder()).
		BorderTop(true).
		BorderBottom(true).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color(GREEN))).
		StyleFunc(func(row, col int) lipgloss.Style {
			style := lipgloss.NewStyle().Padding(0, 2).Align(lipgloss.Center)
			if row == table.HeaderRow {
				style = style.Bold(true)
				if col == 0 {
					style = style.Foreground(lipgloss.Color(GREEN))
				} else {
					style = style.Foreground(lipgloss.Color(ORANGE))
				}
			}
			return style
		}).
		Headers("Endpoint", "Loss rate", "Latency").
		Rows(tableRows...)
	fmt.Println(table.Render())
}

func failMessage(message string) {
	errMark := fmtStr("✗", RED, true)
	fmt.Printf("%s %s\n", errMark, message)
}

func successMessage(message string) {
	succMark := fmtStr("✓", GREEN, true)
	fmt.Printf("\n%s %s\n", succMark, message)
}

func scanEndpoints(endpoints []string, isNoise bool) ([]ScanResult, error) {
	err := createXrayConfig(endpoints, isNoise)
	if err != nil {
		return nil, err
	}

	cmd, err := runXrayCore()
	if err != nil {
		log.Print(err)
		return nil, err
	}

	var wg sync.WaitGroup
	results := make(chan ScanResult, len(endpoints))
	transports := make([]*http.Transport, len(endpoints))

	for i, endpoint := range endpoints {
		wg.Add(1)
		go func(endpoint string, portIdx int) {
			defer wg.Done()
			time.Sleep(time.Duration(portIdx*100) * time.Millisecond)
			proxyURL := must(url.Parse(fmt.Sprintf("http://127.0.0.1:%d", 1080+portIdx)))
			transport := &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
			transports[portIdx] = transport

			const tries = 3
			var successCount int
			var totalLatency int64

			var innerWg sync.WaitGroup
			latencies := make(chan int64, tries)

			for t := range tries {
				innerWg.Add(1)
				go func(delay int) {
					defer innerWg.Done()
					time.Sleep(time.Duration(delay) * time.Millisecond)
					client := &http.Client{
						Timeout:   1 * time.Second,
						Transport: transport,
					}

					start := time.Now()
					resp, err := client.Head("http://www.gstatic.com/generate_204")
					latency := time.Since(start).Milliseconds()
					if err == nil && resp.StatusCode == 204 {
						latencies <- latency
						resp.Body.Close()
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
				log.Printf("%s -> %s\n", fmtStr(endpoint, ORANGE, false), fmtStr("Failed", RED, true))
			} else {
				avgLatency := totalLatency / int64(successCount)
				lossRate := float64(tries-successCount) / float64(tries) * 100
				results <- ScanResult{Endpoint: endpoint, Loss: lossRate, Latency: avgLatency}
				log.Printf("%s -> %s - %s %.2f %% - %s %d ms\n",
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
		return nil, fmt.Errorf("Error killing Xray core: %v\n", err)
	}

	cmd.Wait()

	return allResults, nil
}

func init() {
	showVersion := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *showVersion {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	logDir := filepath.Join(CORE_DIR, "log")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		failMessage("Failed to create Xray log directory")
		log.Fatal(err)
	}

	accessLog := filepath.Join(logDir, "access.log")
	errorLog := filepath.Join(logDir, "error.log")
	for _, file := range []string{accessLog, errorLog} {
		file, err := os.Create(file)
		if err != nil {
			failMessage("Failed to create Xray log file")
			log.Fatal(err)
		}
		defer file.Close()
	}

	var binary string
	if runtime.GOOS == "windows" {
		binary = "xray.exe"
	} else {
		binary = "xray"
	}
	xrayPath = filepath.Join(CORE_DIR, binary)

	renderHeader()
}

func checkNum(num string, min int, max int) (bool, int) {
	n, err := strconv.Atoi(num)
	if err != nil {
		return false, 0
	} else if n < min || n > max {
		return false, 0
	} else {
		return true, n
	}

}

func main() {

	fmt.Printf("\n%s Quick scan - 100 endpoints", fmtStr("1.", BLUE, true))
	fmt.Printf("\n%s Normal scan - 1000 endpoints", fmtStr("2.", BLUE, true))
	fmt.Printf("\n%s Deep scan - 10000 endpoints", fmtStr("3.", BLUE, true))
	fmt.Printf("\n%s Custom scan - you choose how many endpoints", fmtStr("4.", BLUE, true))
	var count int
	for {
		fmt.Print("\n- Please select scan mode (1-4): ")
		var mode string
		fmt.Scanln(&mode)
		switch mode {
		case "1":
			count = 100
		case "2":
			count = 1000
		case "3":
			count = 10000
		case "4":
			for {
				var howMany string
				fmt.Print("\n- How many endpoints do you want to scan?: ")
				fmt.Scanln(&howMany)
				isValid, c := checkNum(howMany, 1, 10000)
				if !isValid {
					failMessage("Invalid input. Please enter a numeric value between 1-10000.")
				} else {
					count = c
					break
				}
			}
		default:
			failMessage("Invalid choice. Please select 1 to 4.")
			continue
		}
		break
	}
	fmt.Printf("\n%s Scan IPv4 only", fmtStr("1.", BLUE, true))
	fmt.Printf("\n%s Scan IPv6 only", fmtStr("2.", BLUE, true))
	fmt.Printf("\n%s IPv4 and IPv6", fmtStr("3.", BLUE, true))
	for {
		var ipVersion string
		fmt.Print("\n- Please select IP version (1-3): ")
		fmt.Scanln(&ipVersion)
		switch ipVersion {
		case "1":
			ipv4Mode = true
			ipv6Mode = false
		case "2":
			ipv4Mode = false
			ipv6Mode = true
		case "3":
			ipv4Mode = true
			ipv6Mode = true
		default:
			failMessage("Invalid choice. Please select 1 to 3.")
			continue
		}
		break
	}

	var useNoise bool
	fmt.Printf("\n%s Warp is totally blocked on my ISP", fmtStr("1.", BLUE, true))
	fmt.Printf("\n%s Warp is OK, just need faster endpoints", fmtStr("2.", BLUE, true))
	for {
		var res string
		fmt.Print("\n- Please select your situation (1 or 2): ")
		fmt.Scanln(&res)
		switch res {
		case "1":
			useNoise = true
		case "2":
			useNoise = false
		default:
			failMessage("Invalid choice. Please select 1 or 2.")
			continue
		}
		break
	}

	var outCount int
	for {
		var res string
		fmt.Print("\n- How many Endpoints do you need: ")
		fmt.Scanln(&res)
		isValid, num := checkNum(res, 1, count) 
		if isValid {
			outCount = num
			break
		} else {
			errorMessage := fmt.Sprintf("Invalid input. Please enter a numeric value between 1-%d.", count)
			failMessage(errorMessage)
		}
	}

	endpoints := generateEndpoints(count)
	results, err := scanEndpoints(endpoints, useNoise)
	if err != nil {
		failMessage("Scan failed.")
		log.Fatal(err)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Latency < results[j].Latency
	})

	lines := make([]string, 0, len(results)+1)
	lines = append(lines, "Endpoint,Loss rate,Avg. Latency")
	for _, r := range results {
		lines = append(lines, fmt.Sprintf("%s,%.2f %%,%d ms", r.Endpoint, r.Loss, r.Latency))
	}
	if err := writeLines("result.csv", lines); err != nil {
		fmt.Printf("Error saving working IPs: %v\n", err)
	}

	renderEndpoints(results[:min(outCount, len(results))])
	successMessage("Scan completed.")
	message := fmt.Sprintf("Found %d endpoints. You can check result.csv for more details.\n", len(results))
	successMessage(message)
	fmt.Println("Press any key to exit...")
	fmt.Scanln()
}
