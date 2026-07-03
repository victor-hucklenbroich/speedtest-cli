package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type tester struct {
	client *http.Client
	base   string
}

func (t *tester) downURL(bytes int) string {
	return fmt.Sprintf("%s/down?bytes=%d", t.base, bytes)
}

func (t *tester) upURL() string {
	return t.base + "/up"
}

func (t *tester) runLatency(samples int) {
	var lats []time.Duration
	for i := 0; i < samples; i++ {
		l, err := t.latencySample()
		if err != nil {
			continue
		}
		lats = append(lats, l)
	}
	if len(lats) == 0 {
		fmt.Println("Latency: failed (no samples)")
		return
	}

	min, sum := lats[0], time.Duration(0)
	for _, l := range lats {
		if l < min {
			min = l
		}
		sum += l
	}
	avg := sum / time.Duration(len(lats))

	var jitterSum time.Duration
	for i := 1; i < len(lats); i++ {
		d := lats[i] - lats[i-1]
		if d < 0 {
			d = -d
		}
		jitterSum += d
	}
	jitter := time.Duration(0)
	if len(lats) > 1 {
		jitter = jitterSum / time.Duration(len(lats)-1)
	}

	fmt.Printf("Latency: %.2f ms (min %.2f ms)  jitter: %.2f ms  [%d samples]\n",
		ms(avg), ms(min), ms(jitter), len(lats))
}

func (t *tester) latencySample() (time.Duration, error) {
	start := time.Now()
	resp, err := t.client.Get(t.downURL(0))
	if err != nil {
		return 0, err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	elapsed := time.Since(start)

	serverTime := parseServerTiming(resp.Header.Get("Server-Timing"))
	if latency := elapsed - serverTime; latency > 0 {
		return latency, nil
	}
	return elapsed, nil
}

type transferFunc func(size int) (int64, time.Duration, error)

func (t *tester) runTransfer(sizes []int, fn transferFunc) {
	var best float64
	for _, size := range sizes {
		n, elapsed, err := fn(size)
		if err != nil {
			fmt.Printf("  %8s  error: %v\n", formatBytes(size), err)
			continue
		}
		mbps := float64(n) * 8 / 1e6 / elapsed.Seconds()
		fmt.Printf("  %8s  %8.2f Mbps  (%.1fs)\n", formatBytes(size), mbps, elapsed.Seconds())
		if mbps > best {
			best = mbps
		}
		if elapsed > 10*time.Second {
			break
		}
	}
	fmt.Printf("  -> max: %.2f Mbps\n", best)
}

func (t *tester) downloadOnce(size int) (int64, time.Duration, error) {
	start := time.Now()
	resp, err := t.client.Get(t.downURL(size))
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return 0, 0, fmt.Errorf("unexpected status %s", resp.Status)
	}
	n, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return 0, 0, err
	}
	return n, time.Since(start), nil
}

func (t *tester) uploadOnce(size int) (int64, time.Duration, error) {
	payload := make([]byte, size) // zero-filled; the server just consumes and discards
	start := time.Now()
	resp, err := t.client.Post(t.upURL(), "application/octet-stream", bytes.NewReader(payload))
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode/100 != 2 {
		return 0, 0, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return int64(size), time.Since(start), nil
}

func parseServerTiming(h string) time.Duration {
	i := strings.Index(h, "dur=")
	if i < 0 {
		return 0
	}
	s := h[i+len("dur="):]
	if end := strings.IndexAny(s, ";,"); end >= 0 {
		s = s[:end]
	}
	val, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return time.Duration(val * float64(time.Millisecond))
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

func formatBytes(b int) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%d MB", b>>20)
	case b >= 1<<10:
		return fmt.Sprintf("%d KB", b>>10)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
