package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"speedtest/internal/measure"
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
	if diff < 0.05 && diff > -0.05 {
		st.display = st.target
		return
	}
	st.display += diff * 0.2
}

type Model struct {
	events  <-chan measure.Event
	cancel  context.CancelFunc
	server  string
	version string

	width int

	latDone    bool
	latSamples int
	latTotal   int
	lat        measure.LatencyResult

	down phaseState
	up   phaseState

	startedAt time.Time
	total     time.Duration
	done      bool
	aborted   bool
}

func New(events <-chan measure.Event, cancel context.CancelFunc, server, version string) Model {
	return Model{
		events:    events,
		cancel:    cancel,
		server:    server,
		version:   version,
		width:     80,
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
	return tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg { return frameMsg{} })
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
		if len(st.spark) > 240 {
			st.spark = st.spark[len(st.spark)-240:]
		}
	case measure.TierResult:
		st := m.phase(e.Phase)
		if n := len(st.tiers); n > 0 && st.tiers[n-1].running {
			st.tiers[n-1] = tierRow{size: e.Size, mbps: e.Mbps, elapsed: e.Elapsed, err: e.Err}
		}
		if e.Err == nil {
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
	var b strings.Builder

	b.WriteString(titleStyle.Render("speedtest") + dimStyle.Render(" "+m.version) + "\n")
	b.WriteString(dimStyle.Render("server  ") + m.server + "\n\n")

	m.renderLatency(&b)
	m.renderPhase(&b, "Download", "↓", &m.down, downStyle)
	m.renderPhase(&b, "Upload", "↑", &m.up, upStyle)

	b.WriteString("\n")
	switch {
	case m.aborted:
		b.WriteString(errStyle.Render("✗ aborted") + "\n")
	case m.done:
		b.WriteString(okStyle.Render("✓ complete") + dimStyle.Render(fmt.Sprintf(" in %.0fs", m.total.Seconds())) + "\n")
	default:
		b.WriteString(dimStyle.Render("q to abort") + "\n")
	}
	return b.String()
}

func (m Model) renderLatency(b *strings.Builder) {
	label := titleStyle.Render("Ping    ")
	switch {
	case m.latDone && m.lat.Samples == 0:
		b.WriteString(label + errStyle.Render("failed (no samples)") + "\n")
	case m.latDone:
		b.WriteString(label + fmt.Sprintf("%.2f ms", ms(m.lat.Avg)) +
			dimStyle.Render(fmt.Sprintf("   min %.2f ms   jitter %.2f ms   [%d samples]",
				ms(m.lat.Min), ms(m.lat.Jitter), m.lat.Samples)) + "\n")
	case m.latTotal > 0:
		b.WriteString(label + fmt.Sprintf("%.2f ms", ms(m.lat.Avg)) +
			dimStyle.Render(fmt.Sprintf("   min %.2f ms   %d/%d", ms(m.lat.Min), m.latSamples, m.latTotal)) + "\n")
	default:
		b.WriteString(label + dimStyle.Render("measuring…") + "\n")
	}
}

func (m Model) renderPhase(b *strings.Builder, name, arrow string, st *phaseState, style lipgloss.Style) {
	if !st.started {
		return
	}
	b.WriteString("\n" + style.Render(arrow+" "+name) + "\n")
	if !st.done {
		for i, row := range bigNumber(st.display) {
			line := "  " + style.Render(row)
			if i == 2 {
				line += dimStyle.Render("  Mbps")
			}
			b.WriteString(line + "\n")
		}
		w := m.width - 4
		if w > 48 {
			w = 48
		}
		if len(st.spark) > 0 && w > 0 {
			b.WriteString("  " + style.Render(sparkline(st.spark, w)) + "\n")
		}
	}
	for _, row := range st.tiers {
		b.WriteString(m.renderTier(row, style) + "\n")
	}
	if st.done {
		b.WriteString(dimStyle.Render("  -> max ") + titleStyle.Render(fmt.Sprintf("%.2f Mbps", st.max)) + "\n")
	}
}

func (m Model) renderTier(row tierRow, style lipgloss.Style) string {
	label := fmt.Sprintf("%8s", measure.FormatBytes(row.size))
	switch {
	case row.running:
		frac := float64(row.bytes) / float64(row.size)
		return fmt.Sprintf("  %s  %s %3.0f%%", label, style.Render(bar(frac, 24)), frac*100)
	case row.err != nil:
		return errStyle.Render(fmt.Sprintf("  %s  error: %v", label, row.err))
	default:
		return dimStyle.Render(fmt.Sprintf("  %s  %8.2f Mbps  (%.1fs)", label, row.mbps, row.elapsed.Seconds()))
	}
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
