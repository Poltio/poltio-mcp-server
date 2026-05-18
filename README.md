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
**Content**

| Tool | Description |
|---|---|
| `list_content` | List polls/quizzes/tests with pagination, type filter, and search |
| `get_content` | Get a single content item with its metrics |
| `create_content` | Create a new poll/quiz/test (starts as draft) |
| `update_content` | Update title, description, or name of a content item |
| `delete_content` | Permanently delete a content item |
| `publish_content` | Publish a draft content item |
| `duplicate_content` | Duplicate a content item into a new draft |
| `list_drafts` | List unpublished content items |
| `get_content_results` | Get paginated vote results (per-answer counts) |
| `get_content_sessions` | Get paginated user sessions |
| `get_content_metrics` | Get time-series metrics grouped by day/week/month/year |

**Questions**

| Tool | Description |
|---|---|
| `add_question` | Add a question to a draft content item |
| `update_question` | Update an existing question |
| `delete_question` | Delete a question |
| `get_question_order` | Get current question positions |
| `update_question_order` | Reorder questions (JSON array of `{id,position}`) |
| `get_content_conditions` | List questions that have display conditions |
| `add_question_condition` | Add an answer as a display condition on a question |
| `remove_question_condition` | Remove a single answer from a question's conditions |
| `clear_question_conditions` | Remove all display conditions from a question |
| `get_question_inputs` | Get paginated free-text vote inputs for a question |

**Answers**

| Tool | Description |
|---|---|
| `add_answer` | Add a single answer to a question |
| `add_answers_bulk` | Add multiple answers (one per line) in one call |
| `update_answer` | Update an existing answer |
| `delete_answer` | Delete an answer |
| `clone_answers` | Copy answers from one question to another |
| `get_answer_order` | Get current answer positions |
| `update_answer_order` | Reorder answers (JSON array of `{id,position}`) |

**Results** (quiz/test outcome screens)

| Tool | Description |
|---|---|
| `add_result` | Add a result screen to a quiz or test |
| `update_result` | Update an existing result screen |
| `delete_result` | Delete a result screen |
| `set_answer_result_point` | Set score-based point linking an answer to a result |

**Lead attachment**

| Tool | Description |
|---|---|
| `set_content_lead` | Attach a lead form to a content cover screen |
| `remove_content_lead` | Remove lead form from a content cover screen |
| `set_question_lead` | Attach a lead form to all answers of a question |
| `remove_question_lead` | Remove lead form from a question |
| `set_answer_lead` | Attach a lead form to a specific answer |
| `remove_answer_lead` | Remove lead form from a specific answer |
| `set_result_lead` | Attach a lead form to a result screen |
| `remove_result_lead` | Remove lead form from a result screen |

**Votes & Stats**

| Tool | Description |
|---|---|
| `get_voters` | Get paginated list of voters for a content item |
| `get_conversion_time_stats` | Get conversion time-series stats (account-wide or per content) |

**Organizations**

| Tool | Description |
|---|---|
| `list_organizations` | List organizations the current user belongs to |
| `switch_organization` | Switch the active organization by ID |

## Running Tests

```bash
go test ./...
```
