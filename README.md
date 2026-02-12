# Gemini-Shell-Wizard

A lightweight, context-aware AI assistant for your Zsh terminal. It detects your OS, suggests commands, and helps debug errors.

## ⚙️ How It Works

1. **Environment Diagnosis:** On first run, the tool detects your OS (Ubuntu/Arch/macOS), Shell (Zsh/Bash), and CPU architecture.
2. **Smart Caching:** This info is saved to `~/.gemini-env` so it doesn't slow down future commands.
3. **Context Injection:** When you ask a question (`>>> ...`), the tool reads this cached file and prepends it to the prompt.
  * *Result:* Gemini knows to suggest `apt` for Ubuntu or `brew` for macOS without you asking.
4. **Command Execution:** The tool parses Gemini's response for code blocks and asks for your confirmation before running anything.

## ⚡ Quick Setup

### 1. Build & Install

```bash
cd gemini-shell-wizard
make
```

### 2. Configure Zsh

Add this to your `~/.zshrc`:

```bash
export GEMINI_SHELL_API_KEY="your_api_key_here"

# Wrapper
function gemini-wizard() { ~/bin/gemini-shell-wizard-bin "$@"; }

# Widget (Handles >>>)
function magic-enter() {
  if [[ "$BUFFER" == \>\>\>* ]]; then
    print -s "$BUFFER"  # Save to history
    BUFFER="gemini-wizard \"${BUFFER:3}\""
  fi
  zle accept-line
}
zle -N magic-enter
bindkey "^M" magic-enter

# Pipe alias
alias gem="gemini-wizard"

```

Then run `source ~/.zshrc`.

## 🚀 Usage

**1. Ask Questions**

Use `>>>` to ask anything. It knows your OS (Ubuntu/Mac/etc).

```bash
>>> how do I kill a process on port 8080?

```

**2. Debug Errors**

Pipe output to `gem` to explain or fix it.

⚠️ Conflict Warning: This aliases `gem` to Gemini. If you are a Ruby developer, this will break your package manager. Then again, if you're a Ruby developer, you're probably used to things breaking. (Use \gem to bypass).

```bash
cat error.log | gem fix this

```

**3. Execute Commands**

Gemini suggests commands. You confirm them with `y` before they run.

```text
SUGGESTED COMMAND(S):
[1] sudo lsof -i :8080 | xargs kill
Do you want to execute these commands? [y/N]:

```
