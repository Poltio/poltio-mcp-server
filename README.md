# Poltio MCP Server

MCP server that exposes [Poltio](https://poltio.com) platform content management as AI-accessible tools. Works with Claude Desktop, Gemini, and any MCP-compatible client.

## Install in Claude Desktop

1. Download **`poltio.mcpb`** from the [latest release](https://github.com/Poltio/poltio-mcp-server/releases/latest).
2. Double-click it. Claude Desktop opens an install dialog — click **Install**.
3. Paste your API token when prompted (Poltio → **Settings → Tokens**), then enable the extension.

Use a long-lived API token, not a short-lived session token — the latter stops working within a day.

The bundle covers macOS (Intel and Apple Silicon) and Windows. On Linux, or for any other MCP client, use the manual setup below.

> The macOS binary is unsigned (no Apple Developer ID), so the bundle clears its own quarantine flag on first launch. Without that, macOS terminates it with no error message.

## Build from source

Requires Go 1.21+ and a Poltio API token.

```bash
go build -o poltio-mcp-server .
```

To rebuild the installer bundle (macOS only — needs `lipo`):

```bash
scripts/build-mcpb.sh v1.2.3
```

## Configuration

| Variable | Description |
|---|---|
| `POLTIO_API_TOKEN` | Bearer token — from Poltio account → Settings → Tokens. Required in stdio mode; ignored for authenticated HTTP callers |
| `PORT` | When set, starts an HTTP server on that port instead of stdio |
| `POLTIO_API_BASE_URL` | Poltio API to talk to. Defaults to the production API |
| `MCP_PUBLIC_URL` | This server's own public URL, published as the OAuth resource identifier. Defaults to `https://mcp.poltio.com` |

The server automatically fetches your organizations at startup and activates the first one. Use the `list_organizations` and `switch_organization` tools to view and change the active organization.

## HTTP Mode (for web UIs and Smithery)

Set `PORT` to start the server as an HTTP/Streamable-HTTP server instead of a stdio process:

```bash
POLTIO_API_TOKEN=your-token PORT=8080 ./poltio-mcp-server
```

The MCP endpoint is available at `http://localhost:8080/mcp`.

In HTTP mode every request must carry its own `Authorization: Bearer <token>` header, and each token gets its own client and active organization. Requests without one are rejected with `401` and a `WWW-Authenticate` header pointing at the OAuth metadata below, which is what lets an MCP client start an authorization flow.

The token can be either a personal API token from Settings → Tokens or an OAuth access token issued by the Poltio API — the API accepts both on the same endpoints.

### OAuth discovery

The server publishes protected resource metadata (RFC 9728) at `/.well-known/oauth-protected-resource`:

```json
{
  "resource": "https://mcp.poltio.com",
  "authorization_servers": ["https://api.poltio.com"]
}
```

`resource` comes from `MCP_PUBLIC_URL` and `authorization_servers` from `POLTIO_API_BASE_URL`, so both must be set correctly for a client to complete the flow.

### Publishing to Smithery

1. Deploy the server to any host with a public HTTPS URL (Fly.io, Railway, Render, VPS).
2. Set `PORT` and `MCP_PUBLIC_URL` in the deployment environment. `POLTIO_API_TOKEN` is not needed — callers supply their own.
3. Go to [smithery.ai](https://smithery.ai) → Publish → enter your server URL (`https://your-host.com/mcp`).
4. Users connecting through Smithery must configure `Authorization: Bearer <token>` in the Smithery connection settings.

## Claude Desktop (manual setup)

Only needed if you are not using the `poltio.mcpb` bundle above. Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

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
| `get_content_edit` | Get full editable content object including all questions, answers, results, and conditions |
| `create_content` | Create a new poll/quiz/test (starts as draft) |
| `update_content` | Update a content item's metadata, images and options (title and type are carried over automatically when omitted) |
| `delete_content` | Permanently delete a content item |
| `publish_content` | Publish a draft content item |
| `duplicate_content` | Duplicate a content item into a new draft |
| `list_drafts` | List unpublished content items |
| `list_templates` | List available Poltio content templates |
| `get_template` | Get a single content template with all its data |
| `use_template` | Clone a content template into a new draft content item |
| `get_content_results` | Get paginated vote results (per-answer counts) |
| `get_content_sessions` | Get paginated user sessions |
| `sync_content_sessions` | Export sessions cursor-paginated with votes, results, clicks and conversions, filterable by `updated_after` for incremental syncs |
| `get_content_result_metrics` | Per-result view count, click count and CTR over a date range |
| `get_content_metrics` | Get time-series metrics grouped by day/week/month/year |
| `get_content_stats` | Get aggregate stat totals for a content item for a date range (device-filterable) |
| `get_vote_sources` | Get paginated vote sources (referring URLs) for a content item |
| `get_sankey` | Get Sankey diagram data showing user flow through a content item |
| `get_sankey_users` | Get users who took a specific path in the Sankey diagram |
| `get_searchable_fields` | Get all searchable and filterable fields for a searchable content item |
| `get_session_urls` | Get session URLs grouped by URL with session counts |
| `upload_image` | Upload an image (<= 5 MB; png, jpg, jpeg, gif, or webp) via a local `image_path`, an `image_url`, or `image_base64`; returns a file path for use in content, questions, answers, or results. Prefer `image_path`/`image_url` — the server reads the bytes directly instead of passing them through the model |

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

**Leads**

| Tool | Description |
|---|---|
| `list_leads` | List lead campaigns for this organization |
| `create_lead` | Create a new lead campaign |
| `get_lead` | Get a single lead campaign by ID |
| `update_lead` | Update a lead campaign |
| `delete_lead` | Delete a lead campaign |
| `get_lead_inputs` | Get paginated user inputs submitted through a lead form |
| `get_lead_logs` | Get paginated activation logs for a lead campaign |
| `get_lead_codes` | Get paginated coupon codes for a lead campaign |
| `add_lead_codes` | Add one or more coupon codes to a lead campaign (one per line) |
| `delete_all_lead_codes` | Remove all coupon codes from a lead campaign |
| `update_lead_code` | Update a single coupon code |
| `delete_lead_code` | Delete a single coupon code |

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

**Pixel Codes**

| Tool | Description |
|---|---|
| `list_pixel_codes` | List pixel code snippets for this organization |
| `create_pixel_code` | Create a new pixel code snippet (iframe, img, or script tag HTML) |
| `update_pixel_code` | Update an existing pixel code snippet |
| `delete_pixel_code` | Delete a pixel code snippet |
| `set_content_pixel_code` | Attach a pixel code to the cover screen of a content item |
| `remove_content_pixel_code` | Remove the pixel code from the cover screen of a content item |
| `set_question_pixel_code` | Attach a pixel code to all answers of a question |
| `remove_question_pixel_code` | Remove the pixel code from all answers of a question |
| `set_answer_pixel_code` | Attach a pixel code to a specific answer |
| `remove_answer_pixel_code` | Remove the pixel code from a specific answer |
| `set_result_pixel_code` | Attach a pixel code to a result screen (fires on result view) |
| `remove_result_pixel_code` | Remove the view pixel code from a result screen |
| `set_result_click_pixel_code` | Attach a pixel code to a result screen's click/CTA action |
| `remove_result_click_pixel_code` | Remove the click pixel code from a result screen |

**Themes**

| Tool | Description |
|---|---|
| `list_themes` | List themes for this organization |
| `get_default_theme` | Get the default theme values to use as a base when creating a new theme |
| `get_theme` | Get a single theme by ID |
| `create_theme` | Create a new theme (call `get_default_theme` first to discover available fields) |
| `update_theme` | Update an existing theme's fields |
| `find_theme` | Auto-extract a theme (colors, fonts) from an existing web page URL |
| `delete_theme` | Delete a theme (fails if the theme is currently in use) |

**Dashboard**

| Tool | Description |
|---|---|
| `get_dashboard` | Get account dashboard data including recent content and aggregate counters |
| `get_dashboard_summary` | Get most recently active content stat summary |
| `get_dashboard_metrics` | Get account-wide time-series metrics grouped by period |
| `get_dashboard_stats` | Get account-wide aggregate stat totals for a date range (device-filterable) |

**Sheet Hooks**

| Tool | Description |
|---|---|
| `list_sheet_hooks` | List Google Sheet hooks for this organization |
| `create_sheet_hook` | Create a new Google Sheet hook to stream votes in real time |
| `get_sheet_hook` | Get details of a Google Sheet hook |
| `update_sheet_hook` | Update an existing Google Sheet hook |
| `delete_sheet_hook` | Delete a Google Sheet hook |
| `get_sheet_hook_logs` | Get execution logs for a Google Sheet hook |

**Webhooks**

| Tool | Description |
|---|---|
| `list_webhooks` | List webhooks for this organization |
| `create_webhook` | Create a webhook to receive vote/session data in real time |
| `get_webhook` | Get details of a webhook |
| `update_webhook` | Update an existing webhook |
| `delete_webhook` | Delete a webhook |
| `get_webhook_logs` | Get execution logs for a webhook |

**Votes & Stats**

| Tool | Description |
|---|---|
| `get_voters` | Get paginated list of voters for a content item |
| `get_conversion_time_stats` | Get conversion time-series stats (account-wide or per content) |

**Reports**

| Tool | Description |
|---|---|
| `list_reports` | List downloadable report requests |
| `create_report` | Request a new downloadable report (sent to your account email) |

**Data Sources** — the panel's *Sources*

| Tool | Description |
|---|---|
| `list_data_sources` | List data sources connected to this account |
| `create_data_source` | Submit a new data source (XML/JSON feed URL) |
| `create_csv_data_source` | Create a data source from a CSV file in one step |
| `create_xml_data_source` | Create an XML data source with its repeating item node set |
| `get_data_source` | Get one data source with its status, elements and feed analysis |
| `update_data_source` | Rename a data source or point it at a different feed |
| `get_data_source_attributes` | Read the analysed feed columns, samples and suggested mappings |
| `refresh_data_source_format` | Re-analyse the feed after it changed |
| `set_data_source_elements` | Map feed columns to Poltio element types |
| `get_data_source_elements` | List the saved element mappings with their ids |
| `update_data_source_element` | Change one element mapping |
| `delete_data_source_element` | Remove one element mapping |
| `publish_data_source` | Queue the import (mark ready) once the mapping is complete |
| `get_data_source_items` | Get the imported items, paginated |
| `delete_data_source` | Remove a data source |
| `upload_data_source` | Upload a file (JSON, XML, CSV, or TXT) as a new data source |

**Product Finders** — a Source connected to a content

| Tool | Description |
|---|---|
| `list_product_finders` | List product finders (data source contents) |
| `get_product_finder` | Get one with its bound content and searchable fields |
| `create_product_finder` | Turn an imported data source into searchable content |
| `update_product_finder` | Update result rendering, pagination, filters or tracking |
| `delete_product_finder` | Delete a product finder |
| `add_product_finder_field` | Make a field searchable, filterable or sortable |
| `update_product_finder_field` | Change a searchable field |
| `delete_product_finder_field` | Remove a searchable field |

**Domains**

| Tool | Description |
|---|---|
| `list_domains` | List custom domains configured for this account |
| `create_domain` | Add a new custom domain (requires DNS verification after creation) |
| `update_domain` | Update a custom domain's settings |
| `delete_domain` | Delete a custom domain |

**Widgets**

| Tool | Description |
|---|---|
| `list_widgets` | List your existing dynamic widgets |
| `create_widget` | Create a new dynamic widget |
| `get_widget` | Get a single dynamic widget |
| `update_widget` | Update an existing dynamic widget |
| `delete_widget` | Delete an existing dynamic widget |

**Settings**

| Tool | Description |
|---|---|
| `update_settings` | Update your account's username, email, or profile photo |
| `update_password` | Change your account password |
| `resend_verification` | Resend the account email verification link |
| `accept_terms` | Accept the Poltio terms and conditions |
| `setup_two_factor` | Begin two-factor authentication setup |
| `verify_two_factor` | Confirm 2FA setup with a TOTP verification code |
| `disable_two_factor` | Disable two-factor authentication on the current account |
| `reset_two_factor_recovery_codes` | Regenerate 2FA recovery codes (existing codes are invalidated) |

**Conversion Settings**

| Tool | Description |
|---|---|
| `list_conversion_settings` | List conversion tracking URLs defined for this account |
| `create_conversion_setting` | Add a new checkout success URL for conversion tracking |
| `update_conversion_setting` | Update an existing conversion tracking URL |
| `delete_conversion_setting` | Delete a conversion tracking URL |

**Organizations**

| Tool | Description |
|---|---|
| `list_organizations` | List organizations the current user belongs to, including their role |
| `switch_organization` | Switch the active organization context |
| `get_organization` | Get an organization's details including members and pending invites |
| `update_organization` | Update an organization's name |
| `invite_org_member` | Invite a new member to an organization via email |
| `join_organization` | Join an organization using an invite token |
| `leave_organization` | Leave an organization (not available to the owner) |
| `cancel_org_invite` | Cancel a pending organization invitation by email |
| `remove_org_member` | Remove a member from an organization |
| `update_org_member` | Update a member's role in an organization |
| `list_ip_rules` | List the IP access rules of an organization |
| `create_ip_rule` | Create an IP access rule (allow/block IPs or CIDR blocks) |
| `update_ip_rule` | Update an existing IP access rule |
| `delete_ip_rule` | Delete an IP access rule |

**Subscription**

| Tool | Description |
|---|---|
| `list_subscription_tiers` | List available subscription tiers and their features |
| `create_subscription` | Create a new subscription for the current organization |
| `cancel_subscription` | Cancel the current organization's subscription |

**Utilities**

| Tool | Description |
|---|---|
| `search_playground` | Test search queries and filters against a searchable content item |
| `check_snippet_page` | Check if a page URL has the Poltio snippet active in the last 48 hours |
| `create_short_link` | Create a polt.io shortened URL from any long URL |

## Running Tests

```bash
go test ./...
```
