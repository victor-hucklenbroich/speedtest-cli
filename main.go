package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"speedtest/internal/cli"
)

//go:generate go run ./internal/gendocs

func main() {
	cfg, err := cli.ParseConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "speedtest: %v\n", err)
		os.Exit(2)
	}
	if cfg.ShowVersion {
		fmt.Printf("speedtest %s (%s/%s)\n", cli.Version, runtime.GOOS, runtime.GOARCH)
		return
	}

	t := &tester{
		client: &http.Client{Timeout: 60 * time.Second},
		base:   cfg.ServerURL,
	}

	fmt.Println("Speed test")
	fmt.Println("==========")
	fmt.Printf("Server: %s\n\n", t.base)

	t.runLatency(20)

	fmt.Println("\nDownload:")
	t.runTransfer([]int{
		1 << 20,  // 1 MB
		10 << 20, // 10 MB
		25 << 20, // 25 MB
		50 << 20, // 50 MB
		90 << 20, // 90 MB
	}, t.downloadOnce)

	fmt.Println("\nUpload:")
	t.runTransfer([]int{
		1 << 20,  // 1 MB
		5 << 20,  // 5 MB
		10 << 20, // 10 MB
		25 << 20, // 25 MB
		50 << 20, // 50 MB
	}, t.uploadOnce)
}
