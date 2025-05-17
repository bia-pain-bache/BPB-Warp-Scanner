package main

import (
	"archive/zip"
	"bufio"
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
)

func generateIPv4(iplist int) error {
	ranges := []string{
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

	for len(uniqueIPs) < iplist {
		prefix := ranges[rand.Intn(len(ranges))]
		ip := fmt.Sprintf("%s%d", prefix, rand.Intn(256))
		if !uniqueIPs[ip] {
			uniqueIPs[ip] = true
			fmt.Fprintln(f, ip)
		}
	}
	return nil
}

func generateIPv6(iplist int) error {
	prefixes := []string{
		"2606:4700:d0::",
		"2606:4700:d1::",
	}

	f, err := os.Create("ip.txt")
	if err != nil {
		return err
	}
	defer f.Close()

	uniqueIPs := make(map[string]bool)
	rand.New(rand.NewSource(time.Now().UnixNano()))

	for len(uniqueIPs) < iplist {
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

func displayEndpoints(count string) error {
	n, err := strconv.Atoi(count)
	if err != nil {
		return fmt.Errorf("invalid count: %v", err)
	}

	file, err := os.Open("result.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Skip header line
	scanner.Scan()

	fmt.Printf("\nTop %d Endpoints:\n", n)
	fmt.Println("----------------")
	displayed := 0
	for scanner.Scan() && displayed < n {
		line := strings.TrimSpace(scanner.Text())
		if parts := strings.Split(line, ","); len(parts) > 0 {
			fmt.Println(parts[0])
			displayed++
		}
	}
	fmt.Println("----------------")
	return scanner.Err()
}

func extractAndRunBinary() error {
	tmpDir, err := os.MkdirTemp("", "bpb-warp-scanner")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := fmt.Sprintf("embed/bin/%s-%s.zip", runtime.GOOS, runtime.GOARCH)
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %v", err)
	}
	defer zipReader.Close()

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

func main() {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		fmt.Println("This program only supports Linux and Windows.")
		return
	}

	os.Create("result.csv")

	fmt.Println("\n1. Normal scan (100 IPs)")
	fmt.Println("2. Deep scan (1000 IPs)")
	fmt.Print("Please select scan mode (1 or 2): ")
	var ipCount int
	var mode string
	fmt.Scanln(&mode)

	for {
		switch mode {
		case "1":
			ipCount = 100
		case "2":
			ipCount = 1000
		default:
			fmt.Println("Invalid choice. Please select 1 or 2.")
			continue
		}
		break
	}

	fmt.Println("\n1. Scan IPv4 endpoints")
	fmt.Println("2. Scan IPv6 endpoints")
	fmt.Print("Please select IP version (1 or 2): ")
	var ipVersion string
	fmt.Scanln(&ipVersion)

	for {
		switch ipVersion {
		case "1":
			generateIPv4(ipCount)
		case "2":
			generateIPv6(ipCount)
		default:
			fmt.Println("Invalid choice. Please select 1 or 2.")
			continue
		}
		break
	}

	fmt.Print("\nHow many Endpoints do you need: ")
	var count string
	fmt.Scanln(&count)

	if err := extractAndRunBinary(); err != nil {
		fmt.Printf("Error running scanner: %v\n", err)
		return
	}

	if err := displayEndpoints(count); err != nil {
		fmt.Printf("Error displaying endpoints: %v\n", err)
	}

	fmt.Println("\nScan completed. You can check result.csv for more details.")
}
