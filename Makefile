# Variables
BINARY_NAME=discuss
BINARY_DIR=bin
INSTALL_BIN_DIR=/usr/local/bin

# Go related variables
GOFILES=$(wildcard *.go)

# Build settings
LDFLAGS=-s -w

.PHONY: all clean build install uninstall

all: clean build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	@go build -ldflags "$(LDFLAGS)" -o $(BINARY_DIR)/$(BINARY_NAME) $(GOFILES)

clean:
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR)

install: build
	@echo "Installing $(BINARY_NAME)..."
	@install -m 755 $(BINARY_DIR)/$(BINARY_NAME) $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	@echo "Installation complete!"

uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	@echo "Uninstallation complete!" 