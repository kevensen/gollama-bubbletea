package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kevensen/gollama-bubbletea/internal/bot"
	"github.com/ollama/ollama/api"
)

const gap = "\n\n"

type errMsg error

type focus int

const (
	focusTextarea focus = iota
	focusModelsViewport
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
Type a message and press Enter to send.`)

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
	}
}

func (m *model) handleChatResponse(resp api.ChatResponse) error {
	m.responseBuffer += resp.Message.Content
	if resp.Done {
		m.bot.AddMessage(resp.Message.Role, m.responseBuffer)
		m.responseBuffer = ""
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(RenderMessages(m.bot.RenderMessages()), "\n")))
		m.viewport.GotoBottom()

	}
	return nil
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.fetchModels)
}

func (m *model) fetchModels() tea.Msg {
	ctx := context.Background()
	models, err := m.bot.Models(ctx)
	if err != nil {
		return errMsg(err)
	}

	m.models = models
	m.updateModelsViewportContent()
	return nil
}

func (m *model) updateModelsViewportContent() {
	currentModel := m.bot.CurrentModel()
	var styledModels []string
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

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		mvCmd tea.Cmd
	)

	if m.focus == focusTextarea {
		m.textarea, tiCmd = m.textarea.Update(msg)
	}
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.modelsViewport, mvCmd = m.modelsViewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		modelsViewportWidth := msg.Width / 10
		chatViewportWidth := msg.Width - modelsViewportWidth - 10 // 10 for padding and border

		m.modelsViewport.Width = modelsViewportWidth
		m.viewport.Width = chatViewportWidth
		m.textarea.SetWidth(chatViewportWidth)
		m.viewport.Height = msg.Height - m.textarea.Height() - lipgloss.Height(gap)
		m.modelsViewport.Height = msg.Height - m.textarea.Height() - lipgloss.Height(gap)

		if m.bot.MessageLen() > 0 {
			// Wrap content before setting it.
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(RenderMessages(m.bot.RenderMessages()), "\n")))
		}
		m.viewport.GotoBottom()
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if m.focus == focusTextarea {
				m.focus = focusModelsViewport
			} else {
				m.focus = focusTextarea
			}
			m.updateModelsViewportContent()
		case "up":
			if m.focus == focusModelsViewport && m.selectedModel > 0 {
				m.selectedModel--
				m.updateModelsViewportContent()
			}
		case "down":
			if m.focus == focusModelsViewport && m.selectedModel < len(m.models)-1 {
				m.selectedModel++
				m.updateModelsViewportContent()
			}
		case "enter":
			if m.focus == focusModelsViewport {
				selectedModel := m.models[m.selectedModel]
				m.bot.UseModel(context.Background(), selectedModel)
				m.updateModelsViewportContent()
			} else if m.textarea.Value() != "" {
				// m.messages = append(m.messages, m.senderStyle.Render("You: ")+m.textarea.Value())
				m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(RenderMessages(m.bot.RenderMessages()), "\n")))
				ctx := context.Background()
				var resp api.ChatResponseFunc = m.handleChatResponse
				err := m.bot.SendMessage(ctx, "user", m.textarea.Value(), resp)
				if err != nil {
					// m.messages = append(m.messages, fmt.Sprintf("Error: %v", err))
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
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.modelsViewport.View(),
		"  ", // Padding between viewports
		m.viewport.View(),
	) + gap + m.textarea.View()
}

func RenderMessages(msg []api.Message) []string {
	var messages []string
	for _, m := range msg {
		role := strings.ToUpper(m.Role[:1]) + strings.ToLower(m.Role[1:])
		var roleStyled string
		if m.Role == "user" {
			roleStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(role) // Green for user
		} else if m.Role == "assistant" {
			roleStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(role) // Yellow for assistant
		} else {
			roleStyled = role
		}
		messages = append(messages, roleStyled+": "+m.Content, "\n")
	}
	return messages
}
