# wachat

Convert WhatsApp chat exports to readable text with automatic voice message transcription.

## Installation

### Homebrew

```bash
brew install joern1811/tap/wachat
```

### From source

```bash
go install github.com/joern1811/wachat/cmd@latest
```

## Configuration

The quickest way to set up wachat is:

```bash
wachat init
```

This interactively prompts for your OpenAI API key, validates it, and writes the config file.

Alternatively, wachat reads the API key (required for voice message transcription) from:

1. Environment variable `OPENAI_API_KEY`
2. Config file `~/.config/wachat/config.json`:

```json
{
  "openai_api_key": "sk-..."
}
```

The XDG config directory (`$XDG_CONFIG_HOME/wachat/`) is respected.

## Usage

```bash
# Basic usage — process a WhatsApp export
wachat "WhatsApp Chat - John.zip"

# Filter by date range
wachat --from 01.01.2024 --to 31.12.2024 export.zip

# Output as markdown to a file
wachat -f markdown -o chat.md export.zip

# Preview which API calls would be made
wachat --dry-run export.zip

# Show version
wachat version
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--from` | | Start time filter (`DD.MM.YYYY` or `DD.MM.YYYY HH:MM`) |
| `--to` | | End time filter (`DD.MM.YYYY` or `DD.MM.YYYY HH:MM`) |
| `--output` | `-o` | Output file (default: stdout) |
| `--format` | `-f` | Output format: `text` or `markdown` (default: `text`) |
| `--dry-run` | | Show what API calls would be made without executing them |

## License

[MIT](LICENSE)
