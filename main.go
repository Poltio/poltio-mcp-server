package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Poltio/poltio-mcp-server/client"
	"github.com/Poltio/poltio-mcp-server/tools"
)

func main() {
	token := os.Getenv("POLTIO_API_TOKEN")
	if token == "" {
		log.Fatal("POLTIO_API_TOKEN environment variable is required")
	}
	orgID := os.Getenv("POLTIO_ORG_ID")
	if orgID == "" {
		log.Fatal("POLTIO_ORG_ID environment variable is required")
	}

	c := client.New(token, orgID)

	s := server.NewMCPServer("poltio", "1.0.0")

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

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
