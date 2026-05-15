# Poltio MCP Server

MCP server that exposes [Poltio](https://poltio.com) platform content management as AI-accessible tools. Works with Claude Desktop, Gemini, and any MCP-compatible client.

## Prerequisites

- Go 1.21+
- A Poltio API token (from your account settings)

## Build

```bash
go build -o poltio-mcp-server .
```

## Configuration

One environment variable is required at startup:

| Variable | Description |
|---|---|
| `POLTIO_API_TOKEN` | Bearer token — from Poltio account → Settings → Tokens |

The server automatically fetches your organizations at startup and activates the first one. Use the `list_organizations` and `switch_organization` tools to view and change the active organization.

## Claude Desktop Integration

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "poltio": {
      "command": "/absolute/path/to/poltio-mcp-server",
      "env": {
        "POLTIO_API_TOKEN": "your-token-here"
      }
    }
  }
}
```

Restart Claude Desktop after saving.

## Gemini Integration

Gemini supports MCP servers through **Gemini CLI** and **Google AI Studio** extensions.

### Gemini CLI

Install the Gemini CLI and add the server to your MCP config (`~/.gemini/settings.json`):

```json
{
  "mcpServers": {
    "poltio": {
      "command": "/absolute/path/to/poltio-mcp-server",
      "env": {
        "POLTIO_API_TOKEN": "your-token-here"
      }
    }
  }
}
```

### Google AI Studio

1. Open **Google AI Studio → Extensions → Add MCP Server**.
2. Set the binary path and environment variables as above.
3. Reload AI Studio to activate the tools.

## Available Tools

| Tool | Description |
|---|---|
| `list_content` | List polls/quizzes/tests with pagination, type filter, and search |
| `get_content` | Get a single content item with its metrics by `public_id` |
| `create_content` | Create a new poll/quiz/test (starts as draft) |
| `publish_content` | Publish a draft content item |
| `list_drafts` | List unpublished content items |
| `list_organizations` | List organizations the current user belongs to |
| `switch_organization` | Switch the active organization by ID |

## Running Tests

```bash
go test ./...
```
