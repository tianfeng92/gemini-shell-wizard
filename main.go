package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"google.golang.org/genai"
)

// Define the System Prompt
const SystemPromptBase = `You are an expert Command Line Interface (CLI) assistant.
Rules:
1. Be concise.
2. If the user asks for a command, provide it in a markdown code block (e.g., ` + "```bash" + ` ... ` + "```" + `).
3. Context provided below describes the user's current environment.`

// File to cache environment info
var cacheFile = filepath.Join(os.Getenv("HOME"), ".gemini-env")

func main() {
	// 1. Setup API
	apiKey := os.Getenv("GEMINI_SHELL_API_KEY")
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "Error: GEMINI_SHELL_API_KEY environment variable not set.\n")
		os.Exit(1)
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	// 2. Get Environment Info (Cached)
	envInfo := getOrUpdateEnv()

	// 3. Read Arguments (User Question)
	userPrompt := strings.Join(os.Args[1:], " ")

	// 4. Read Stdin (Piped Context)
	var stdinContent string
	stat, _ := os.Stdin.Stat()
	isPiped := (stat.Mode() & os.ModeCharDevice) == 0
	if isPiped {
		bytes, _ := io.ReadAll(os.Stdin)
		stdinContent = string(bytes)
	}

	if userPrompt == "" && stdinContent == "" {
		fmt.Println("Usage: command | >>> [question]")
		fmt.Println("   or: >>> [question]")
		return
	}

	if userPrompt == "" && stdinContent != "" {
		userPrompt = "Explain this output and suggest a fix if there is an error."
	}

	// 5. Construct the final prompt
	var fullPrompt string
	contextBlock := fmt.Sprintf("System Info:\n%s", envInfo)
	if stdinContent != "" {
		fullPrompt = fmt.Sprintf("%s\n\n%s\n\nInput Context:\n%s\n\nUser Question:\n%s", SystemPromptBase, contextBlock, stdinContent, userPrompt)
	} else {
		fullPrompt = fmt.Sprintf("%s\n\n%s\n\nUser Question:\n%s", SystemPromptBase, contextBlock, userPrompt)
	}

	// 6. Call Gemini
	resp, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash", genai.Text(fullPrompt), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating content: %v\n", err)
		os.Exit(1)
	}

	// 7. Print Output
	geminiResponse := ""
	if resp != nil {
		geminiResponse = resp.Text()
	}
	fmt.Print("\n\033[1;34mGemini:\033[0m ")
	fmt.Println(geminiResponse)
	fmt.Println()

	// 8. Extract and Propose Execution
	commands := extractCommands(geminiResponse)
	if len(commands) > 0 {
		confirmAndExecute(commands)
	}
}

// --- Environment Diagnosis ---

func getOrUpdateEnv() string {
	// Try reading cache first
	content, err := os.ReadFile(cacheFile)
	if err == nil && len(content) > 0 {
		return string(content)
	}

	// Generate fresh info
	info := generateEnvInfo()

	// Save to cache
	_ = os.WriteFile(cacheFile, []byte(info), 0644)
	return info
}

func generateEnvInfo() string {
	var info strings.Builder
	info.WriteString(fmt.Sprintf("OS: %s\n", runtime.GOOS))
	info.WriteString(fmt.Sprintf("Architecture: %s\n", runtime.GOARCH))
	info.WriteString(fmt.Sprintf("Shell: %s\n", os.Getenv("SHELL")))

	// Try to get Linux Distro details
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/etc/os-release"); err == nil {
			info.WriteString("OS Release Info:\n")
			// Grab PRETTY_NAME for conciseness
			re := regexp.MustCompile(`PRETTY_NAME="(.*?)"`)
			match := re.FindStringSubmatch(string(data))
			if len(match) > 1 {
				info.WriteString(match[1] + "\n")
			} else {
				info.WriteString(string(data) + "\n") // Fallback
			}
		}
	} else if runtime.GOOS == "darwin" {
		out, _ := exec.Command("sw_vers").Output()
		info.WriteString("MacOS Version:\n" + string(out))
	}

	return info.String()
}

// --- Command Handling ---

func extractCommands(text string) []string {
	// Regex to find content inside ```bash or ```sh blocks, or just ```
	// We are generous to catch most shell commands
	re := regexp.MustCompile("(?s)```(?:bash|sh|zsh)?\\n(.*?)\\n```")
	matches := re.FindAllStringSubmatch(text, -1)

	var cmds []string
	for _, match := range matches {
		if len(match) > 1 {
			// Trim whitespace and split multiple lines if they look like separate commands
			block := strings.TrimSpace(match[1])
			if block != "" {
				cmds = append(cmds, block)
			}
		}
	}
	return cmds
}

func confirmAndExecute(cmds []string) {
	// Open /dev/tty for user interaction because os.Stdin might be exhausted if piped
	tty, err := os.Open("/dev/tty")
	if err != nil {
		// Fallback to os.Stdin if TTY isn't available (rare)
		tty = os.Stdin
	}
	defer tty.Close()
	scanner := bufio.NewScanner(tty)

	fmt.Println("\033[1;33mSUGGESTED COMMAND(S):\033[0m")
	for i, cmd := range cmds {
		fmt.Printf("[%d] %s\n", i+1, cmd)
	}

	fmt.Print("\n\033[1;33mDo you want to execute these commands? [y/N]: \033[0m")

	if scanner.Scan() {
		input := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if input == "y" || input == "yes" {
			for _, cmdStr := range cmds {
				fmt.Printf("\n\033[1;32mExecuting:\033[0m %s\n", cmdStr)

				// Run the command using the user's shell
				shell := os.Getenv("SHELL")
				if shell == "" {
					shell = "sh"
				}

				cmd := exec.Command(shell, "-c", cmdStr)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Stdin = os.Stdin // Allow the command to be interactive (e.g. vim, sudo)

				if err := cmd.Run(); err != nil {
					fmt.Printf("\033[1;31mCommand failed:\033[0m %v\n", err)
					break // Stop sequence on error
				}
			}
		} else {
			fmt.Println("Aborted.")
		}
	}
}
