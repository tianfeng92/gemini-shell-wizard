# Variables
BINARY_NAME=gemini-shell-wizard
INSTALL_NAME=gemini-shell-wizard-bin
INSTALL_DIR=$(HOME)/bin
SRC=main.go

# Phony targets to prevent conflicts with file names
.PHONY: all build install clean

# Default target
all: install

# Build the binary using Docker
build:
	docker run --rm \
		-v $(PWD):/workspace \
		-w /workspace \
		golang:1.25-alpine \
		sh -c "go mod download && go build -o $(BINARY_NAME) $(SRC)"

# Install the binary to the destination
install: build
	mkdir -p $(INSTALL_DIR)
	mv $(BINARY_NAME) $(INSTALL_DIR)/$(INSTALL_NAME)
	@echo "Installed to $(INSTALL_DIR)/$(INSTALL_NAME)"

# Clean up build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f $(INSTALL_DIR)/$(INSTALL_NAME)
