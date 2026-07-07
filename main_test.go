package main

import (
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestDestructiveOption(t *testing.T) {
	tool := mcp.NewTool("delete_x",
		mcp.WithDescription("Deletes x."),
		destructive(),
	)
	if !strings.HasSuffix(tool.Description, destructiveWarning) {
		t.Errorf("description missing warning: %q", tool.Description)
	}
	if !strings.HasPrefix(tool.Description, "Deletes x.") {
		t.Errorf("original description lost: %q", tool.Description)
	}
	if tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint {
		t.Error("DestructiveHint annotation not set to true")
	}
}
