package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"speedtest/internal/constants"
	"speedtest/internal/measure"
	"speedtest/style"
	"speedtest/style/builtin"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "244", Dark: "241"})
	downStyle  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "31", Dark: "45"})
	upStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "166", Dark: "208"})
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "28", Dark: "42"})
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "124", Dark: "203"})
)

type (
	eventMsg struct{ ev measure.Event }
	frameMsg struct{}
)

type tierRow struct {
	size    int
	mbps    float64
	elapsed time.Duration
	bytes   int64
	err     error
	running bool
}

type phaseState struct {
	started bool
	done    bool
	max     float64
	tiers   []tierRow
	spark   []float64
	target  float64
	display float64
}

func (st *phaseState) animate() {
	diff := st.target - st.display
	if diff < constants.AnimationSnap && diff > -constants.AnimationSnap {
		st.display = st.target
		return
	}
	st.display += diff * constants.AnimationEase
}

type Model struct {
	events <-chan measure.Event
	cancel context.CancelFunc
	server string

	width int

	ping       bool
	latDone    bool
	latSamples int
	latTotal   int
	lat        measure.LatencyResult

	down phaseState
	up   phaseState

	style style.Style

	startedAt time.Time
	total     time.Duration
	done      bool
	aborted   bool
	failed    bool
}

func New(events <-chan measure.Event, cancel context.CancelFunc, server string, ping bool, s style.Style) Model {
	return Model{
		events:    events,
		cancel:    cancel,
		server:    server,
		ping:      ping,
		width:     constants.DefaultWidth,
		style:     s,
		startedAt: time.Now(),
	}
}

func waitEvent(ch <-chan measure.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return eventMsg{ev}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(constants.FrameInterval, func(time.Time) tea.Msg { return frameMsg{} })
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(waitEvent(m.events), tickCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.aborted = true
			m.total = time.Since(m.startedAt)
			m.cancel()
			return m, tea.Quit
		}
		return m, nil
	case frameMsg:
		if m.done || m.aborted {
			return m, nil
		}
		(&m.down).animate()
		(&m.up).animate()
		return m, tickCmd()
	case eventMsg:
		return m.handleEvent(msg.ev)
	}
	return m, nil
}

func (m Model) handleEvent(ev measure.Event) (tea.Model, tea.Cmd) {
	switch e := ev.(type) {
	case measure.LatencyProgress:
		m.latSamples, m.latTotal = e.Done, e.Total
		m.lat.Min, m.lat.Avg = e.Min, e.Avg
	case measure.LatencyResult:
		m.lat, m.latDone = e, true
	case measure.TierStarted:
		st := m.phase(e.Phase)
		st.started = true
		st.tiers = append(st.tiers, tierRow{size: e.Size, running: true})
	case measure.TierProgress:
		st := m.phase(e.Phase)
		if n := len(st.tiers); n > 0 && st.tiers[n-1].running {
			st.tiers[n-1].bytes = e.Bytes
		}
		st.target = e.Mbps
		st.spark = append(st.spark, e.Mbps)
		if len(st.spark) > constants.SparkHistory {
			st.spark = st.spark[len(st.spark)-constants.SparkHistory:]
		}
	case measure.TierResult:
		st := m.phase(e.Phase)
		if n := len(st.tiers); n > 0 && st.tiers[n-1].running {
			st.tiers[n-1] = tierRow{size: e.Size, mbps: e.Mbps, elapsed: e.Elapsed, err: e.Err}
		}
		if e.Err != nil {
			m.failed = true
		} else {
			st.target = e.Mbps
		}
	case measure.PhaseResult:
		st := m.phase(e.Phase)
		st.done = true
		st.max = e.MaxMbps
	case measure.Done:
		m.done = true
		m.total = time.Since(m.startedAt)
		return m, tea.Quit
	}
	return m, waitEvent(m.events)
}

func (m *Model) phase(p measure.Phase) *phaseState {
	if p == measure.Upload {
		return &m.up
	}
	return &m.down
}

func (m Model) View() string {
	var blocks []string
	if m.server != "" {
		blocks = append(blocks, dimStyle.Render("server  ")+m.server)
	}
	if m.ping {
		blocks = append(blocks, m.latencyBlock())
	}
	if blk := m.phaseBlock("Download", "↓", &m.down, downStyle); blk != "" {
		blocks = append(blocks, blk)
	}
	if blk := m.phaseBlock("Upload", "↑", &m.up, upStyle); blk != "" {
		blocks = append(blocks, blk)
	}

	switch {
	case m.aborted:
		blocks = append(blocks, errStyle.Render("✗ aborted"))
	case m.done && m.failed:
		blocks = append(blocks, errStyle.Render("✗ incomplete")+dimStyle.Render(fmt.Sprintf(" in %.0fs", m.total.Seconds())))
	case m.done:
		blocks = append(blocks, okStyle.Render("✓ complete")+dimStyle.Render(fmt.Sprintf(" in %.0fs", m.total.Seconds())))
	default:
		blocks = append(blocks, dimStyle.Render("q to abort"))
	}
	return strings.Join(blocks, "\n\n") + "\n"
}

func (m Model) latencyBlock() string {
	label := titleStyle.Render("Ping    ")
	switch {
	case m.latDone && m.lat.Samples == 0:
		return label + errStyle.Render("failed (no samples)")
	case m.latDone:
		return label + fmt.Sprintf("%.2f ms", ms(m.lat.Avg)) +
			dimStyle.Render(fmt.Sprintf("   min %.2f ms   [%d samples]", ms(m.lat.Min), m.lat.Samples)) +
			"\n" + titleStyle.Render("Jitter  ") + fmt.Sprintf("%.2f ms", ms(m.lat.Jitter))
	case m.latTotal > 0:
		return label + fmt.Sprintf("%.2f ms", ms(m.lat.Avg)) +
			dimStyle.Render(fmt.Sprintf("   min %.2f ms   %d/%d", ms(m.lat.Min), m.latSamples, m.latTotal))
	default:
		return label + dimStyle.Render("measuring…")
	}
}

func (m Model) phaseBlock(name, arrow string, st *phaseState, accent lipgloss.Style) string {
	if !st.started {
		return ""
	}
	title := accent.Render(fmt.Sprintf("%-10s", arrow+" "+name))
	if st.done && st.max > 0 {
		title += "  " + titleStyle.Render(fmt.Sprintf("%.2f Mbps", st.max))
	}
	lines := []string{title}
	f := style.Frame{
		Phase:   strings.ToLower(name),
		Mbps:    st.display,
		History: st.spark,
		Width:   m.width - 4,
		Accent:  func(s string) string { return accent.Render(s) },
		Dim:     func(s string) string { return dimStyle.Render(s) },
	}
	if n := len(st.tiers); n > 0 && st.tiers[n-1].running {
		f.Progress = float64(st.tiers[n-1].bytes) / float64(st.tiers[n-1].size)
	}
	if !st.done {
		for _, line := range m.live(f) {
			lines = append(lines, "  "+line)
		}
	}
	for _, row := range st.tiers {
		lines = append(lines, m.renderTier(row, f))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderTier(row tierRow, f style.Frame) string {
	label := fmt.Sprintf("%8s", measure.FormatBytes(row.size))
	switch {
	case row.running:
		f.Width = constants.BarWidth
		return fmt.Sprintf("  %s  %s %3.0f%%", label, m.bar(f), f.Progress*100)
	case row.err != nil:
		return errStyle.Render(fmt.Sprintf("  %s  %s", label, measure.Reason(row.err)))
	default:
		return dimStyle.Render(fmt.Sprintf("  %s  %8.2f Mbps  (%.1fs)", label, row.mbps, row.elapsed.Seconds()))
	}
}

func (m Model) live(f style.Frame) (lines []string) {
	defer func() {
		if r := recover(); r != nil {
			lines = append(builtin.Digits.Live(f), errStyle.Render(fmt.Sprintf("style %q: %v", m.style.Name, r)))
		}
	}()
	return m.style.Live(f)
}

func (m Model) bar(f style.Frame) (s string) {
	defer func() {
		if recover() != nil {
			s = builtin.Digits.Bar(f)
		}
	}()
	if b := m.style.Bar; b != nil {
		return b(f)
	}
	return builtin.Digits.Bar(f)
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
