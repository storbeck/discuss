package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Bold(true)

	botStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			Bold(true)

	timestampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	subtle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
)

func main() {
	// Add command line flags
	interactive := flag.Bool("it", false, "Enable interactive chat mode")
	prompt := flag.String("p", "", "Single prompt to analyze content with")
	flag.Parse()

	var initialMessages []Message

	// Check for input from stdin (now optional)
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// There is stdin input
		content := readFromStdin()

		if *interactive {
			fmt.Printf("%s\r\n", subtle.Render(fmt.Sprintf("Content loaded: %d lines, %d characters",
				len(strings.Split(content, "\n")),
				len(content))))
		}

		initialMessages = []Message{
			{
				Role:    "user",
				Content: fmt.Sprintf("Content to analyze:\n%s", content),
			},
		}
	}

	if *interactive {
		runInteractiveMode(initialMessages)
	} else if *prompt != "" {
		// Single prompt mode
		messages := append(initialMessages, Message{
			Role:    "user",
			Content: *prompt,
		})

		response, err := sendPromptWithHistory(messages)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		// Just print the response directly without chat formatting
		fmt.Println(response)
	} else {
		fmt.Println("Error: Must specify either -it for interactive mode or -p for single prompt mode")
		return
	}
}

// Add this new function to handle interactive mode
func runInteractiveMode(initialMessages []Message) {
	messages := initialMessages

	// Get the first user message before starting the chat loop
	tty, err := os.Open("/dev/tty")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening /dev/tty: %v\n", err)
		return
	}
	defer tty.Close()

	reader := bufio.NewReader(tty)

	fmt.Println(subtle.Render("What would you like to know?"))

	// Show prompt for first message
	timestamp := timestampStyle.Render(time.Now().Format("15:04"))
	fmt.Printf("\n%s %s ", timestamp, userStyle.Render("<you>"))

	firstInput, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		return
	}
	firstInput = strings.TrimSpace(firstInput)
	if firstInput == "" {
		return
	}

	// Show thinking message
	fmt.Printf("%s %s %s\n",
		timestampStyle.Render(time.Now().Format("15:04")),
		botStyle.Render("<bot>"),
		subtle.Render("thinking..."))

	// Get first response
	response, err := sendPromptWithHistory(messages)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Move up one line and clear it before printing response
	fmt.Print("\033[1A\033[K")

	botMsg := Message{Role: "assistant", Content: response}
	messages = append(messages, botMsg)
	printMessage(botMsg)

	// Main chat loop
	for {
		timestamp := timestampStyle.Render(time.Now().Format("15:04"))
		fmt.Printf("\n%s %s ", timestamp, userStyle.Render("<you>"))

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		userMsg := Message{Role: "user", Content: input}
		messages = append(messages, userMsg)

		fmt.Printf("%s %s %s\n",
			timestampStyle.Render(time.Now().Format("15:04")),
			botStyle.Render("<bot>"),
			subtle.Render("thinking..."))

		response, err := sendPromptWithHistory(messages)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Print("\033[1A\033[K")

		botMsg := Message{Role: "assistant", Content: response}
		messages = append(messages, botMsg)
		printMessage(botMsg)
	}
}

func printMessage(msg Message) {
	timestamp := timestampStyle.Render(time.Now().Format("15:04"))
	switch msg.Role {
	case "user":
		fmt.Printf("%s %s %s\n",
			timestamp,
			userStyle.Render("<you>"),
			msg.Content)
	case "assistant":
		fmt.Printf("%s %s %s",
			timestamp,
			botStyle.Render("<bot>"),
			msg.Content)
	}
}

// Message type and other functions remain the same
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func sendPromptWithHistory(messages []Message) (string, error) {
	// Get Ollama host from environment variable, default to localhost if not set
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}
	url := ollamaHost + "/api/generate"
	var responseText bytes.Buffer

	// Format messages for Ollama
	var prompt strings.Builder
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			prompt.WriteString("Human: " + msg.Content + "\n")
		case "assistant":
			prompt.WriteString("Assistant: " + msg.Content + "\n")
		}
	}

	payload := map[string]interface{}{
		"model":  "qwen2.5-coder",
		"prompt": prompt.String(),
		"stream": true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling JSON: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		type ResponseChunk struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
		}
		var chunk ResponseChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			fmt.Println("Error parsing chunk:", err)
			continue
		}

		responseText.WriteString(chunk.Response)

		if chunk.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading response stream: %w", err)
	}

	return responseText.String(), nil
}

func readFromStdin() string {
	var content strings.Builder
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		content.WriteString(scanner.Text() + "\n")
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	if content.Len() == 0 {
		fmt.Println("No input received")
		os.Exit(1)
	}

	return content.String()
}
