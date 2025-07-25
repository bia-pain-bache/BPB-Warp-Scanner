package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
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

type ScanConfig struct {
	EndpointCount        int
	Ipv4Mode             bool
	Ipv6Mode             bool
	IPv4Retries          int
	IPv6Retries          int
	RetryStaggeringMs    int
	EndpointStaggeringMs int
	UseNoise             bool
	UdpNoise             Noise
	Endpoints            []string
	OutputCount          int
}

var (
	VERSION  = "dev"
	prompt   = fmtStr("●", GREEN, true)
	errMark  = fmtStr("✗", RED, true)
	succMark = fmtStr("✓", GREEN, true)
	// ask      = fmtStr("-", "", true)
	// info     = fmtStr("+", "", true)
	// warning  = fmtStr("Warning", RED, true)
	xrayPath string
)

var scanConfig = ScanConfig{
	EndpointCount: 100,
	Ipv4Mode:      true,
	Ipv6Mode:      false,
	UseNoise:      true,
	UdpNoise: Noise{
		Type:   "rand",
		Packet: "50-100",
		Delay:  "1-5",
		Count:  5,
	},
	IPv4Retries:          3,
	IPv6Retries:          3,
	RetryStaggeringMs:    200,
	EndpointStaggeringMs: 100,
}

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

func generateEndpoints() {
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
		"8.34.146.", "8.39.214.", "8.39.204.", "8.6.112.", "8.35.211.", "8.39.125.", "8.47.69.",
	}
	ipv6Prefixes := []string{
		"2606:4700:d0::", "2606:4700:d1::",
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	endpoints := make([]string, 0, scanConfig.EndpointCount)
	seen := make(map[string]bool)

	ipv4Count, ipv6Count := 0, 0
	if scanConfig.Ipv4Mode && scanConfig.Ipv6Mode {
		ipv4Count = scanConfig.EndpointCount / 2
		ipv6Count = scanConfig.EndpointCount - ipv4Count
	} else if scanConfig.Ipv4Mode {
		ipv4Count = scanConfig.EndpointCount
	} else if scanConfig.Ipv6Mode {
		ipv6Count = scanConfig.EndpointCount
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

	message := fmt.Sprintf("Generated %d endpoints to test", len(endpoints))
	successMessage(message)
	scanConfig.Endpoints = endpoints
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
			fmt.Sprintf("%.1f %%", r.Loss),
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
	fmt.Printf("%s %s\n", errMark, message)
}

func successMessage(message string) {
	fmt.Printf("\n%s %s\n", succMark, message)
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

	if _, err := os.Stat(xrayPath); err != nil {
		failMessage("Xray core not found.")
		log.Fatal(err)
	}

	err := os.Chmod(xrayPath, 0755)
	if err != nil {
		failMessage("Failed to set Xray core permissions.")
		log.Fatal(err)
	}

	path := os.Getenv("PATH")
	if runtime.GOOS == "android" || strings.Contains(path, "com.termux") {
		prefix := os.Getenv("PREFIX")
		certPath := filepath.Join(prefix, "etc/tls/cert.pem")
		if err := os.Setenv("SSL_CERT_FILE", certPath); err != nil {
			failMessage("Failed to set Termux cert file.")
			log.Fatalln(err)
		}
	}

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

func isValidHex(value string) bool {
	matched, err := regexp.MatchString(`^[0-9a-fA-F]+$`, value)
	if err != nil {
		return false
	}

	return len(value) > 0 && matched
}

func isValidBase64(value string) bool {
	if len(value) == 0 {
		return false
	}

	_, err := base64.StdEncoding.DecodeString(value)
	return err == nil
}

func isValidRange(value string) bool {
	if value == "" {
		return false
	}

	regex := `^(?:[1-9][0-9]*|[1-9][0-9]*-[1-9][0-9]*)$`
	matched, err := regexp.MatchString(regex, value)
	if err != nil {
		return false
	}

	split := strings.Split(value, "-")
	if len(split) == 2 {
		min, _ := strconv.Atoi(split[0])
		max, _ := strconv.Atoi(split[1])
		return max >= min
	}

	return matched
}

func main() {
	fmt.Printf("\n%s Quick scan - 100 endpoints", fmtStr("1.", BLUE, true))
	fmt.Printf("\n%s Normal scan - 1000 endpoints", fmtStr("2.", BLUE, true))
	fmt.Printf("\n%s Deep scan - 10000 endpoints", fmtStr("3.", BLUE, true))
	fmt.Printf("\n%s Custom scan - you choose how many endpoints", fmtStr("4.", BLUE, true))
	for {
		fmt.Printf("\n\n%s Please select scan mode (1-4): ", prompt)
		var mode string
		fmt.Scanln(&mode)
		switch mode {
		case "1":
		case "2":
			scanConfig.EndpointCount = 1000
		case "3":
			scanConfig.EndpointCount = 10000
		case "4":
			for {
				var howMany string
				fmt.Printf("\n\n%s Please enter your desired endpoints count: ", prompt)
				fmt.Scanln(&howMany)
				isValid, c := checkNum(howMany, 1, 10000)
				if !isValid {
					failMessage("Invalid input. Please enter a numeric value between 1-10000.")
				} else {
					scanConfig.EndpointCount = c
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
		fmt.Printf("\n\n%s Please select IP version (1-3): ", prompt)
		fmt.Scanln(&ipVersion)
		switch ipVersion {
		case "1":
		case "2":
			scanConfig.Ipv4Mode = false
			scanConfig.Ipv6Mode = true
		case "3":
			scanConfig.Ipv6Mode = true
		default:
			failMessage("Invalid choice. Please select 1 to 3.")
			continue
		}
		break
	}

	fmt.Printf("\n%s Warp is totally blocked on my ISP", fmtStr("1.", BLUE, true))
	fmt.Printf("\n%s Warp is OK, just need faster endpoints", fmtStr("2.", BLUE, true))
	for {
		var res string
		fmt.Printf("\n\n%s Please select your situation (1 or 2): ", prompt)
		fmt.Scanln(&res)
		switch res {
		case "1":
		case "2":
			scanConfig.UseNoise = false
		default:
			failMessage("Invalid choice. Please select 1 or 2.")
			continue
		}
		break
	}

	fmt.Printf("\n%s Use default noise", fmtStr("1.", BLUE, true))
	fmt.Printf("\n%s Setup custom noise", fmtStr("2.", BLUE, true))
	for {
		var res string
		fmt.Printf("\n\n%s Please select (1 or 2): ", prompt)
		fmt.Scanln(&res)
		switch res {
		case "1":
		case "2":
			fmt.Printf("\n%s Base64", fmtStr("1.", BLUE, true))
			fmt.Printf("\n%s Hex", fmtStr("2.", BLUE, true))
			fmt.Printf("\n%s String", fmtStr("3.", BLUE, true))
			fmt.Printf("\n%s Random", fmtStr("4.", BLUE, true))
			var noiseType, packet, delay, count string
			for {
				var res string
				fmt.Printf("\n\n%s Please select UDP noise type (1-4): ", prompt)
				fmt.Scanln(&res)
				switch res {
				case "1":
					noiseType = "base64"
				case "2":
					noiseType = "hex"
				case "3":
					noiseType = "str"
				case "4":
					noiseType = "rand"
				default:
					failMessage("Invalid choice. Please select 1-4.")
					continue
				}
				break
			}

			for {
				fmt.Printf("\n%s Please enter a %s packet: ", prompt, fmtStr(noiseType, GREEN, true))
				fmt.Scanln(&packet)
				switch noiseType {
				case "base64":
					if !isValidBase64(packet) {
						msg := fmt.Sprintf("Invalid packet for Base64 type, please enter a valid Base64 value like %s.", fmtStr("aGVsbG8gd29ybGQ=", GREEN, true))
						failMessage(msg)
						continue
					}
				case "hex":
					if !isValidHex(packet) {
						msg := fmt.Sprintf("Invalid packet for Hex type, please enter a valid Hex value like %s.", fmtStr("68656c6c6f20776f726c64", GREEN, true))
						failMessage(msg)
						continue
					}
				case "rand":
					if !isValidRange(packet) {
						msg := fmt.Sprintf("Invalid packet for Random type, please enter packet length, it can be a fixed number or an interval like %s.", fmtStr("50-100", GREEN, true))
						failMessage(msg)
						continue
					}
				}
				break
			}

			for {
				fmt.Printf("\n%s Please enter noise delay in miliseconds, it can be a fixed number or an interval like %s: ", prompt, fmtStr("1-5", GREEN, true))
				fmt.Scanln(&delay)
				if !isValidRange(delay) {
					failMessage("Invalid delay value, please try again.")
					continue
				}
				break
			}

			for {
				fmt.Printf("\n%s Please enter number of noise packets (up to 50): ", prompt)
				fmt.Scanln(&count)
				isValid, noiseCount := checkNum(count, 1, 50)
				if !isValid {
					failMessage("Invalid value. Please enter a numeric value between 1 and 50.")
					continue
				}
				scanConfig.UdpNoise = Noise{
					Type:   noiseType,
					Packet: packet,
					Delay:  delay,
					Count:  noiseCount,
				}
				break
			}

		default:
			failMessage("Invalid choice. Please select 1 or 2.")
			continue
		}
		break
	}

	for {
		var res string
		fmt.Printf("\n%s How many Endpoints do you need: ", prompt)
		fmt.Scanln(&res)
		isValid, num := checkNum(res, 1, scanConfig.EndpointCount)
		if isValid {
			scanConfig.OutputCount = num
			break
		} else {
			errorMessage := fmt.Sprintf("Invalid input. Please enter a numeric value between 1-%d.", scanConfig.EndpointCount)
			failMessage(errorMessage)
		}
	}

	if scanConfig.Ipv4Mode {
		checkNetworkStats(false)
	}
	if scanConfig.Ipv6Mode {
		checkNetworkStats(true)
	}

	generateEndpoints()

	results, err := scanEndpoints()
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

	renderEndpoints(results[:min(scanConfig.OutputCount, len(results))])
	successMessage("Scan completed.")
	message := fmt.Sprintf("Found %d endpoints. You can check result.csv for more details.\n", len(results))
	successMessage(message)
	fmt.Printf("%s Press any key to exit...", prompt)
	fmt.Scanln()
}
