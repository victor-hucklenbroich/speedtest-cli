package constants

import "time"

// Version is a var release builds can inject it
var Version = "0.1.1"

// CLI and config file
const (
	DefaultServerURL = "https://speedtest-worker.speedtest-cli.workers.dev"

	DefaultTimeoutMinutes = 1

	MaxSize = 1 << 40
)

// Measurement
const (
	LatencySamples = 20

	EventBuffer = 64

	EscalationCutoff = 10 * time.Second

	ProgressInterval = 100 * time.Millisecond
	ProgressWindow   = 7
)

var (
	DownloadLadder = []int{
		10 << 20,
		25 << 20,
		50 << 20,
		90 << 20,
	}
	UploadLadder = []int{
		5 << 20,
		10 << 20,
		25 << 20,
		50 << 20,
	}
)

// TUI
const (
	FrameInterval = 50 * time.Millisecond

	DefaultWidth  = 80
	SparkHistory  = 240
	SparkMaxWidth = 48
	BarWidth      = 24

	AnimationEase = 0.2
	AnimationSnap = 0.05
)

// gendocs
const (
	ReadmePath       = "README.md"
	UsageBeginMarker = "<!-- BEGIN USAGE -->"
	UsageEndMarker   = "<!-- END USAGE -->"
)
