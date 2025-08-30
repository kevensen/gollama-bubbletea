package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
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
	inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder   = tabBorderWithBottom("┘", " ", "└")
	highlightColor    = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	greenColor        = lipgloss.AdaptiveColor{Light: "#00AA00", Dark: "#00FF00"}
	redColor          = lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#FF4444"}
	blueColor         = lipgloss.AdaptiveColor{Light: "#0066CC", Dark: "#4499FF"}

	// Dark mode colors
	darkModeTextColor       = lipgloss.Color("#E0E0E0") // Non-accented text
	darkModeAccentColor     = lipgloss.Color("#DAA520") // Goldenrod for accented text
	darkModeBackgroundColor = lipgloss.Color("#121212") // Dark background
	darkModeBorderColor     = lipgloss.Color("#FFD700") // Gold for lines/borders

	inactiveTabStyle         = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(highlightColor).Padding(0, 1)
	activeTabStyle           = inactiveTabStyle.Border(activeTabBorder, true)
	inactiveModelsTabStyle   = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(greenColor).Padding(0, 1)
	activeModelsTabStyle     = inactiveModelsTabStyle.Border(activeTabBorder, true)
	inactiveRAGTabStyle      = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(redColor).Padding(0, 1)
	activeRAGTabStyle        = inactiveRAGTabStyle.Border(activeTabBorder, true)
	inactiveSettingsTabStyle = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(blueColor).Padding(0, 1)
	activeSettingsTabStyle   = inactiveSettingsTabStyle.Border(activeTabBorder, true)
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
	focusSettingsViewport
	focusURLInput
)

type tab int

const (
	chatTab tab = iota
	modelsTab
	ragTab
	settingsTab
)

type model struct {
	viewport         viewport.Model
	modelsViewport   viewport.Model
	ragViewport      viewport.Model
	settingsViewport viewport.Model
	textarea         textarea.Model
	urlTextInput     textinput.Model // Text input for URL in settings
	senderStyle      lipgloss.Style
	bot              *bot.Bot
	err              error
	inputError       string // Error message for invalid commands or wrong tab input
	responseBuffer   string
	focus            focus
	models           []string
	selectedModel    int
	activeTab        tab
	tabs             []string
	ragEnabled       bool // RAG enable/disable state
	settings         *settings.Settings
	connectionValid  bool   // Whether Ollama connection is valid
	urlInput         string // Current URL being entered in settings
	darkMode         bool   // Dark mode state
}

func New(b *bot.Bot) *model {
	// Load settings
	appSettings, err := settings.Load()
	if err != nil {
		// If settings can't be loaded, use defaults
		appSettings = settings.DefaultSettings()
	}

	// Test connection to determine initial state
	connectionValid := false
	if appSettings.OllamaURL != "" {
		if bot.TestConnection(appSettings.OllamaURL) == nil {
			connectionValid = true
		}
	}

	ta := textarea.New()
	ta.Placeholder = "Send a message..."

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	// Initialize URL text input for settings
	urlInput := textinput.New()
	urlInput.Placeholder = "http://localhost:11434"
	urlInput.Width = 40
	urlInput.Prompt = "URL: "
	if appSettings.OllamaURL != "" {
		urlInput.SetValue(appSettings.OllamaURL)
	}

	vp := viewport.New(30, 5)
	vp.SetContent(`Welcome to Gollama-Chat!
Type a message and press Enter to send.` + ascii + `

Use the Tab key to switch between Chat, Models, RAG, and Settings tabs.
Use Ctrl+T to focus the input field for commands from any tab.

Special commands:
  • /clear - clear chat history
  • /chat - switch to chat tab
  • /models - switch to models tab
  • /rag - switch to RAG tab
  • /settings - switch to settings tab
  • /dark - toggle dark mode
  • /exit or /quit - quit application

Key bindings:
  • Ctrl+U - clear input
  • Ctrl+A - go to start
  • Ctrl+E - go to end
  • Ctrl+C or Esc - quit`)

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

	settingsVp := viewport.New(50, 5)
	settingsVp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(blueColor) // Blue border

	// Determine initial tab and focus
	initialTab := chatTab
	initialFocus := focusTextarea
	if !connectionValid {
		initialTab = settingsTab
		initialFocus = focusTextarea // Use textarea for URL input
		ta.Placeholder = "Enter Ollama URL (e.g., http://localhost:11434)"
		ta.Focus() // Focus textarea for URL input
		// Pre-fill with current URL if any
		if appSettings.OllamaURL != "" {
			ta.SetValue(appSettings.OllamaURL)
		}
	} else {
		ta.Focus()
		urlInput.Blur()
		// Apply saved last model if it exists and is valid
		if appSettings.LastModel != "" && b.ModelManager != nil {
			err := b.ModelManager.UseModel(appSettings.LastModel)
			if err != nil {
				// If saved model is invalid, keep default but don't error
				appSettings.LastModel = b.ModelManager.CurrentModel()
			}
		}
	}

	// Initialize models list if we have a valid connection
	var initialModels []string
	if connectionValid && b.ModelManager != nil {
		initialModels = b.ModelManager.ModelNames()
	}

	return &model{
		textarea:         ta,
		viewport:         vp,
		modelsViewport:   modelsVp,
		ragViewport:      ragVp,
		settingsViewport: settingsVp,
		urlTextInput:     urlInput,
		senderStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		bot:              b,
		err:              nil,
		focus:            initialFocus,
		selectedModel:    0,
		activeTab:        initialTab,
		tabs:             []string{"Chat", "Models", "RAG", "Settings"},
		ragEnabled:       appSettings.RAGEnabled, // Load RAG state from settings
		settings:         appSettings,
		connectionValid:  connectionValid,
		urlInput:         appSettings.OllamaURL,
		darkMode:         appSettings.DarkMode, // Load dark mode state from settings
		models:           initialModels,        // Initialize models list
	}
}

func (m *model) handleChatResponse(resp llm.Answer) error {
	m.responseBuffer += resp.Message.Content
	if resp.Done && m.responseBuffer != "" {
		m.bot.MessageManager.AddMessage(resp.Message)
		m.responseBuffer = ""
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.bot.MessageManager.StyledMessages(), "\n")))
		m.viewport.GotoBottom()
		// Update tab names to reflect new token count
		m.updateTabNames()
	}
	return nil
}

func (m *model) updateTabNames() {
	currentModel := "No Model"
	if m.bot.ModelManager != nil {
		currentModel = m.bot.ModelManager.CurrentModel()
	}
	ragStatus := "Disabled"
	if m.ragEnabled {
		ragStatus = "Enabled"
	}
	connectionStatus := "Not Connected"
	if m.connectionValid {
		connectionStatus = "Connected"
	}

	// Get token count for chat tab
	tokenCount := m.bot.EstimateTokens()
	chatTabName := "Chat"
	if tokenCount > 0 {
		// Try to get context window size as well
		if m.connectionValid && m.bot.ModelManager != nil {
			contextSize, err := m.bot.GetContextWindowSize()
			if err == nil {
				contextSizeFormatted := formatTokenCount(contextSize)
				chatTabName = fmt.Sprintf("Chat (%d/%s tokens)", tokenCount, contextSizeFormatted)
			} else {
				chatTabName = fmt.Sprintf("Chat (%d tokens)", tokenCount)
			}
		} else {
			chatTabName = fmt.Sprintf("Chat (%d tokens)", tokenCount)
		}
	}

	m.tabs[0] = chatTabName
	m.tabs[1] = "Models: " + currentModel
	m.tabs[2] = "RAG: " + ragStatus
	m.tabs[3] = "Settings: " + connectionStatus
}

// formatTokenCount converts large token counts to more readable format (e.g., 131072 -> "128K")
func formatTokenCount(count int) string {
	if count >= 1048576 { // 1024 * 1024
		// Convert to millions (M) using binary base
		millions := float64(count) / 1048576.0
		if millions == float64(int(millions)) {
			return fmt.Sprintf("%.0fM", millions)
		}
		return fmt.Sprintf("%.1fM", millions)
	} else if count >= 1024 {
		// Convert to thousands (K) using binary base
		thousands := float64(count) / 1024.0
		if thousands == float64(int(thousands)) {
			return fmt.Sprintf("%.0fK", thousands)
		}
		return fmt.Sprintf("%.1fK", thousands)
	}
	// For small numbers, show as-is
	return fmt.Sprintf("%d", count)
}

// applyTheme applies the current theme (light or dark mode) to all UI elements
func (m *model) applyTheme() {
	if m.darkMode {
		// Dark mode theme
		borderColor := darkModeBorderColor
		textColor := darkModeTextColor

		// Update viewport styles
		m.viewport.Style = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Foreground(textColor)

		m.modelsViewport.Style = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Foreground(textColor)

		m.ragViewport.Style = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Foreground(textColor)

		m.settingsViewport.Style = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Foreground(textColor)

		// Update sender style for dark mode
		m.senderStyle = lipgloss.NewStyle().Foreground(darkModeAccentColor)
	} else {
		// Light mode theme (original colors)
		m.viewport.Style = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

		m.modelsViewport.Style = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(greenColor)

		m.ragViewport.Style = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("1"))

		m.settingsViewport.Style = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(blueColor)

		// Restore original sender style
		m.senderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	}
}

// getTabStyles returns theme-aware tab styles
func (m *model) getTabStyles() (inactive, active, inactiveModels, activeModels, inactiveRAG, activeRAG, inactiveSettings, activeSettings lipgloss.Style) {
	if m.darkMode {
		// Dark mode tab styles - all use gold borders
		borderColor := darkModeBorderColor
		textColor := darkModeTextColor

		inactive = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(borderColor).Foreground(textColor).Padding(0, 1)
		active = inactive.Border(activeTabBorder, true).Foreground(darkModeAccentColor)

		// All tabs use same dark mode styling
		inactiveModels = inactive
		activeModels = active
		inactiveRAG = inactive
		activeRAG = active
		inactiveSettings = inactive
		activeSettings = active
	} else {
		// Light mode tab styles (original colors)
		inactive = inactiveTabStyle
		active = activeTabStyle
		inactiveModels = inactiveModelsTabStyle
		activeModels = activeModelsTabStyle
		inactiveRAG = inactiveRAGTabStyle
		activeRAG = activeRAGTabStyle
		inactiveSettings = inactiveSettingsTabStyle
		activeSettings = activeSettingsTabStyle
	}
	return
}

func (m *model) Init() tea.Cmd {
	m.applyTheme() // Apply theme first
	m.updateTabNames()
	m.updateModelsViewportContent()
	m.updateRAGViewportContent()
	m.updateSettingsViewportContent()
	m.updateInputPlaceholder()
	return textarea.Blink
}

func (m *model) fetchModels() tea.Msg {
	// Only fetch models if connection is valid and ModelManager exists
	if m.connectionValid && m.bot.ModelManager != nil {
		models := m.bot.ModelManager.ModelNames()
		m.models = models
	}
	m.updateTabNames()
	m.updateModelsViewportContent()
	m.updateRAGViewportContent()
	m.updateSettingsViewportContent()
	return nil
}

func (m *model) updateModelsViewportContent() {
	if m.bot.ModelManager == nil {
		m.modelsViewport.SetContent("No connection to Ollama server.\nPlease configure URL in Settings tab.")
		return
	}

	currentModel := m.bot.ModelManager.CurrentModel()
	styledModels := []string{"Available Models:", ""}
	for i, model := range m.models {
		style := lipgloss.NewStyle()
		prefix := "  "

		if model == currentModel {
			// Use theme-aware colors
			if m.darkMode {
				style = style.Foreground(darkModeAccentColor) // Goldenrod for current model in dark mode
			} else {
				style = style.Foreground(lipgloss.Color("2")) // Green for current model in light mode
			}
			prefix = "→ "
		}
		if i == m.selectedModel && m.focus == focusModelsViewport {
			if m.darkMode {
				style = style.Background(darkModeAccentColor).Foreground(darkModeBackgroundColor)
			} else {
				style = style.Background(lipgloss.Color("7")).Foreground(lipgloss.Color("0"))
			}
		}
		styledModels = append(styledModels, prefix+style.Render(model))
	}

	// Add instructions
	styledModels = append(styledModels, "", "Controls:", "↑/↓ - Navigate", "Enter - Select Model", "Tab - Switch to Chat")

	m.modelsViewport.SetContent(strings.Join(styledModels, "\n"))
}

func (m *model) updateRAGViewportContent() {
	var statusColor lipgloss.Color
	statusText := "DISABLED"
	toggleText := "Press Enter to Enable"

	if m.ragEnabled {
		statusText = "ENABLED"
		toggleText = "Press Enter to Disable"
	}

	// Use theme-aware colors
	if m.darkMode {
		if m.ragEnabled {
			statusColor = darkModeAccentColor // Goldenrod for enabled in dark mode
		} else {
			statusColor = darkModeTextColor // Light gray for disabled in dark mode
		}
	} else {
		if m.ragEnabled {
			statusColor = lipgloss.Color("2") // Green for enabled in light mode
		} else {
			statusColor = lipgloss.Color("1") // Red for disabled in light mode
		}
	}

	content := []string{
		"RAG Settings",
		"",
		"Status: " + lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(statusText),
		"",
		toggleText,
		"",
		"Controls:",
		"Enter - Toggle RAG On/Off",
		"Tab - Switch to Chat",
	}

	m.ragViewport.SetContent(strings.Join(content, "\n"))
}

func (m *model) updateInputPlaceholder() {
	switch m.activeTab {
	case chatTab:
		if m.connectionValid {
			m.textarea.Placeholder = "Send a message..."
		} else {
			m.textarea.Placeholder = "Configure Ollama URL in Settings tab first"
		}
	case modelsTab:
		if m.connectionValid {
			m.textarea.Placeholder = "Type model name or command..."
		} else {
			m.textarea.Placeholder = "Configure Ollama URL in Settings tab first"
		}
	case ragTab:
		if m.connectionValid {
			m.textarea.Placeholder = "RAG configuration..."
		} else {
			m.textarea.Placeholder = "Configure Ollama URL in Settings tab first"
		}
	case settingsTab:
		m.textarea.Placeholder = "Enter Ollama URL (e.g., http://localhost:11434)"
	}
}

func (m *model) updateSettingsViewportContent() {
	connectionStatus := "✗ DISCONNECTED"
	var connectionColor lipgloss.Color
	statusMessage := "Enter Ollama server URL in the input field below."

	if m.connectionValid {
		connectionStatus = "✓ CONNECTED"
		if m.darkMode {
			connectionColor = darkModeAccentColor // Goldenrod for connected in dark mode
		} else {
			connectionColor = lipgloss.Color("2") // Green for connected in light mode
		}
		statusMessage = "Connection established! Press Enter to edit URL if needed."
	} else {
		if m.darkMode {
			connectionColor = darkModeTextColor // Light gray for disconnected in dark mode
		} else {
			connectionColor = lipgloss.Color("1") // Red for disconnected in light mode
		}
	}

	currentURL := m.settings.OllamaURL
	if currentURL == "" {
		currentURL = "(not configured)"
	}

	content := []string{
		"Ollama Server URL",
		"",
		"Current URL: " + currentURL,
		"Connection: " + lipgloss.NewStyle().Foreground(connectionColor).Bold(true).Render(connectionStatus),
		"",
		statusMessage,
		"Press Enter to test the connection.",
		"",
		"Example: http://localhost:11434",
		"",
		"Controls:",
		"Enter - Edit/Test URL",
		"Tab - Switch tabs",
	}

	m.settingsViewport.SetContent(strings.Join(content, "\n"))
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
		modelsViewportWidth := 30 // Default width
		if m.bot.ModelManager != nil {
			modelsViewportWidth = min(msg.Width-4, m.bot.ModelManager.MaxModelNameLength()+10)
		}

		// Settings viewport width - make it wider to accommodate URLs
		settingsViewportWidth := min(msg.Width-4, 60) // Larger width for settings

		m.viewport.Width = chatViewportWidth
		m.modelsViewport.Width = modelsViewportWidth
		m.ragViewport.Width = modelsViewportWidth
		m.settingsViewport.Width = settingsViewportWidth
		m.textarea.SetWidth(chatViewportWidth)

		// Height calculations (accounting for tab header)
		tabHeaderHeight := 3 // Height of tab header
		availableHeight := msg.Height - m.textarea.Height() - lipgloss.Height(gap) - tabHeaderHeight

		m.viewport.Height = availableHeight
		m.modelsViewport.Height = availableHeight
		m.ragViewport.Height = availableHeight
		m.settingsViewport.Height = availableHeight

		if m.bot.MessageLen() > 0 {
			// Wrap content before setting it.
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.bot.MessageManager.StyledMessages(), "\n")))
		}
		m.viewport.GotoBottom()
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			// Only allow tab switching if connection is valid OR if we're currently on settings tab
			if !m.connectionValid && m.activeTab != settingsTab {
				m.inputError = "Please configure Ollama URL in Settings tab first"
				return m, nil
			}

			// Switch between tabs
			switch m.activeTab {
			case chatTab:
				m.activeTab = modelsTab
				m.focus = focusModelsViewport
				m.textarea.Blur()
				// Fetch models when switching to models tab
				if m.connectionValid && m.bot.ModelManager != nil {
					m.models = m.bot.ModelManager.ModelNames()
				}
			case modelsTab:
				m.activeTab = ragTab
				m.focus = focusRAGViewport
				m.textarea.Blur()
			case ragTab:
				m.activeTab = settingsTab
				m.focus = focusSettingsViewport
				m.textarea.Blur()
			case settingsTab:
				// Only allow leaving settings if connection is valid
				if m.connectionValid {
					m.activeTab = chatTab
					m.focus = focusTextarea
					m.textarea.Focus()
				} else {
					m.inputError = "Please configure a valid Ollama URL before leaving Settings"
					return m, nil
				}
			}
			m.updateModelsViewportContent()
			m.updateRAGViewportContent()
			m.updateSettingsViewportContent()
			m.updateInputPlaceholder()
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
				case settingsTab:
					m.focus = focusSettingsViewport
					m.textarea.Blur()
				}
			}
			m.updateSettingsViewportContent()
			m.updateInputPlaceholder()
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
					if m.bot.ModelManager != nil && len(m.models) > 0 {
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
					}
				} else if m.activeTab == ragTab && m.focus == focusRAGViewport {
					// Toggle RAG enabled/disabled
					m.ragEnabled = !m.ragEnabled
					// Save RAG state to settings
					m.settings.SetRAGEnabled(m.ragEnabled)
					m.updateTabNames()
					m.updateRAGViewportContent()
				} else if m.activeTab == settingsTab && m.focus == focusSettingsViewport {
					// On settings tab with viewport focus, switch to textarea focus for URL editing
					m.focus = focusTextarea
					m.textarea.Focus()
					// Pre-fill with current URL if any for editing
					if m.settings.OllamaURL != "" {
						m.textarea.SetValue(m.settings.OllamaURL)
					}
					m.updateInputPlaceholder()
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
						if !m.connectionValid {
							m.inputError = "Please configure Ollama URL in Settings tab first"
							m.textarea.Reset()
							return m, nil
						}
						m.activeTab = chatTab
						m.textarea.Reset()
						return m, nil
					case "/models":
						if !m.connectionValid {
							m.inputError = "Please configure Ollama URL in Settings tab first"
							m.textarea.Reset()
							return m, nil
						}
						m.activeTab = modelsTab
						// Fetch models when switching via command
						if m.bot.ModelManager != nil {
							m.models = m.bot.ModelManager.ModelNames()
						}
						m.updateTabNames()
						m.updateModelsViewportContent()
						m.textarea.Reset()
						return m, nil
					case "/rag":
						if !m.connectionValid {
							m.inputError = "Please configure Ollama URL in Settings tab first"
							m.textarea.Reset()
							return m, nil
						}
						m.activeTab = ragTab
						m.textarea.Reset()
						return m, nil
					case "/settings":
						m.activeTab = settingsTab
						m.textarea.Reset()
						return m, nil
					case "/dark":
						// Toggle dark mode
						m.darkMode = !m.darkMode
						m.settings.SetDarkMode(m.darkMode)
						m.applyTheme() // Apply the new theme
						m.textarea.Reset()
						return m, nil
					case "/clear":
						// /clear only works on chat tab
						if m.activeTab == chatTab {
							m.bot.ClearMessages()
							m.viewport.SetContent(`Welcome to Gollama-Chat!
Type a message and press Enter to send.` + ascii + `

Use the Tab key to switch between Chat, Models, RAG, and Settings tabs.
Use Ctrl+T to focus the input field for commands from any tab.

Special commands:
  • /clear - clear chat history
  • /chat - switch to chat tab
  • /models - switch to models tab
  • /rag - switch to RAG tab
  • /settings - switch to settings tab
  • /dark - toggle dark mode
  • /exit or /quit - quit application

Key bindings:
  • Ctrl+U - clear input
  • Ctrl+A - go to start
  • Ctrl+E - go to end
  • Ctrl+C or Esc - quit`)
							// Update tab names to reflect cleared tokens (should be 0 now)
							m.updateTabNames()
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
					if m.activeTab == settingsTab {
						// Settings tab: treat input as URL to test
						m.inputError = "" // Clear any existing errors

						// Test the connection to the entered URL
						err := bot.TestConnection(input)
						if err != nil {
							// Connection failed - don't save the invalid URL
							// Keep the existing connectionValid state and URL
							errorMsg := err.Error()
							if strings.Contains(errorMsg, "no such host") {
								m.inputError = "Invalid hostname or server not reachable - keeping current URL"
							} else if strings.Contains(errorMsg, "connection refused") {
								m.inputError = "Connection refused - check if Ollama is running - keeping current URL"
							} else if strings.Contains(errorMsg, "timeout") {
								m.inputError = "Connection timeout - server not responding - keeping current URL"
							} else {
								m.inputError = "Connection failed - please check URL - keeping current URL"
							}
							// Don't change connectionValid or save the bad URL
						} else {
							// Connection successful - save URL and initialize bot
							m.settings.SetOllamaURL(input)
							m.connectionValid = true

							// Initialize model manager with the new URL
							defaultModel := "tinyllama:latest"
							if m.settings.LastModel != "" {
								defaultModel = m.settings.LastModel
							}

							err := m.bot.InitializeModelManager(input, defaultModel)
							if err != nil {
								m.inputError = "Failed to initialize models: " + err.Error()
								// Don't change connectionValid if we had a working URL before
								// The HTTP connection worked, but model initialization failed
							} else {
								// Success! Update everything
								m.models = m.bot.ModelManager.ModelNames()
								m.updateTabNames()
								m.updateModelsViewportContent()

								// Switch to chat tab now that we're connected
								m.activeTab = chatTab
								m.focus = focusTextarea
								m.textarea.Focus()
							}
						}

						m.textarea.Reset()
						m.updateTabNames() // Update tab names to reflect current connection status
						m.updateSettingsViewportContent()
						m.updateInputPlaceholder()
						return m, nil
					} else if m.activeTab != chatTab {
						// Non-settings, non-chat tab with non-command input
						m.inputError = "Chat input detected. Please switch to the chat tab and press enter"
						return m, nil // Don't clear textarea for this case
					} else {
						// Valid chat input - clear any existing error
						m.inputError = ""

						// Check if we have a valid connection and ModelManager before sending
						if !m.connectionValid || m.bot.ModelManager == nil {
							m.inputError = "No connection to Ollama server. Please configure URL in Settings tab."
							m.textarea.Reset()
							return m, nil
						}

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
						// Update tab names to reflect new token count after sending message
						m.updateTabNames()
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
	// Get theme-aware tab styles
	inactiveStyle, activeStyle, inactiveModelsStyle, activeModelsStyle,
		inactiveRAGStyle, activeRAGStyle, inactiveSettingsStyle, activeSettingsStyle := m.getTabStyles()

	// Render tabs
	var renderedTabs []string

	for i, tabName := range m.tabs {
		var style lipgloss.Style
		isFirst, isLast, isActive := i == 0, i == len(m.tabs)-1, i == int(m.activeTab)

		// Apply styles based on tab type and state
		if i == 1 { // Models tab
			if isActive {
				style = activeModelsStyle
			} else {
				style = inactiveModelsStyle
			}
		} else if i == 2 { // RAG tab
			if isActive {
				style = activeRAGStyle
			} else {
				style = inactiveRAGStyle
			}
		} else if i == 3 { // Settings tab
			if isActive {
				style = activeSettingsStyle
			} else {
				style = inactiveSettingsStyle
			}
		} else { // Chat tab
			if isActive {
				style = activeStyle
			} else {
				style = inactiveStyle
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
	} else if m.activeTab == ragTab {
		// RAG tab: show RAG viewport centered
		content = lipgloss.NewStyle().
			Align(lipgloss.Center).
			Width(m.viewport.Width).
			Render(m.ragViewport.View())
	} else { // Settings tab
		// Settings tab: show settings viewport centered
		content = lipgloss.NewStyle().
			Align(lipgloss.Center).
			Width(m.viewport.Width).
			Render(m.settingsViewport.View())
	}

	// Handle error message display without affecting layout spacing
	var errorDisplay string
	var adjustedGap = gap
	if m.inputError != "" {
		var errorColor lipgloss.Color
		if m.darkMode {
			errorColor = darkModeTextColor // Light gray for errors in dark mode
		} else {
			errorColor = lipgloss.Color("196") // Red for errors in light mode
		}

		errorStyle := lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true).
			Padding(0, 1)
		errorDisplay = errorStyle.Render("⚠ "+m.inputError) + "\n"
		adjustedGap = "\n" // Reduce gap since we're adding error message line
	}

	// Combine tabs, content, error message, and textarea
	return tabRow + "\n" + content + adjustedGap + errorDisplay + m.textarea.View()
}
