package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/raisfordiner/wattching/pkg/cpuinfo"
)

type Model struct {
	CPUInfo     cpuinfo.CPUInfo
	PowerData   map[string]float64
	DomainOrder []string
	Error       error
	Spinner     spinner.Model
	showSpinner bool
}

type PowerUpdateMsg struct {
	Data map[string]float64
}

type ErrorMsg struct {
	Err error
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			PaddingLeft(1).PaddingRight(1)

	infoStyle = lipgloss.NewStyle()

	powerStyle = lipgloss.NewStyle()

	errorStyle = lipgloss.NewStyle().Bold(true)
)

func InitialModel(info cpuinfo.CPUInfo, domainOrder []string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle()
	return Model{
		CPUInfo:     info,
		PowerData:   make(map[string]float64),
		DomainOrder: domainOrder,
		Spinner:     s,
		showSpinner: true,
	}
}

func (m Model) Init() tea.Cmd {
	return m.Spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case PowerUpdateMsg:
		m.showSpinner = false
		m.PowerData = msg.Data
		return m, nil

	case ErrorMsg:
		m.Error = msg.Err
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	if m.Error != nil {
		return fmt.Sprintf("\n%s\n\n", errorStyle.Render(fmt.Sprintf("Error: %v", m.Error)))
	}

	ss := titleStyle.Render("Wattching - CPU Power Monitor") + "\n\n"

	ss += infoStyle.Render(fmt.Sprintf(" %-8s %s\n", "Vendor:", m.CPUInfo.VendorString))
	ss += infoStyle.Render(fmt.Sprintf(" %-8s %s\n", "Model:", m.CPUInfo.BrandString))
	ss += "\n"

	if m.showSpinner {
		ss += fmt.Sprintf(" %s Waiting for first power reading...", m.Spinner.View())
	} else {
		ss += " " + lipgloss.NewStyle().Bold(true).Render("Current Power Consumption") + "\n"
		for _, name := range m.DomainOrder {
			if power, ok := m.PowerData[name]; ok {
				ss += powerStyle.Render(fmt.Sprintf("  %-15s: %.2f W\n", name, power))
			}
		}
	}

	ss += infoStyle.Render("\n\n Press 'q' to quit.")
	return ss
}

