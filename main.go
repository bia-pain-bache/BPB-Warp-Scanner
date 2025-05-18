package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

const (
	VERSION = "v1.0.0"
	RED     = "1"
	GREEN   = "2"
	ORANGE  = "208"
	BLUE    = "39"
)

var (
	title   = fmtStr("●", BLUE, true)
	ask     = fmtStr("-", "", true)
	info    = fmtStr("+", "", true)
	warning = fmtStr("Warning", RED, true)
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

func generateIPv4(ipCount int) error {
	prefixes := []string{
		"188.114.96.",
		"188.114.97.",
		"188.114.98.",
		"188.114.99.",
		"162.159.192.",
		"162.159.193.",
		"162.159.195.",
	}

	f, err := os.Create("ip.txt")
	if err != nil {
		return err
	}
	defer f.Close()

	uniqueIPs := make(map[string]bool)
	rand.New(rand.NewSource(time.Now().UnixNano()))
	var n = ipCount
	if ipCount == -1 {
		n = len(prefixes) * 256
	}

	for len(uniqueIPs) < n {
		prefix := prefixes[rand.Intn(len(prefixes))]
		ip := fmt.Sprintf("%s%d", prefix, rand.Intn(256))
		if !uniqueIPs[ip] {
			uniqueIPs[ip] = true
			fmt.Fprintln(f, ip)
		}
	}

	return nil
}

func generateIPv6(ipCount int, isUpdate bool) error {
	prefixes := []string{
		"2606:4700:d0::",
		"2606:4700:d1::",
	}

	var f *os.File
	var err error
	if isUpdate {
		f, err = os.OpenFile("ip.txt", os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
	} else {
		f, err = os.Create("ip.txt")
		if err != nil {
			return err
		}
		defer f.Close()
	}

	uniqueIPs := make(map[string]bool)
	rand.New(rand.NewSource(time.Now().UnixNano()))
	var n = ipCount
	if ipCount == -1 {
		n = len(prefixes) * 65536
		fmt.Printf("\n%s IPv6 range is too big to scan all, we consider %s IPs only.", fmtStr("INFO:", BLUE, true), fmtStr(strconv.Itoa(n), ORANGE, true))
	}

	for len(uniqueIPs) < n {
		prefix := prefixes[rand.Intn(len(prefixes))]
		ip := fmt.Sprintf("[%s%x:%x:%x:%x]", prefix,
			rand.Intn(65536), rand.Intn(65536),
			rand.Intn(65536), rand.Intn(65536))
		if !uniqueIPs[ip] {
			uniqueIPs[ip] = true
			fmt.Fprintln(f, ip)
		}
	}

	return nil
}

func displayAndCopyEndpoints(count int) error {
	file, err := os.Open("result.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Skip header line
	scanner.Scan()

	message := fmt.Sprintf("Top %d Endpoints:\n", count)
	successMessage(message)

	var results [][]string
	var endpoints = ""
	displayed := 0
	for scanner.Scan() && displayed < count {
		line := strings.TrimSpace(scanner.Text())
		if parts := strings.Split(line, ","); len(parts) > 0 {
			results = append(results, []string{parts[0], parts[1], parts[2]})
			endpoints += fmt.Sprintf("%s\n", parts[0])
			displayed++
		}
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
		Rows(results...)
	fmt.Println(table.Render())

	if err := clipboard.WriteAll(endpoints); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %v", err)
	}
	return scanner.Err()
}

func extractAndRunBinary() error {
	tmpDir, err := os.MkdirTemp("", "bpb-warp-scanner")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := fmt.Sprintf("embed/%s-%s.zip", runtime.GOOS, runtime.GOARCH)
	data, err := binary.ReadFile(archivePath)
	if err != nil {
		return fmt.Errorf("failed to read embedded zip: %v", err)
	}

	readerAt := bytes.NewReader(data)
	zipReader, err := zip.NewReader(readerAt, int64(len(data)))
	if err != nil {
		panic(fmt.Errorf("failed to open zip: %v", err))
	}

	if len(zipReader.File) == 0 {
		return fmt.Errorf("empty zip archive")
	}

	tmpBin := filepath.Join(tmpDir, zipReader.File[0].Name)
	binFile, err := os.OpenFile(tmpBin, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("failed to create temp binary: %v", err)
	}

	rc, err := zipReader.File[0].Open()
	if err != nil {
		binFile.Close()
		return fmt.Errorf("failed to open file in zip: %v", err)
	}

	_, err = io.Copy(binFile, rc)
	rc.Close()
	binFile.Close()
	if err != nil {
		return fmt.Errorf("failed to extract binary: %v", err)
	}

	cmd := exec.Command(tmpBin)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func failMessage(message string) {
	errMark := fmtStr("✗", RED, true)
	fmt.Printf("%s %s\n", errMark, message)
}

func successMessage(message string) {
	succMark := fmtStr("✓", GREEN, true)
	fmt.Printf("\n%s %s\n", succMark, message)
}

func init() {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		fmt.Println("This program only supports Linux and Windows.")
		return
	}

	renderHeader()
	os.Create("result.csv")
}

func main() {

	fmt.Printf("\n%s Normal scan (100 IPs)", fmtStr("1.", BLUE, true))
	fmt.Printf("\n%s Deep scan (1000 IPs)", fmtStr("2.", BLUE, true))
	fmt.Printf("\n%s All available IPs", fmtStr("3.", BLUE, true))
	fmt.Print("\n- Please select scan mode (1 or 2): ")
	var ipCount int
	var mode string
	fmt.Scanln(&mode)

	for {
		switch mode {
		case "1":
			ipCount = 100
		case "2":
			ipCount = 1000
		case "3":
			ipCount = -1
		default:
			failMessage("Invalid choice. Please select 1 or 2.")
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
	for {
		switch ipVersion {
		case "1":
			if err := generateIPv4(ipCount); err != nil {
				message := fmt.Sprintf("Error generating IPv4 list: %v", err)
				failMessage(message)
				return
			}
		case "2":
			if err := generateIPv6(ipCount, false); err != nil {
				message := fmt.Sprintf("Error generating IPv6 list: %v", err)
				failMessage(message)
				return
			}
		case "3":
			if err := generateIPv4(ipCount); err != nil {
				message := fmt.Sprintf("Error generating IPv4 list: %v", err)
				failMessage(message)
				return
			}

			if err := generateIPv6(ipCount, true); err != nil {
				message := fmt.Sprintf("Error appending IPv6 list: %v", err)
				failMessage(message)
				return
			}
		default:
			failMessage("Invalid choice. Please select 1 to 3.")
			continue
		}
		break
	}

	var count int
	for {
		var res string
		fmt.Print("\n- How many Endpoints do you need: ")
		fmt.Scanln(&res)
		num, err := strconv.Atoi(res)
		if err != nil {
			failMessage("Invalid input. Please enter a number.")
			continue
		}
		count = num
		break
	}

	if err := extractAndRunBinary(); err != nil {
		message := fmt.Sprintf("Error running scanner: %v", err)
		failMessage(message)
		return
	}

	if err := displayAndCopyEndpoints(count); err != nil {
		message := fmt.Sprintf("Error displaying endpoints: %v", err)
		failMessage(message)
		return
	}

	successMessage("Scan completed and Endpoints are copied to clipboard. You can check result.csv for more details.\n")
}
