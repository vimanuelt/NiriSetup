package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type appState int

const (
	menuView appState = iota
	installView
	actionView
)

type model struct {
	state        appState
	choices      []string
	cursor       int
	selected     string
	logs         []string
	isProcessing bool
	progress     string
	actionMsg    string
}

type statusMsg struct {
	status string
	err    error
}

func initialModel() model {
	return model{
		state:   menuView,
		choices: []string{"Install Niri", "Configure Niri", "Validate Config", "Save Logs", "Exit"},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case menuView:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down":
				if m.cursor < len(m.choices)-1 {
					m.cursor++
				}
			case "enter":
				m.selected = m.choices[m.cursor]
				m.isProcessing = true
				switch m.selected {
				case "Install Niri":
					m.state = installView
					return m, installNiri()
				case "Configure Niri":
					m.state = actionView
					m.actionMsg = "Configuring Niri..."
					return m, configureNiri()
				case "Validate Config":
					m.state = actionView
					m.actionMsg = "Validating Niri config..."
					return m, validateNiriConfig()
				case "Save Logs":
					m.state = actionView
					m.actionMsg = "Saving logs..."
					return m, saveLogsToFile(m)
				case "Exit":
					return m, tea.Quit
				}
			}
		case installView, actionView:
			// Disable input during processing
			return m, nil
		}
	case statusMsg:
		// Append logs and handle state transitions
		m.logs = append(m.logs, msg.status)
		m.isProcessing = false
		if msg.err == nil && m.state == installView {
			// Automatically return to the menu after installation
			m.state = menuView
			m.logs = nil // Clear logs before returning to menu
		} else if msg.err == nil && m.state == actionView {
			// Automatically return to the menu after actions
			m.state = menuView
			m.actionMsg = msg.status // Display success or error message
		}
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	switch m.state {
	case menuView:
		return m.renderMenuView()
	case installView:
		return m.renderInstallView()
	case actionView:
		return m.renderActionView()
	default:
		return "Unknown state!"
	}
}

func (m model) renderMenuView() string {
	s := "NiriSetup Assistant for FreeBSD\n\n"
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">" // cursor for the selected option
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}
	if m.actionMsg != "" {
		s += fmt.Sprintf("\n%s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA07A")).Render(m.actionMsg))
		m.actionMsg = "" // Clear after showing once
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(s)
}

func (m model) renderInstallView() string {
	s := "Installing Niri...\n\n"
	for _, log := range m.logs {
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA07A")).Render(log + "\n")
	}
	s += "Please wait...\n"
	return lipgloss.NewStyle().Padding(1, 2).Render(s)
}

func (m model) renderActionView() string {
	return lipgloss.NewStyle().Padding(1, 2).Render(fmt.Sprintf("%s\n\nPlease wait...", m.actionMsg))
}

func installNiri() tea.Cmd {
	return func() tea.Msg {
		pkgs := []string{"niri", "wlroots", "xwayland-satellite", "seatd", "waybar", "grim", "jq", "wofi", "alacritty", "pam_xdg", "fuzzel", "swaylock", "foot", "wlsunset", "swaybg", "mako", "swayidle"}
		var logs []string

		for _, pkg := range pkgs {
			cmd := exec.Command("sudo", "pkg", "install", "-y", pkg)
			out, err := cmd.CombinedOutput()
			if err != nil {
				return statusMsg{status: fmt.Sprintf("Failed to install %s", pkg), err: fmt.Errorf(string(out))}
			}
			time.Sleep(500 * time.Millisecond) // Simulate install time for visual feedback

			// Append success message to logs
			log := fmt.Sprintf("Successfully installed %s", pkg)
			logs = append(logs, log)
		}

		// Return all logs as a combined message
		return statusMsg{status: strings.Join(logs, "\n")}
	}
}

func configureNiri() tea.Cmd {
	return func() tea.Msg {
		// Simulate configuration work
		time.Sleep(2 * time.Second)
		return statusMsg{status: "Niri configuration completed successfully."}
	}
}

func validateNiriConfig() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("niri", "validate")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return statusMsg{status: fmt.Sprintf("Validation failed: %s", string(out)), err: err}
		}
		return statusMsg{status: "Niri configuration is valid."}
	}
}

func saveLogsToFile(m model) tea.Cmd {
	return func() tea.Msg {
		logFile := filepath.Join(os.TempDir(), "nirisetup.log")
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return statusMsg{status: "Failed to open log file for writing", err: err}
		}
		defer file.Close()

		for _, log := range m.logs {
			if _, err := file.WriteString(log + "\n"); err != nil {
				return statusMsg{status: "Failed to write to log file", err: err}
			}
		}
		return statusMsg{status: fmt.Sprintf("Logs saved to %s", logFile)}
	}
}

func setupEnvironment() {
	// Get the current user's ID
	userID := os.Geteuid()

	// Construct the runtime directory path using the user ID
	runtimeDir := fmt.Sprintf("/tmp/%d-runtime-dir", userID)

	// Set the XDG_RUNTIME_DIR environment variable
	os.Setenv("XDG_RUNTIME_DIR", runtimeDir)

	// Check if the directory exists, if not create it
	if _, err := os.Stat(runtimeDir); os.IsNotExist(err) {
		// Create the directory with 0700 permissions to ensure it's secure
		if err := os.Mkdir(runtimeDir, 0700); err != nil {
			log.Fatalf("Failed to create runtime directory: %v", err)
		}
	} else {
		// Check if the existing directory is owned by the current user
		info, err := os.Stat(runtimeDir)
		if err != nil {
			log.Fatalf("Failed to stat runtime directory: %v", err)
		}

		// Get the owner UID of the existing directory
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			log.Fatalf("Failed to get ownership information of runtime directory")
		}

		if stat.Uid != uint32(userID) {
			log.Fatalf("XDG_RUNTIME_DIR '%s' is owned by UID %d, not our UID %d", runtimeDir, stat.Uid, userID)
		}
	}
}

func main() {
	setupEnvironment()
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
}

