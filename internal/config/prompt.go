package config

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func handleCookieFlow(configFile string) bool {
	ok, err := runCookiePrompt(configFile)
	if err != nil {
		fmt.Println("Cookie setup failed:", err)
	}
	return ok
}

type cookiePromptModel struct {
	step       int
	cursor     int
	input      textinput.Model
	passInput  textinput.Model
	errMsg     string
	configFile string
	ok         bool
	width      int
	height     int
}

const (
	cookieStepChoice = iota
	cookieStepPaste
	cookieStepPassword
)

func runCookiePrompt(configFile string) (bool, error) {
	m := newCookiePromptModel(configFile)
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return false, err
	}
	if res, ok := final.(cookiePromptModel); ok {
		return res.ok, nil
	}
	return false, fmt.Errorf("prompt canceled")
}

func newCookiePromptModel(configFile string) cookiePromptModel {
	input := textinput.New()
	input.Placeholder = "Paste Cookie here"
	input.Prompt = "Cookie: "
	input.CharLimit = 4096
	passInput := textinput.New()
	passInput.Placeholder = "Chrome Safe Storage password"
	passInput.Prompt = "Password: "
	passInput.CharLimit = 256
	passInput.EchoMode = textinput.EchoPassword
	passInput.EchoCharacter = '*'
	return cookiePromptModel{
		step:       cookieStepChoice,
		cursor:     0,
		input:      input,
		passInput:  passInput,
		configFile: configFile,
	}
}

func (m cookiePromptModel) Init() tea.Cmd {
	return nil
}

func (m cookiePromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.step == cookieStepChoice {
				m.cursor--
				if m.cursor < 0 {
					m.cursor = 2
				}
				return m, nil
			}
		case "down", "j":
			if m.step == cookieStepChoice {
				m.cursor++
				if m.cursor > 2 {
					m.cursor = 0
				}
				return m, nil
			}
		case "esc":
			if m.step == cookieStepPaste {
				m.step = cookieStepChoice
				m.input.Blur()
				return m, nil
			}
			if m.step == cookieStepPassword {
				m.step = cookieStepChoice
				m.passInput.Blur()
				return m, nil
			}
		case "enter":
			if m.step == cookieStepChoice {
				if m.cursor == 0 {
					ok, errMsg := tryImportChromeCookie(m.configFile)
					if ok {
						m.ok = true
						return m, tea.Quit
					}
					m.errMsg = "Chrome import failed"
					if errMsg != "" {
						m.errMsg += ": " + errMsg
					}
					return m, nil
				}
				if m.cursor == 1 {
					m.step = cookieStepPassword
					m.passInput.SetValue("")
					m.passInput.Focus()
					return m, nil
				}
				m.step = cookieStepPaste
				m.input.Focus()
				return m, nil
			}
			if m.step == cookieStepPaste {
				cookie := normalizeCookieHeader(m.input.Value())
				if cookie == "" {
					m.errMsg = "Cookie is empty."
					return m, nil
				}
				Config.Cookie = cookie
				if ok, errMsg := setAuthFromCookieHeader(Config.Cookie); !ok {
					m.errMsg = strings.ToUpper(errMsg[:1]) + errMsg[1:] + "."
					return m, nil
				}
				ok, err := validateCookie(Config.Cookie)
				if err != nil {
					m.errMsg = "Cookie check failed."
					return m, nil
				}
				if !ok {
					m.errMsg = "Cookie invalid."
					return m, nil
				}
				if err := saveConfig(m.configFile); err != nil {
					m.errMsg = "Failed to save config."
					return m, nil
				}
				m.ok = true
				return m, tea.Quit
			}
			if m.step == cookieStepPassword {
				password := strings.TrimSpace(m.passInput.Value())
				if password == "" {
					m.errMsg = "Password is empty."
					return m, nil
				}
				ok, errMsg := tryImportChromeCookieWithPassword(m.configFile, password)
				if ok {
					m.ok = true
					return m, tea.Quit
				}
				m.errMsg = "Chrome import failed"
				if errMsg != "" {
					m.errMsg += ": " + errMsg
				}
				return m, nil
			}
		}
		if m.step == cookieStepPaste {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		if m.step == cookieStepPassword {
			var cmd tea.Cmd
			m.passInput, cmd = m.passInput.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m cookiePromptModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#89B4FA"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))
	choiceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4"))
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A6E3A1"))

	var b strings.Builder
	b.WriteString(titleStyle.Render("Cookie Setup"))
	b.WriteString("\n")
	b.WriteString("Choose how to provide your Bilibili cookie.\n\n")
	if m.errMsg != "" {
		b.WriteString(errStyle.Render(m.errMsg))
		b.WriteString("\n\n")
	}
	if m.step == cookieStepChoice {
		choices := []string{"Import from Chrome (auto)", "Import from Chrome (enter password)", "Paste manually"}
		for i, c := range choices {
			prefix := "  "
			style := choiceStyle
			if i == m.cursor {
				prefix = "▸ "
				style = selectedStyle
			}
			b.WriteString(prefix + style.Render(c))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓ select • Enter confirm • q quit"))
		return b.String()
	}
	if m.step == cookieStepPassword {
		b.WriteString(helpStyle.Render("Enter your Chrome Safe Storage password."))
		b.WriteString("\n\n")
		b.WriteString(m.passInput.View())
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Enter confirm • Esc back • q quit"))
		return b.String()
	}
	b.WriteString(helpStyle.Render("Open https://live.bilibili.com and copy the Cookie header."))
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Enter confirm • Esc back • q quit"))
	return b.String()
}
