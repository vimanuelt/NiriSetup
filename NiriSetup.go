package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	choices      []string
	cursor       int
	selected     string
	quitting     bool
	logs         []string
	progress     string
	isProcessing bool
}

type statusMsg struct {
	status string
	err    error
}

func initialModel() model {
	return model{
		choices: []string{"Install Niri", "Configure Niri", "Validate Config", "Save Logs", "Exit"},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(time.Second*1, func(t time.Time) tea.Msg { return nil }),
		func() tea.Msg { return statusMsg{status: "Initializing NiriSetup..."} },
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
				return m, installNiri()
			case "Configure Niri":
				return m, configureNiri()
			case "Validate Config":
				return m, validateNiriConfig()
			case "Save Logs":
				return m, saveLogsToFile(m)
			case "Exit":
				m.quitting = true
				return m, tea.Quit
			}
		}
	case statusMsg:
		m.logs = append(m.logs, msg.status)
		m.isProcessing = false
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("Error: %v", msg.err))
		}
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	s := "NiriSetup Assistant for FreeBSD\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">" // cursor for the selected option
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	if m.selected != "" {
		s += "\nSelected: " + m.selected + "\n"
		if m.isProcessing {
			s += "Processing" + strings.Repeat(".", len(m.progress)%4) + "\n"
		}
		for _, log := range m.logs {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA07A")).Render(log + "\n")
		}
	}

	if m.quitting {
		s += "\nExiting..."
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(s)
}

func installNiri() tea.Cmd {
	return func() tea.Msg {
		pkgs := []string{"niri", "wlroots", "xwayland-satellite", "seatd", "waybar", "grim", "jq", "wofi", "alacritty", "pam_xdg", "fuzzel", "swaylock", "foot", "wlsunset", "swaybg", "mako", "swayidle"}
		for _, pkg := range pkgs {
			cmd := exec.Command("sudo", "pkg", "install", "-y", pkg)
			out, err := cmd.CombinedOutput()
			if err != nil {
				return statusMsg{status: fmt.Sprintf("Failed to install %s", pkg), err: fmt.Errorf(string(out))}
			}
			time.Sleep(500 * time.Millisecond) // Simulate install time for visual feedback
			fmt.Printf("Successfully installed %s\n", pkg) // Output to console for each successful install
		}

		// Use the config.kdl from the current working directory
		workingDir, err := os.Getwd()
		if err != nil {
			return statusMsg{status: "Failed to get current working directory", err: err}
		}

		sourceConfigPath := filepath.Join(workingDir, "config.kdl")
		homeDir, _ := os.UserHomeDir()
		configDir := filepath.Join(homeDir, ".config", "niri")
		destConfigPath := filepath.Join(configDir, "config.kdl")

		// Create the Niri configuration directory if it doesn't exist
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return statusMsg{status: "Failed to create niri configuration directory", err: err}
		}

		// Copy the config.kdl file from the current working directory to the config directory
		if err := copyFile(sourceConfigPath, destConfigPath); err != nil {
			return statusMsg{status: "Failed to copy config.kdl", err: err}
		}

		return statusMsg{status: "Niri installation and configuration completed successfully."}
	}
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

func configureNiri() tea.Cmd {
	return func() tea.Msg {
		// Configuration process can be customized based on user input or file changes
		return statusMsg{status: "Configuration completed."}
	}
}

func validateNiriConfig() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("niri", "validate")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return statusMsg{status: fmt.Sprintf("Configuration validation failed: %s", string(out)), err: err}
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

	// Create the directory if it doesn't exist
	if _, err := os.Stat(runtimeDir); os.IsNotExist(err) {
		if err := os.Mkdir(runtimeDir, 0700); err != nil {
			log.Println("Failed to create runtime directory:", err)
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

