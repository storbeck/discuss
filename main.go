package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Add styles
var (
	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Bold(true)

	botStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			Bold(true)

	timestampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Bold(false)

	subtle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// Model represents the UI state
type Model struct {
	viewport    viewport.Model
	textarea    textarea.Model
	messages    []Message
	content     string
	ready       bool
	err         error
	initialized bool
	loading     bool
}

// Initialize the model
func initialModel(content string) Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message and press Enter"
	ta.Focus()
	ta.ShowLineNumbers = false
	ta.Prompt = "│ "
	ta.CharLimit = 280
	ta.SetWidth(30)
	ta.SetHeight(1)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()

	// Create initial message about the content
	initialMessages := []Message{
		{
			Role: "system",
			Content: fmt.Sprintf("Content loaded: %d lines, %d characters\nReady to discuss the content. What would you like to know?",
				len(strings.Split(content, "\n")),
				len(content)),
		},
	}

	return Model{
		textarea:    ta,
		messages:    initialMessages,
		content:     content,
		err:         nil,
		initialized: false,
		loading:     false,
	}
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if !m.initialized {
				m.initialized = true
				m.loading = true
				hiddenMessage := Message{
					Role:    "user",
					Content: fmt.Sprintf("Here is the content to analyze:\n\n%s", m.content),
				}
				visibleMessage := Message{
					Role:    "user",
					Content: m.textarea.Value(),
				}

				m.messages = append(m.messages, visibleMessage)

				messagesToSend := append([]Message{}, m.messages[:len(m.messages)-1]...)
				messagesToSend = append(messagesToSend, hiddenMessage, visibleMessage)

				m.textarea.Reset()
				m.viewport.SetContent(formatMessages(m.messages))
				m.viewport.GotoBottom()
				return m, sendMessage(messagesToSend)
			} else if m.textarea.Value() != "" {
				m.loading = true
				userMessage := Message{Role: "user", Content: m.textarea.Value()}
				m.messages = append(m.messages, userMessage)
				m.textarea.Reset()
				m.viewport.SetContent(formatMessages(m.messages))
				m.viewport.GotoBottom()
				return m, sendMessage(m.messages)
			}
		}

	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-m.textarea.Height()-2)
			m.viewport.SetContent(formatMessages(m.messages))
			m.viewport.GotoBottom()
			m.textarea.SetWidth(msg.Width)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - m.textarea.Height() - 2
			m.textarea.SetWidth(msg.Width)
		}

	case responseMsg:
		m.loading = false
		response := msg
		if response.error != nil {
			m.messages = append(m.messages, Message{
				Role:    "system",
				Content: fmt.Sprintf("Error: %v", response.error),
			})
		} else {
			m.messages = append(m.messages, Message{
				Role:    "assistant",
				Content: response.content,
			})
		}
		m.viewport.SetContent(formatMessages(m.messages))
		m.viewport.GotoBottom()
		return m, nil

	case tea.MouseMsg:
		switch {
		case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonWheelUp:
			m.viewport.LineUp(3)
		case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonWheelDown:
			m.viewport.LineDown(3)
		}
	}

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m Model) View() string {
	if !m.ready {
		return "\nInitializing..."
	}

	var view strings.Builder
	view.WriteString(m.viewport.View())
	if m.loading {
		view.WriteString(fmt.Sprintf("\n%s %s %s",
			timestampStyle.Render(time.Now().Format("15:04")),
			botStyle.Render("<bot>"),
			subtle.Render("is typing...")))
	}
	view.WriteString("\n" + m.textarea.View())

	return view.String()
}

// Format messages for display
func formatMessages(messages []Message) string {
	var b strings.Builder
	for _, msg := range messages {
		timestamp := timestampStyle.Render(time.Now().Format("15:04"))
		switch msg.Role {
		case "user":
			b.WriteString(fmt.Sprintf("%s %s %s\n",
				timestamp,
				userStyle.Render("<you>"),
				msg.Content))
		case "assistant":
			b.WriteString(fmt.Sprintf("%s %s %s\n",
				timestamp,
				botStyle.Render("<bot>"),
				msg.Content))
		case "system":
			b.WriteString(fmt.Sprintf("%s %s\n",
				subtle.Render("---"),
				subtle.Render(msg.Content)))
		}
	}
	return b.String()
}

// Custom message types for tea.Msg
type responseMsg struct {
	content string
	error   error
}

func sendMessage(messages []Message) tea.Cmd {
	return func() tea.Msg {
		content, err := sendPromptWithHistory(messages)
		return responseMsg{content: content, error: err}
	}
}

func sendInitialMessage(messages []Message) tea.Cmd {
	return func() tea.Msg {
		content, err := sendPromptWithHistory(messages)
		return responseMsg{content: content, error: err}
	}
}

func main() {
	// Check for input from stdin
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		fmt.Println("No input detected. Usage: cat file.txt | ./analyze")
		return
	}

	content := readFromStdin()
	// Enable mouse support when creating the program
	p := tea.NewProgram(
		initialModel(content),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithMouseAllMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

// ResponseChunk represents each streamed response JSON object
type ResponseChunk struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Add new type for conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func sendPromptWithHistory(messages []Message) (string, error) {
	url := "http://192.168.1.213:11434/api/generate"
	var responseText bytes.Buffer

	// Modified context to be more general-purpose
	conversationContext := "You are an intelligent assistant analyzing documents and files. Help understand and extract insights from the content. Keep responses clear and informative. Here's the conversation:\n\n"

	for _, msg := range messages {
		conversationContext += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	payload := map[string]interface{}{
		"model":  "qwen2.5-coder",
		"prompt": conversationContext,
		"stream": true,
	}

	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling JSON: %w", err)
	}

	// Send the POST request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Process the response stream
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		// Parse each line as a JSON chunk
		var chunk ResponseChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			fmt.Println("Error parsing chunk:", err)
			continue
		}

		// Print and store the response content
		responseText.WriteString(chunk.Response)

		// Stop if done
		if chunk.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading response stream: %w", err)
	}

	return responseText.String(), nil
}

// readFromStdin reads all input piped into stdin and returns it as a string
func readFromStdin() string {
	var code bytes.Buffer
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		code.WriteString(scanner.Text() + "\n")
	}
	return code.String()
}

// readPromptFile reads a prompt template from the given file path
func readPromptFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var content bytes.Buffer
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		content.WriteString(scanner.Text() + "\n")
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return content.String(), nil
}
