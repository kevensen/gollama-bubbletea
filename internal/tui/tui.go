package tui

import (
	"context"
	"fmt"
	"slices"
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

type focus int

const (
	focusTextarea focus = iota
	focusModelsViewport
	focusToolsViewport
)

type model struct {
	viewport       viewport.Model
	modelsViewport viewport.Model
	toolsViewport  viewport.Model
	textarea       textarea.Model
	senderStyle    lipgloss.Style
	bot            *bot.Bot
	err            error
	responseBuffer string
	focus          focus
	models         []string
	tools          []string
	selectedModel  int
	selectedTool   int
	toolsEnabled   bool
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
Type a message and press Enter to send.` + ascii + "\n\n Use the Tab key to switch between the models and tools viewports.  Use ctrl+c or esc to quit.")

	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	modelsVp := viewport.New(20, 5)

	modelsVp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("1")) // Red border

	toolsVp := viewport.New(20, 5)
	toolsVp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("1")) // Red border

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return &model{
		textarea:       ta,
		viewport:       vp,
		modelsViewport: modelsVp,
		toolsViewport:  toolsVp,
		senderStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		bot:            b,
		err:            nil,
		focus:          focusTextarea,
		selectedModel:  0,
		selectedTool:   0,
		toolsEnabled:   false, // Initialize selectedTool to 0
	}
}

func (m *model) handleChatResponse(resp llm.Answer) error {
	m.responseBuffer += resp.Message.Content
	if resp.Done && m.responseBuffer != "" && resp.Message.ToolCalls == nil {
		m.bot.MessageManager.AddMessage(resp.Message)
		m.responseBuffer = ""
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.bot.MessageManager.StyledMessages(), "\n")))
		m.viewport.GotoBottom()
	}
	if resp.Message.ToolCalls != nil {
		for _, toolCall := range resp.Message.ToolCalls {
			if toolCall.Error != nil {
				msg := llm.Message{Role: "error", Content: toolCall.Error.Error()}
				m.bot.MessageManager.AddMessage(msg)
				continue
			}
			callable, ok := m.bot.ToolManager.ByName(toolCall.Function.Name)
			if !ok {
				msg := llm.Message{Role: "error", Content: fmt.Sprintf("tool not found: %s", toolCall.Function.Name)}
				m.bot.MessageManager.AddMessage(msg)
				continue
			}
			ans, err := callable.Call(toolCall.Function.Arguments)
			if err != nil {
				msg := llm.Message{Role: "error", Content: err.Error()}
				m.bot.MessageManager.AddMessage(msg)
				continue
			}
			ansMessage := fmt.Sprintf("The answer from the tool named %s is %s", toolCall.Function.Name, ans)
			msg := llm.Message{Role: "system", Content: ansMessage}

			m.bot.MessageManager.AddMessage(msg)
		}
		m.responseBuffer = ""
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.bot.MessageManager.StyledMessages(), "\n")))
		m.viewport.GotoBottom()
	}
	return nil
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.fetchModels, m.fetchTools)
}

func (m *model) fetchModels() tea.Msg {
	models := m.bot.ModelManager.ModelNames()
	m.models = models
	m.updateModelsViewportContent()
	return nil
}

func (m *model) fetchTools() tea.Msg {
	m.tools = []string{"Disabled", "Enabled"}
	m.updateToolsViewportContent()
	return nil
}

func (m *model) updateModelsViewportContent() {
	currentModel := m.bot.ModelManager.CurrentModel()
	styledModels := []string{"Models:"}
	for i, model := range m.models {
		style := lipgloss.NewStyle()
		if model == currentModel {
			style = style.Foreground(lipgloss.Color("2")) // Highlight current model in green
		}
		if i == m.selectedModel && m.focus == focusModelsViewport {
			style = style.Background(lipgloss.Color("7")).Foreground(lipgloss.Color("0")) // Highlight selected model
		}
		styledModels = append(styledModels, style.Render(model))
	}
	m.modelsViewport.SetContent(strings.Join(styledModels, "\n"))
}

func (m *model) updateToolsViewportContent() {
	tools := []string{"Disabled", "Enabled"}
	slices.Sort(tools)

	styledTools := []string{"Tools:"}
	for i, tool := range tools {
		style := lipgloss.NewStyle()

		if tool == "Disabled" && !m.toolsEnabled {
			style = style.Foreground(lipgloss.Color("2")) // Highlight current tool in green
		} else if tool == "Enabled" && m.toolsEnabled {
			style = style.Foreground(lipgloss.Color("2")) // Highlight current tool in green
		}

		if i == m.selectedTool && m.focus == focusToolsViewport {
			style = style.Background(lipgloss.Color("7")).Foreground(lipgloss.Color("0")) // Highlight selected tool
		}

		styledTools = append(styledTools, style.Render(tool))
	}
	m.toolsViewport.SetContent(strings.Join(styledTools, "\n"))
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		mvCmd tea.Cmd
		tlCmd tea.Cmd
	)

	if m.focus == focusTextarea {
		m.textarea, tiCmd = m.textarea.Update(msg)
	}
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.modelsViewport, mvCmd = m.modelsViewport.Update(msg)
	m.toolsViewport, tlCmd = m.toolsViewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		modelsViewportWidth := m.bot.ModelManager.MaxModelNameLength() // 2 for padding
		modelsViewportWidth += 2
		chatViewportWidth := msg.Width - modelsViewportWidth - 10 // 10 for padding and border

		m.modelsViewport.Width = modelsViewportWidth
		m.toolsViewport.Width = modelsViewportWidth
		m.viewport.Width = chatViewportWidth
		m.textarea.SetWidth(chatViewportWidth)
		m.viewport.Height = msg.Height - m.textarea.Height() - lipgloss.Height(gap)

		toolAndModelAreaHeight := msg.Height - m.textarea.Height() - lipgloss.Height(gap)

		m.modelsViewport.Height = (toolAndModelAreaHeight / 2) - 2
		m.toolsViewport.Height = (toolAndModelAreaHeight / 2) - 2

		if m.bot.MessageLen() > 0 {
			// Wrap content before setting it.
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.bot.MessageManager.StyledMessages(), "\n")))
		}
		m.viewport.GotoBottom()
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			switch m.focus {
			case focusModelsViewport:
				m.focus = focusToolsViewport
			case focusToolsViewport:
				m.focus = focusTextarea
			case focusTextarea:
				m.focus = focusModelsViewport
			}
			m.updateModelsViewportContent()
			m.updateToolsViewportContent()
		case "up":
			if m.focus == focusModelsViewport && m.selectedModel > 0 {
				m.selectedModel--
			} else if m.focus == focusTextarea {
				m.viewport.LineUp(1)
			} else if m.focus == focusToolsViewport && m.selectedTool > 0 {
				m.selectedTool--
			}
			m.updateModelsViewportContent()
			m.updateToolsViewportContent()
		case "down":
			if m.focus == focusModelsViewport && m.selectedModel < len(m.models)-1 {
				m.selectedModel++
			} else if m.focus == focusTextarea {
				m.viewport.LineDown(1)
			} else if m.focus == focusToolsViewport && m.selectedTool < 1 {
				m.selectedTool++
			}

			m.updateModelsViewportContent()
			m.updateToolsViewportContent()
		case "enter":
			if m.focus == focusModelsViewport {
				selectedModel := m.models[m.selectedModel]
				err := m.bot.ModelManager.UseModel(selectedModel)
				if err != nil {
					msg := llm.Message{Role: "error", Content: err.Error()}
					m.bot.MessageManager.AddMessage(msg)
				}
				m.updateModelsViewportContent()
				m.updateToolsViewportContent()
			} else if m.focus == focusToolsViewport {
				if m.selectedTool == 0 {
					m.toolsEnabled = false
				} else {
					m.toolsEnabled = true
				}
				m.updateModelsViewportContent()
				m.updateToolsViewportContent()
			} else if m.textarea.Value() != "" {
				if m.textarea.Value() == "exit" {
					return m, tea.Quit
				}
				m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.bot.MessageManager.StyledMessages(), "\n")))
				ctx := context.Background()
				ans, err := m.bot.SendMessage(ctx, "user", m.textarea.Value(), m.toolsEnabled)
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

	return m, tea.Batch(tiCmd, vpCmd, mvCmd, tlCmd)
}

func (m *model) View() string {
	// Stack modelsViewport and toolsViewport vertically
	sideView := lipgloss.JoinVertical(lipgloss.Top, m.modelsViewport.View(), m.toolsViewport.View())

	// Join the side view with the main viewport horizontally
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		sideView,
		"  ", // Padding between viewports
		m.viewport.View(),
	) + gap + m.textarea.View()
}
