# Poltio MCP Server

MCP server that exposes [Poltio](https://poltio.com) platform content management as AI-accessible tools. Works with Claude Desktop and any MCP-compatible client.

## Prerequisites

- Go 1.21+
- A Poltio API token (from your account settings)
- Your Poltio organization ID (integer)

## Build

```bash
go build -o poltio-mcp-server .
```

## Configuration

Two environment variables are required at startup:

| Variable | Description |
|---|---|
| `POLTIO_API_TOKEN` | Bearer token — from Poltio account → Settings → Tokens |
| `POLTIO_ORG_ID` | Organization ID (integer) |

## Claude Desktop Integration

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "poltio": {
      "command": "/absolute/path/to/poltio-mcp-server",
      "env": {
        "POLTIO_API_TOKEN": "your-token-here",
        "POLTIO_ORG_ID": "42"
      }
    }
  }
}
```

Restart Claude Desktop after saving.

## Available Tools

| Tool | Description |
|---|---|
| `list_content` | List polls/quizzes/tests with pagination, type filter, and search |
| `get_content` | Get a single content item with its metrics by `public_id` |
| `create_content` | Create a new poll/quiz/test (starts as draft) |
| `publish_content` | Publish a draft content item |
| `list_drafts` | List unpublished content items |

## Running Tests

```bash
go test ./...
```
