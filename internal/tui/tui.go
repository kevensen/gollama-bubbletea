package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kevensen/gollama-bubbletea/internal/bot"
	"github.com/kevensen/gollama-bubbletea/internal/settings"
	"github.com/parakeet-nest/parakeet/llm"
)

const gap = "\n\n"

type errMsg error

// Tab styling
func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

var (
	inactiveTabBorder      = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder        = tabBorderWithBottom("┘", " ", "└")
	highlightColor         = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	greenColor             = lipgloss.AdaptiveColor{Light: "#00AA00", Dark: "#00FF00"}
	redColor               = lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#FF4444"}
	inactiveTabStyle       = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(highlightColor).Padding(0, 1)
	activeTabStyle         = inactiveTabStyle.Border(activeTabBorder, true)
	inactiveModelsTabStyle = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(greenColor).Padding(0, 1)
	activeModelsTabStyle   = inactiveModelsTabStyle.Border(activeTabBorder, true)
	inactiveRAGTabStyle    = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(redColor).Padding(0, 1)
	activeRAGTabStyle      = inactiveRAGTabStyle.Border(activeTabBorder, true)
)

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type focus int

const (
	focusTextarea focus = iota
	focusModelsViewport
	focusRAGViewport
)

type tab int

const (
	chatTab tab = iota
	modelsTab
	ragTab
)

type model struct {
	viewport       viewport.Model
	modelsViewport viewport.Model
	ragViewport    viewport.Model
	textarea       textarea.Model
	senderStyle    lipgloss.Style
	bot            *bot.Bot
	err            error
	inputError     string // Error message for invalid commands or wrong tab input
	responseBuffer string
	focus          focus
	models         []string
	selectedModel  int
	activeTab      tab
	tabs           []string
	ragEnabled     bool // RAG enable/disable state
	settings       *settings.Settings
}

func New(b *bot.Bot) *model {
	// Load settings
	appSettings, err := settings.Load()
	if err != nil {
		// If settings can't be loaded, use defaults
		appSettings = settings.DefaultSettings()
	}

	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	vp := viewport.New(30, 5)
	vp.SetContent(`Welcome to Gollama-Chat!
Type a message and press Enter to send.` + ascii + "\n\n Use the Tab key to switch between Chat, Models, and RAG tabs.\n Use Ctrl+T to focus the input field for commands from any tab.\n Special commands: /clear (clear history), /chat (chat tab), /models (models tab), /rag (rag tab), /exit or /quit (quit)\n Key bindings: Ctrl+U (clear input), Ctrl+A (start), Ctrl+E (end)\n Use ctrl+c or esc to quit.")

	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	modelsVp := viewport.New(20, 5)

	modelsVp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(greenColor) // Green border to match tab

	ragVp := viewport.New(20, 5)
	ragVp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("1")) // Red border

	ta.KeyMap.InsertNewline.SetEnabled(false)

	// Apply saved last model if it exists and is valid
	if appSettings.LastModel != "" {
		err := b.ModelManager.UseModel(appSettings.LastModel)
		if err != nil {
			// If saved model is invalid, keep default but don't error
			appSettings.LastModel = b.ModelManager.CurrentModel()
		}
	}

	return &model{
		textarea:       ta,
		viewport:       vp,
		modelsViewport: modelsVp,
		ragViewport:    ragVp,
		senderStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		bot:            b,
		err:            nil,
		focus:          focusTextarea,
		selectedModel:  0,
		activeTab:      chatTab,
		tabs:           []string{"Chat", "Models", "RAG"},
		ragEnabled:     appSettings.RAGEnabled, // Load RAG state from settings
		settings:       appSettings,
	}
}

func (m *model) handleChatResponse(resp llm.Answer) error {
	m.responseBuffer += resp.Message.Content
	if resp.Done && m.responseBuffer != "" {
		m.bot.MessageManager.AddMessage(resp.Message)
		m.responseBuffer = ""
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.bot.MessageManager.StyledMessages(), "\n")))
		m.viewport.GotoBottom()
	}
	return nil
}

func (m *model) updateTabNames() {
	currentModel := m.bot.ModelManager.CurrentModel()
	ragStatus := "Disabled"
	if m.ragEnabled {
		ragStatus = "Enabled"
	}
	m.tabs[0] = "Chat"
	m.tabs[1] = "Models: " + currentModel
	m.tabs[2] = "RAG: " + ragStatus
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.fetchModels)
}

func (m *model) fetchModels() tea.Msg {
	models := m.bot.ModelManager.ModelNames()
	m.models = models
	m.updateTabNames()
	m.updateModelsViewportContent()
	m.updateRAGViewportContent()
	return nil
}

func (m *model) updateModelsViewportContent() {
	currentModel := m.bot.ModelManager.CurrentModel()
	styledModels := []string{"Available Models:", ""}
	for i, model := range m.models {
		style := lipgloss.NewStyle()
		prefix := "  "

		if model == currentModel {
			style = style.Foreground(lipgloss.Color("2")) // Highlight current model in green
			prefix = "→ "
		}
		if i == m.selectedModel && m.focus == focusModelsViewport {
			style = style.Background(lipgloss.Color("7")).Foreground(lipgloss.Color("0")) // Highlight selected model
		}
		styledModels = append(styledModels, prefix+style.Render(model))
	}

	// Add instructions
	styledModels = append(styledModels, "", "Controls:", "↑/↓ - Navigate", "Enter - Select Model", "Tab - Switch to Chat")

	m.modelsViewport.SetContent(strings.Join(styledModels, "\n"))
}

func (m *model) updateRAGViewportContent() {
	statusColor := "1" // Red for disabled
	statusText := "DISABLED"
	toggleText := "Press Enter to Enable"

	if m.ragEnabled {
		statusColor = "2" // Green for enabled
		statusText = "ENABLED"
		toggleText = "Press Enter to Disable"
	}

	content := []string{
		"RAG Settings",
		"",
		"Status: " + lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Bold(true).Render(statusText),
		"",
		toggleText,
		"",
		"Controls:",
		"Enter - Toggle RAG On/Off",
		"Tab - Switch to Chat",
	}

	m.ragViewport.SetContent(strings.Join(content, "\n"))
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd  tea.Cmd
		vpCmd  tea.Cmd
		mvCmd  tea.Cmd
		ragCmd tea.Cmd
	)

	if m.focus == focusTextarea {
		m.textarea, tiCmd = m.textarea.Update(msg)
	}
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.modelsViewport, mvCmd = m.modelsViewport.Update(msg)
	m.ragViewport, ragCmd = m.ragViewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Calculate full width for chat tab
		chatViewportWidth := msg.Width - 4 // 4 for padding and border

		// Models viewport width for models tab
		modelsViewportWidth := min(msg.Width-4, m.bot.ModelManager.MaxModelNameLength()+10)

		m.viewport.Width = chatViewportWidth
		m.modelsViewport.Width = modelsViewportWidth
		m.ragViewport.Width = modelsViewportWidth
		m.textarea.SetWidth(chatViewportWidth)

		// Height calculations (accounting for tab header)
		tabHeaderHeight := 3 // Height of tab header
		availableHeight := msg.Height - m.textarea.Height() - lipgloss.Height(gap) - tabHeaderHeight

		m.viewport.Height = availableHeight
		m.modelsViewport.Height = availableHeight
		m.ragViewport.Height = availableHeight

		if m.bot.MessageLen() > 0 {
			// Wrap content before setting it.
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.bot.MessageManager.StyledMessages(), "\n")))
		}
		m.viewport.GotoBottom()
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			// Switch between tabs
			switch m.activeTab {
			case chatTab:
				m.activeTab = modelsTab
				m.focus = focusModelsViewport
				m.textarea.Blur()
			case modelsTab:
				m.activeTab = ragTab
				m.focus = focusRAGViewport
				m.textarea.Blur()
			case ragTab:
				m.activeTab = chatTab
				m.focus = focusTextarea
				m.textarea.Focus()
			}
			m.updateModelsViewportContent()
			m.updateRAGViewportContent()
		case "ctrl+t":
			// Switch focus to textarea for command input (works from any tab)
			if m.focus != focusTextarea {
				m.focus = focusTextarea
				m.textarea.Focus()
			} else {
				// Switch back to tab-specific focus
				switch m.activeTab {
				case chatTab:
					m.focus = focusTextarea // Stay on textarea for chat
				case modelsTab:
					m.focus = focusModelsViewport
					m.textarea.Blur()
				case ragTab:
					m.focus = focusRAGViewport
					m.textarea.Blur()
				}
			}
		case "up":
			if m.activeTab == modelsTab && m.focus == focusModelsViewport && m.selectedModel > 0 {
				m.selectedModel--
				m.updateModelsViewportContent()
			} else if m.activeTab == chatTab && m.focus == focusTextarea {
				m.viewport.ScrollUp(1)
			}
		case "down":
			if m.activeTab == modelsTab && m.focus == focusModelsViewport && m.selectedModel < len(m.models)-1 {
				m.selectedModel++
				m.updateModelsViewportContent()
			} else if m.activeTab == chatTab && m.focus == focusTextarea {
				m.viewport.ScrollDown(1)
			}
		case "enter":
			input := m.textarea.Value()

			// Handle tab-specific viewport interactions first (when not focused on textarea)
			if m.focus != focusTextarea {
				if m.activeTab == modelsTab && m.focus == focusModelsViewport {
					selectedModel := m.models[m.selectedModel]
					err := m.bot.ModelManager.UseModel(selectedModel)
					if err != nil {
						msg := llm.Message{Role: "error", Content: err.Error()}
						m.bot.MessageManager.AddMessage(msg)
					} else {
						// Save the selected model to settings
						m.settings.SetLastModel(selectedModel)
					}
					m.updateTabNames()
					m.updateModelsViewportContent()
				} else if m.activeTab == ragTab && m.focus == focusRAGViewport {
					// Toggle RAG enabled/disabled
					m.ragEnabled = !m.ragEnabled
					// Save RAG state to settings
					m.settings.SetRAGEnabled(m.ragEnabled)
					m.updateTabNames()
					m.updateRAGViewportContent()
				}
				return m, nil
			}

			// Handle textarea input
			if input != "" {
				isCommand := strings.HasPrefix(input, "/")

				if isCommand {
					// Clear any existing error when processing a command
					m.inputError = ""

					// Handle valid commands
					switch input {
					case "/exit", "/quit":
						return m, tea.Quit
					case "/chat":
						m.activeTab = chatTab
						m.textarea.Reset()
						return m, nil
					case "/models":
						m.activeTab = modelsTab
						m.updateTabNames()
						m.updateModelsViewportContent()
						m.textarea.Reset()
						return m, nil
					case "/rag":
						m.activeTab = ragTab
						m.textarea.Reset()
						return m, nil
					case "/clear":
						// /clear only works on chat tab
						if m.activeTab == chatTab {
							m.bot.ClearMessages()
							m.viewport.SetContent(`Welcome to Gollama-Chat!
Type a message and press Enter to send.` + ascii + "\n\n Use the Tab key to switch between Chat, Models, and RAG tabs.\n Use Ctrl+T to focus the input field for commands from any tab.\n Special commands: /clear (clear history), /chat (chat tab), /models (models tab), /rag (rag tab), /exit or /quit (quit)\n Key bindings: Ctrl+U (clear input), Ctrl+A (start), Ctrl+E (end)\n Use ctrl+c or esc to quit.")
							m.textarea.Reset()
							return m, nil
						} else {
							// Invalid command on non-chat tab
							m.inputError = "Command '/clear' is only available on the chat tab"
							m.textarea.Reset()
							return m, nil
						}
					default:
						// Invalid command
						m.inputError = "Invalid command: " + input
						m.textarea.Reset()
						return m, nil
					}
				} else {
					// Non-command input
					if m.activeTab != chatTab {
						// Non-chat tab with non-command input
						m.inputError = "Chat input detected. Please switch to the chat tab and press enter"
						return m, nil // Don't clear textarea for this case
					} else {
						// Valid chat input - clear any existing error
						m.inputError = ""

						// Regular chat message handling
						m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.bot.MessageManager.StyledMessages(), "\n")))
						ctx := context.Background()
						ans, err := m.bot.SendMessage(ctx, "user", input)
						if err != nil {
							msg := llm.Message{Role: "error", Content: err.Error()}
							m.bot.MessageManager.AddMessage(msg)
						}
						if ans != nil {
							m.handleChatResponse(*ans)
						}
						m.textarea.Reset()
						m.viewport.GotoBottom()
					}
				}
			}
		case "ctrl+u":
			// Clear text before cursor (like bash)
			if m.focus == focusTextarea {
				m.textarea.SetValue("")
			}
		case "ctrl+a":
			// Go to beginning of text (like bash)
			if m.focus == focusTextarea {
				m.textarea.SetCursor(0)
			}
		case "ctrl+e":
			// Go to end of text (like bash)
			if m.focus == focusTextarea {
				m.textarea.SetCursor(len(m.textarea.Value()))
			}
		case "ctrl+c", "esc":
			return m, tea.Quit
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd, mvCmd, ragCmd)
}

func (m *model) View() string {
	// Render tabs
	var renderedTabs []string

	for i, tabName := range m.tabs {
		var style lipgloss.Style
		isFirst, isLast, isActive := i == 0, i == len(m.tabs)-1, i == int(m.activeTab)

		// Apply styles based on tab type and state
		if i == 1 { // Models tab - always green
			if isActive {
				style = activeModelsTabStyle
			} else {
				style = inactiveModelsTabStyle
			}
		} else if i == 2 { // RAG tab - always red
			if isActive {
				style = activeRAGTabStyle
			} else {
				style = inactiveRAGTabStyle
			}
		} else { // Chat tab - purple/blue
			if isActive {
				style = activeTabStyle
			} else {
				style = inactiveTabStyle
			}
		}

		border, _, _, _, _ := style.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst && !isActive {
			border.BottomLeft = "├"
		} else if isLast && isActive {
			border.BottomRight = "│"
		} else if isLast && !isActive {
			border.BottomRight = "┤"
		}
		style = style.Border(border)
		renderedTabs = append(renderedTabs, style.Render(tabName))
	}

	tabRow := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	// Render content based on active tab
	var content string
	if m.activeTab == chatTab {
		// Chat tab: show full-width chat viewport
		content = m.viewport.View()
	} else if m.activeTab == modelsTab {
		// Models tab: show models viewport centered
		content = lipgloss.NewStyle().
			Align(lipgloss.Center).
			Width(m.viewport.Width).
			Render(m.modelsViewport.View())
	} else { // RAG tab
		// RAG tab: show RAG viewport centered
		content = lipgloss.NewStyle().
			Align(lipgloss.Center).
			Width(m.viewport.Width).
			Render(m.ragViewport.View())
	}

	// Handle error message display without affecting layout spacing
	var errorDisplay string
	var adjustedGap = gap
	if m.inputError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")). // Red color
			Bold(true).
			Padding(0, 1)
		errorDisplay = errorStyle.Render("⚠ "+m.inputError) + "\n"
		adjustedGap = "\n" // Reduce gap since we're adding error message line
	}

	// Combine tabs, content, error message, and textarea
	return tabRow + "\n" + content + adjustedGap + errorDisplay + m.textarea.View()
}
