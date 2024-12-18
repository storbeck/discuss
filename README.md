# Discuss - Interactive Content Analysis Tool

A command-line tool that enables interactive discussions about code or text content using AI assistance.

## Features

- Interactive TUI (Terminal User Interface) powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- Real-time AI responses using local LLM server (Ollama)
- Support for scrolling through conversation history
- Stylish interface with distinct user and bot messages
- Mouse wheel support for navigation

## Prerequisites

- Go 1.x
- [Ollama](https://ollama.ai/) running locally with the `qwen2.5-coder` model
- Make (for building and installation)

## Installation

```bash
# Clone the repository
git clone https://github.com/storbeck/discuss
cd discuss

# Build and install
make
sudo make install
```

The installation will:
- Build the binary
- Install it to `/usr/local/bin/discuss`

To uninstall:
```bash
make uninstall
```

## Usage

The tool reads input from stdin, making it perfect for analyzing files:

```bash
cat file.txt | discuss
```

### Controls

- Type your message and press `Enter` to send
- Use mouse wheel or trackpad to scroll through conversation history
- Press `Ctrl+C` or `Esc` to quit
- Up to 280 characters per message

## Building from Source

```bash
# Build only
make build

# Clean build artifacts
make clean

# Build and install
make install

# Uninstall
make uninstall
```

## License

MIT