package measure

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type Phase int

const (
	Latency Phase = iota
	Download
	Upload
)

type Config struct {
	BaseURL        string
	Client         *http.Client
	LatencySamples int
	DownloadSizes  []int
	UploadSizes    []int
}

type (
	Event any

	LatencyProgress struct {
		Done, Total int
		Min, Avg    time.Duration
	}
	LatencyResult struct {
		Min, Avg, Jitter time.Duration
		Samples          int
	}
	TierStarted struct {
		Phase Phase
		Size  int
	}
	TierProgress struct {
		Phase   Phase
		Size    int
		Bytes   int64
		Mbps    float64
		Elapsed time.Duration
	}
	TierResult struct {
		Phase   Phase
		Size    int
		Mbps    float64
		Elapsed time.Duration
		Err     error
	}
	PhaseResult struct {
		Phase   Phase
		MaxMbps float64
	}
	Done struct{}
)

func Run(ctx context.Context, cfg Config, events chan<- Event) {
	r := &runner{cfg: cfg, events: events, ctx: ctx}
	if cfg.LatencySamples > 0 {
		r.latency()
	}
	if len(cfg.DownloadSizes) > 0 {
		r.phase(Download, cfg.DownloadSizes, r.download)
	}
	if len(cfg.UploadSizes) > 0 {
		r.phase(Upload, cfg.UploadSizes, r.upload)
	}
	r.emit(Done{})
}

type runner struct {
	cfg    Config
	events chan<- Event
	ctx    context.Context
}

func (r *runner) emit(ev Event) {
	select {
	case r.events <- ev:
	case <-r.ctx.Done():
	}
}

func (r *runner) downURL(bytes int) string {
	return fmt.Sprintf("%s/down?bytes=%d", r.cfg.BaseURL, bytes)
}

func (r *runner) upURL() string {
	return r.cfg.BaseURL + "/up"
}

func (r *runner) latency() {
	var lats []time.Duration
	var min, sum time.Duration
	total := r.cfg.LatencySamples
	for i := 0; i < total && r.ctx.Err() == nil; i++ {
		l, err := r.latencySample()
		if err != nil {
			continue
		}
		lats = append(lats, l)
		if len(lats) == 1 || l < min {
			min = l
		}
		sum += l
		r.emit(LatencyProgress{Done: i + 1, Total: total, Min: min, Avg: sum / time.Duration(len(lats))})
	}
	if len(lats) == 0 {
		r.emit(LatencyResult{})
		return
	}

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
	r.emit(LatencyResult{Min: min, Avg: sum / time.Duration(len(lats)), Jitter: jitter, Samples: len(lats)})
}

func (r *runner) latencySample() (time.Duration, error) {
	req, err := http.NewRequestWithContext(r.ctx, http.MethodGet, r.downURL(0), nil)
	if err != nil {
		return 0, err
	}
	start := time.Now()
	resp, err := r.cfg.Client.Do(req)
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

func (r *runner) phase(p Phase, sizes []int, fn func(size int, count *atomic.Int64) (int64, error)) {
	var best float64
	for _, size := range sizes {
		if r.ctx.Err() != nil {
			break
		}
		r.emit(TierStarted{Phase: p, Size: size})

		var count atomic.Int64
		start := time.Now()
		stop := make(chan struct{})
		go r.trackProgress(p, size, &count, start, stop)
		n, err := fn(size, &count)
		close(stop)
		elapsed := time.Since(start)

		if err != nil {
			r.emit(TierResult{Phase: p, Size: size, Elapsed: elapsed, Err: err})
			continue
		}
		mbps := float64(n) * 8 / 1e6 / elapsed.Seconds()
		r.emit(TierResult{Phase: p, Size: size, Mbps: mbps, Elapsed: elapsed})
		if mbps > best {
			best = mbps
		}
		if elapsed > 10*time.Second {
			break
		}
	}
	r.emit(PhaseResult{Phase: p, MaxMbps: best})
}

func (r *runner) trackProgress(p Phase, size int, count *atomic.Int64, start time.Time, stop <-chan struct{}) {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	type sample struct {
		t time.Time
		b int64
	}
	window := []sample{{start, 0}}
	for {
		select {
		case <-stop:
			return
		case <-r.ctx.Done():
			return
		case now := <-tick.C:
			b := count.Load()
			window = append(window, sample{now, b})
			if len(window) > 7 {
				window = window[1:]
			}
			oldest := window[0]
			dt := now.Sub(oldest.t).Seconds()
			if dt <= 0 {
				continue
			}
			mbps := float64(b-oldest.b) * 8 / 1e6 / dt
			r.emit(TierProgress{Phase: p, Size: size, Bytes: b, Mbps: mbps, Elapsed: now.Sub(start)})
		}
	}
}

func (r *runner) download(size int, count *atomic.Int64) (int64, error) {
	req, err := http.NewRequestWithContext(r.ctx, http.MethodGet, r.downURL(size), nil)
	if err != nil {
		return 0, err
	}
	resp, err := r.cfg.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return 0, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return io.Copy(countWriter{count}, resp.Body)
}

func (r *runner) upload(size int, count *atomic.Int64) (int64, error) {
	req, err := http.NewRequestWithContext(r.ctx, http.MethodPost, r.upURL(), &countReader{r: &zeroReader{n: int64(size)}, n: count})
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = int64(size)
	resp, err := r.cfg.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode/100 != 2 {
		return 0, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return int64(size), nil
}

type countWriter struct{ n *atomic.Int64 }

func (w countWriter) Write(p []byte) (int, error) {
	w.n.Add(int64(len(p)))
	return len(p), nil
}

type countReader struct {
	r io.Reader
	n *atomic.Int64
}

func (c *countReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n.Add(int64(n))
	return n, err
}

type zeroReader struct{ n int64 }

func (z *zeroReader) Read(p []byte) (int, error) {
	if z.n <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > z.n {
		p = p[:z.n]
	}
	for i := range p {
		p[i] = 0
	}
	z.n -= int64(len(p))
	return len(p), nil
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

func FormatBytes(b int) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%d MB", b>>20)
	case b >= 1<<10:
		return fmt.Sprintf("%d KB", b>>10)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
