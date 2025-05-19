package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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
	VERSION     = "v1.0.0"
	RED         = "1"
	GREEN       = "2"
	ORANGE      = "208"
	BLUE        = "39"
	CONFIG_PATH = "core/config.json"
)

var (
	title    = fmtStr("●", BLUE, true)
	ask      = fmtStr("-", "", true)
	info     = fmtStr("+", "", true)
	warning  = fmtStr("Warning", RED, true)
	xrayPath string
)

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

type ScanResult struct {
	Endpoint string
	Loss     float64
	Latency  int64
}

func generateEndpoints(count int, ipv4 bool, ipv6 bool) []string {
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
	if ipv4 && ipv6 {
		ipv4Count = count / 2
		ipv6Count = count - ipv4Count
	} else if ipv4 {
		ipv4Count = count
	} else if ipv6 {
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

	return endpoints
}

func must[T any](v T, _ error) T { return v }

func writeLines(path string, lines []string) error {
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

func UpdateXrayConfig(batch []string) error {
	data, err := os.ReadFile(CONFIG_PATH)
	if err != nil {
		return fmt.Errorf("Error reading Xray config: %v\n", err)
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("Error parsing Xray config: %v\n", err)
	}

	outbounds := config["outbounds"].([]any)
	for i := 0; i < len(batch) && i < len(outbounds); i++ {
		ob := outbounds[i].(map[string]any)
		if ob["protocol"] == "wireguard" {
			settings := ob["settings"].(map[string]any)
			peers := settings["peers"].([]any)
			peer := peers[0].(map[string]any)
			peer["endpoint"] = batch[i]
		}
	}
	updatedData, _ := json.MarshalIndent(config, "", "    ")
	if err := os.WriteFile(CONFIG_PATH, updatedData, 0644); err != nil {
		return fmt.Errorf("Error updating Xray config: %v\n", err)
	}

	return nil
}

func runXrayCore() (*exec.Cmd, error) {
	cmd := exec.Command(xrayPath, "-c", CONFIG_PATH)
	// stdout, _ := cmd.StdoutPipe()
	// stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("Error starting XRay core: %v\n", err)
	}
	// Print XRay logs (optional)
	// go func() {
	// 	scanner := bufio.NewScanner(stdout)
	// 	for scanner.Scan() {
	// 		fmt.Printf("[XRay] %s\n", scanner.Text())
	// 	}
	// }()
	// go func() {
	// 	scanner := bufio.NewScanner(stderr)
	// 	for scanner.Scan() {
	// 		fmt.Printf("[XRay-ERR] %s\n", scanner.Text())
	// 	}
	// }()

	fmt.Println("Waiting for XRay core to initialize...")
	time.Sleep(200 * time.Millisecond)
	return cmd, nil
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

func scanEndpoints(endpoints []string) ([]ScanResult, error) {
	fmt.Printf("Generated %d endpoints to test\n", len(endpoints))
	var allResults []ScanResult

	batchSize := 100
	for batchStart := 0; batchStart < len(endpoints); batchStart += batchSize {
		batchEnd := min(batchStart+batchSize, len(endpoints))
		batch := endpoints[batchStart:batchEnd]

		err := UpdateXrayConfig(batch)
		if err != nil {
			return nil, err
		}

		cmd, err := runXrayCore()
		if err != nil {
			log.Print(err)
			continue
		}

		var wg sync.WaitGroup
		results := make(chan ScanResult, len(batch))

		for i, endpoint := range batch {
			wg.Add(1)
			go func(endpoint string, portIdx int) {
				defer wg.Done()
				time.Sleep(time.Duration(portIdx*100) * time.Millisecond)
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
							Timeout: 500 * time.Millisecond,
							Transport: &http.Transport{
								Proxy: http.ProxyURL(must(url.Parse(fmt.Sprintf("http://127.0.0.1:%d", 10808+portIdx)))),
							},
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
					}(t * 100)
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

		for r := range results {
			allResults = append(allResults, r)
		}

		if err := cmd.Process.Kill(); err != nil {
			return nil, fmt.Errorf("Error killing Xray core: %v\n", err)
		}

		cmd.Wait()
	}

	return allResults, nil
}

func init() {
	if runtime.GOOS == "windows" {
		xrayPath = "core/xray.exe"
	} else {
		xrayPath = "core/xray"
	}

	renderHeader()
}

func main() {

	fmt.Printf("\n%s Quick scan - 100 endpoints", fmtStr("1.", BLUE, true))
	fmt.Printf("\n%s Normal scan - 1000 endpoints", fmtStr("2.", BLUE, true))
	fmt.Printf("\n%s Deep scan - 10000 endpoints", fmtStr("3.", BLUE, true))
	fmt.Print("\n- Please select scan mode (1-3): ")
	var count int
	var mode string
	fmt.Scanln(&mode)

	for {
		switch mode {
		case "1":
			count = 100
		case "2":
			count = 1000
		case "3":
			count = 10000
		default:
			failMessage("Invalid choice. Please select 1 to 3.")
			continue
		}
		break
	}

	var ipVersion string
	fmt.Printf("\n%s Scan IPv4 only", fmtStr("1.", BLUE, true))
	fmt.Printf("\n%s Scan IPv6 only", fmtStr("2.", BLUE, true))
	fmt.Printf("\n%s IPv4 and IPv6", fmtStr("3.", BLUE, true))
	fmt.Print("\n- Please select IP version (1-3): ")
	fmt.Scanln(&ipVersion)
	var ipv4, ipv6 bool
	for {
		switch ipVersion {
		case "1":
			ipv4 = true
			ipv6 = false
		case "2":
			ipv4 = false
			ipv6 = true
		case "3":
			ipv4 = true
			ipv6 = true
		default:
			failMessage("Invalid choice. Please select 1 to 3.")
			continue
		}
		break
	}

	endpoints := generateEndpoints(count, ipv4, ipv6)

	var outCount int
	for {
		var res string
		fmt.Print("\n- How many Endpoints do you need: ")
		fmt.Scanln(&res)
		num, err := strconv.Atoi(res)
		if err != nil {
			failMessage("Invalid input. Please enter a number.")
			continue
		}
		outCount = num
		break
	}

	results, err := scanEndpoints(endpoints)
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
