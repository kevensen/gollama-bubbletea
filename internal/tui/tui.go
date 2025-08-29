package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kevensen/gollama-bubbletea/internal/bot"
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
	inactiveTabStyle       = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(highlightColor).Padding(0, 1)
	activeTabStyle         = inactiveTabStyle.Border(activeTabBorder, true)
	inactiveModelsTabStyle = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(greenColor).Padding(0, 1)
	activeModelsTabStyle   = inactiveModelsTabStyle.Border(activeTabBorder, true)
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
)

type tab int

const (
	chatTab tab = iota
	modelsTab
)

type model struct {
	viewport       viewport.Model
	modelsViewport viewport.Model
	textarea       textarea.Model
	senderStyle    lipgloss.Style
	bot            *bot.Bot
	err            error
	responseBuffer string
	focus          focus
	models         []string
	selectedModel  int
	activeTab      tab
	tabs           []string
}

func New(b *bot.Bot) *model {
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
Type a message and press Enter to send.` + ascii + "\n\n Use the Tab key to switch between Chat and Models tabs.\n Special commands: /clear (clear history), /exit (quit)\n Use ctrl+c or esc to quit.")

	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	modelsVp := viewport.New(20, 5)

	modelsVp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("1")) // Red border

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return &model{
		textarea:       ta,
		viewport:       vp,
		modelsViewport: modelsVp,
		senderStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		bot:            b,
		err:            nil,
		focus:          focusTextarea,
		selectedModel:  0,
		activeTab:      chatTab,
		tabs:           []string{"Chat", "Models"},
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
	m.tabs[0] = "Chat"
	m.tabs[1] = "Models: " + currentModel
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.fetchModels)
}

func (m *model) fetchModels() tea.Msg {
	models := m.bot.ModelManager.ModelNames()
	m.models = models
	m.updateTabNames()
	m.updateModelsViewportContent()
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

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		mvCmd tea.Cmd
	)

	if m.focus == focusTextarea && m.activeTab == chatTab {
		m.textarea, tiCmd = m.textarea.Update(msg)
	}
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.modelsViewport, mvCmd = m.modelsViewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Calculate full width for chat tab
		chatViewportWidth := msg.Width - 4 // 4 for padding and border

		// Models viewport width for models tab
		modelsViewportWidth := min(msg.Width-4, m.bot.ModelManager.MaxModelNameLength()+10)

		m.viewport.Width = chatViewportWidth
		m.modelsViewport.Width = modelsViewportWidth
		m.textarea.SetWidth(chatViewportWidth)

		// Height calculations (accounting for tab header)
		tabHeaderHeight := 3 // Height of tab header
		availableHeight := msg.Height - m.textarea.Height() - lipgloss.Height(gap) - tabHeaderHeight

		m.viewport.Height = availableHeight
		m.modelsViewport.Height = availableHeight

		if m.bot.MessageLen() > 0 {
			// Wrap content before setting it.
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.bot.MessageManager.StyledMessages(), "\n")))
		}
		m.viewport.GotoBottom()
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			// Switch between tabs
			if m.activeTab == chatTab {
				m.activeTab = modelsTab
				m.focus = focusModelsViewport
				m.textarea.Blur()
			} else {
				m.activeTab = chatTab
				m.focus = focusTextarea
				m.textarea.Focus()
			}
			m.updateModelsViewportContent()
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
			if m.activeTab == modelsTab && m.focus == focusModelsViewport {
				selectedModel := m.models[m.selectedModel]
				err := m.bot.ModelManager.UseModel(selectedModel)
				if err != nil {
					msg := llm.Message{Role: "error", Content: err.Error()}
					m.bot.MessageManager.AddMessage(msg)
				}
				m.updateTabNames()
				m.updateModelsViewportContent()
			} else if m.activeTab == chatTab && m.textarea.Value() != "" {
				input := m.textarea.Value()

				// Handle special commands
				if input == "/exit" {
					return m, tea.Quit
				} else if input == "/clear" {
					// Clear chat history and viewport
					m.bot.ClearMessages()
					m.viewport.SetContent(`Welcome to Gollama-Chat!
Type a message and press Enter to send.` + ascii + "\n\n Use the Tab key to switch between Chat and Models tabs.\n Special commands: /clear (clear history), /exit (quit)\n Use ctrl+c or esc to quit.")
					m.textarea.Reset()
					return m, nil
				}

				// Regular message handling
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
		case "ctrl+c", "esc":
			return m, tea.Quit
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd, mvCmd)
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
	} else {
		// Models tab: show models viewport centered
		content = lipgloss.NewStyle().
			Align(lipgloss.Center).
			Width(m.viewport.Width).
			Render(m.modelsViewport.View())
	}

	// Combine tabs, content, and textarea
	return tabRow + "\n" + content + gap + m.textarea.View()
}
