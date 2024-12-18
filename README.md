# Discuss - Interactive Content Analysis Tool

A command-line tool that enables AI-assisted analysis and discussion of any text content, code, or direct questions.

## Features

- Interactive chat mode for ongoing discussions
- Single prompt mode for quick analysis
- Flexible input handling (files, clipboard, or direct questions)
- Clean, readable terminal output with timestamps
- Local LLM integration via Ollama

## Prerequisites

- Go 1.x
- [Ollama](https://ollama.ai/) running locally with the `qwen2.5-coder` model
- Make (for building and installation)

## Installation

```bash
git clone https://github.com/storbeck/discuss
cd discuss
make
sudo make install
```

## Usage

### Interactive Mode (-it)

Start an interactive chat session:
```bash
discuss -it
```

Analyze a file interactively:
```bash
cat main.go | discuss -it
```

### Single Prompt Mode (-p)

Quick analysis without entering chat mode:
```bash
# Analyze a file with a specific question
cat main.go | discuss -p "What could be improved in this code?"

# Direct questions without file input
discuss -p "What's the best way to handle errors in Go?"

# Pipe in git diff for review
git diff | discuss -p "Review these changes and suggest improvements"
```

## Examples

1. Code Review:
```bash
git show | discuss -p "Review this commit and suggest improvements"
```

2. Log Analysis:
```bash
tail -n 100 error.log | discuss -it
```

3. Documentation Help:
```bash
discuss -p "How do I write a good README.md file?"
```

## Configuration

### Environment Variables

- `OLLAMA_HOST`: URL of your Ollama instance (default: `http://localhost:11434`)

You can set environment variables in two ways:

1. Set in your shell:
```bash
export OLLAMA_HOST=http://192.168.1.213:11434
```

2. Set when running the command:
```bash
OLLAMA_HOST=http://192.168.1.213:11434 discuss -it
```

## License

MIT
