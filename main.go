package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Poltio/poltio-mcp-server/client"
	"github.com/Poltio/poltio-mcp-server/tools"
)

var version = "dev"

type orgEntry struct {
	ID int `json:"id"`
}

func main() {
	token := os.Getenv("POLTIO_API_TOKEN")
	if token == "" {
		log.Fatal("POLTIO_API_TOKEN environment variable is required")
	}

	c := client.New(token)

	data, err := c.GetOrganizations()
	if err != nil {
		log.Fatalf("failed to fetch organizations at startup: %v", err)
	}
	var orgs []orgEntry
	if err := json.Unmarshal(data, &orgs); err != nil {
		log.Fatalf("failed to parse organizations at startup: %v", err)
	}
	if len(orgs) == 0 {
		log.Fatal("no organizations found for this token — ensure your account belongs to at least one organization")
	}
	c.SetOrgID(strconv.Itoa(orgs[0].ID))

	s := server.NewMCPServer("poltio", version)

	s.AddTool(mcp.NewTool("list_content",
		mcp.WithDescription("List Poltio content (polls, quizzes, tests) with optional pagination and filtering."),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("per_page", mcp.Description("Results per page (default: 12)")),
		mcp.WithString("type", mcp.Description("Filter by type: poll, set, test, quiz, this-that")),
		mcp.WithString("q", mcp.Description("Search query against title and description")),
		mcp.WithString("order", mcp.Description("Sort field: created_at (default), updated_at, vote_count, voter_count, type, id, end_date")),
		mcp.WithString("sort", mcp.Description("Sort direction: desc (default) or asc")),
	), tools.ListContent(c))

	s.AddTool(mcp.NewTool("get_content",
		mcp.WithDescription("Get a single Poltio content item with its metrics."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
	), tools.GetContent(c))

	s.AddTool(mcp.NewTool("create_content",
		mcp.WithDescription("Create a new Poltio content item. The item starts as a draft — call publish_content to make it live."),
		mcp.WithString("type", mcp.Description("Content type: poll, set, test, quiz, this-that"), mcp.Required()),
		mcp.WithString("title", mcp.Description("End-user facing title"), mcp.Required()),
		mcp.WithString("desc", mcp.Description("Cover screen description (optional)")),
		mcp.WithString("name", mcp.Description("Internal non-public name (optional)")),
	), tools.CreateContent(c))

	s.AddTool(mcp.NewTool("publish_content",
		mcp.WithDescription("Publish a draft Poltio content item, making it publicly accessible."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
	), tools.PublishContent(c))

	s.AddTool(mcp.NewTool("list_drafts",
		mcp.WithDescription("List unpublished (draft) Poltio content items."),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("per_page", mcp.Description("Results per page (default: 12)")),
		mcp.WithString("type", mcp.Description("Filter by type: poll, set, test, quiz, this-that")),
		mcp.WithString("q", mcp.Description("Search query against title and description")),
	), tools.ListDrafts(c))

	s.AddTool(mcp.NewTool("list_organizations",
		mcp.WithDescription("List Poltio organizations the current user belongs to, including their role in each."),
	), tools.ListOrganizations(c))

	s.AddTool(mcp.NewTool("switch_organization",
		mcp.WithDescription("Switch the active organization context. All subsequent tool calls will operate under the selected organization."),
		mcp.WithNumber("id", mcp.Description("Organization ID (from list_organizations)"), mcp.Required()),
	), tools.SwitchOrganization(c))

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
