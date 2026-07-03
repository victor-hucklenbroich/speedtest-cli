package main

import (
	"fmt"
	"time"

	"speedtest/internal/measure"
)

func runPlain(events <-chan measure.Event, server string) {
	fmt.Println("Speed test")
	fmt.Println("==========")
	fmt.Printf("Server: %s\n\n", server)

	printed := map[measure.Phase]bool{}
	for ev := range events {
		switch e := ev.(type) {
		case measure.LatencyResult:
			if e.Samples == 0 {
				fmt.Println("Latency: failed (no samples)")
			} else {
				fmt.Printf("Latency: %.2f ms (min %.2f ms)  jitter: %.2f ms  [%d samples]\n",
					ms(e.Avg), ms(e.Min), ms(e.Jitter), e.Samples)
			}
		case measure.TierStarted:
			if !printed[e.Phase] {
				printed[e.Phase] = true
				name := "Download"
				if e.Phase == measure.Upload {
					name = "Upload"
				}
				fmt.Printf("\n%s:\n", name)
			}
		case measure.TierResult:
			if e.Err != nil {
				fmt.Printf("  %8s  error: %v\n", measure.FormatBytes(e.Size), e.Err)
			} else {
				fmt.Printf("  %8s  %8.2f Mbps  (%.1fs)\n", measure.FormatBytes(e.Size), e.Mbps, e.Elapsed.Seconds())
			}
		case measure.PhaseResult:
			fmt.Printf("  -> max: %.2f Mbps\n", e.MaxMbps)
		case measure.Done:
			return
		}
	}
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
