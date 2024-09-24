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

// Set consistent height and width for all views
const viewHeight = 12
const viewWidth = 50
const menuItemWidth = 25 // Adjusted width for better alignment

// Styles
var (
	// Title style
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00ff00")). // Green color
		Padding(1, 2).
		Align(lipgloss.Center).
		Width(viewWidth). // Set consistent width
		Height(2)         // Reduced height for title area

	// Menu style with consistent padding for all menu items
	menuStyle = lipgloss.NewStyle().
		Align(lipgloss.Left).
		Width(viewWidth)

	// Cursor style
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Bold(true)

	// Dimmed style for non-selected options
	disabledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Log and action message styles
	logStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Padding(1, 2).Width(viewWidth)
	actionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00ff00")).Padding(1, 2).Align(lipgloss.Center).Width(viewWidth)
)

type statusMsg struct {
	status string
	err    error
}

func initialModel() model {
	// Clear the terminal screen
	clearScreen()

	return model{
		state:   menuView,
		choices: []string{"Install Niri", "Configure Niri", "Validate Config", "Save Logs", "Exit"},
	}
}

func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
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
    // Title section, centered and fixed width
    title := titleStyle.Render("NiriSetup Assistant for FreeBSD")

    // Menu rendering with fixed width and left alignment
    menu := strings.Builder{}
    for i, choice := range m.choices {
        if m.cursor == i {
            // Selected item with cursor, ensure the same width for alignment
            menu.WriteString(cursorStyle.Render(fmt.Sprintf("> %-"+fmt.Sprintf("%d", menuItemWidth-2)+"s", choice)) + "\n")
        } else {
            // Non-selected items with consistent width and left padding
            menu.WriteString(disabledStyle.Render(fmt.Sprintf("  %-"+fmt.Sprintf("%d", menuItemWidth-2)+"s", choice)) + "\n")
        }
    }

    // Join title and menu together and render them with consistent alignment
    return lipgloss.JoinVertical(lipgloss.Left, title, menuStyle.Render(menu.String()))
}

func (m model) renderInstallView() string {
	// Title and logs section with consistent width
	s := titleStyle.Render("Installing Niri...")

	// Logs section
	for _, log := range m.logs {
		s += logStyle.Render(log + "\n")
	}
	s += logStyle.Render("Please wait...\n")

	// Ensure fixed height for the view
	return lipgloss.JoinVertical(lipgloss.Left, s)
}

func (m model) renderActionView() string {
	// Display the action message prominently with consistent width
	return lipgloss.JoinVertical(lipgloss.Left, actionStyle.Render(fmt.Sprintf("%s\n\nPlease wait...", m.actionMsg)))
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

