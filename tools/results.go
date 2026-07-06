package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

func AddResult(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		title, err := req.RequireString("title")
		if err != nil || title == "" {
			return nil, fmt.Errorf("title is required")
		}
		body := map[string]any{"title": title}
		if v := req.GetString("desc", ""); v != "" {
			body["desc"] = v
		}
		if v := req.GetString("background", ""); v != "" {
			body["background"] = v
		}
		if v := req.GetString("alt", ""); v != "" {
			body["alt"] = v
		}
		if v := req.GetString("url", ""); v != "" {
			body["url"] = v
		}
		if v := req.GetString("url_text", ""); v != "" {
			body["url_text"] = v
		}
		if v := req.GetInt("min_c", -1); v >= 0 {
			body["min_c"] = v
		}
		if v := req.GetInt("max_c", -1); v >= 0 {
			body["max_c"] = v
		}
		if v := req.GetString("luv", ""); v != "" {
			body["luv"] = v
		}
		if v := req.GetString("search", ""); v != "" {
			body["search"] = v
		}
		if v := req.GetString("search2", ""); v != "" {
			body["search2"] = v
		}
		if v := req.GetInt("is_default", -1); v >= 0 {
			body["is_default"] = v
		}
		if v := req.GetString("secondary_url", ""); v != "" {
			body["secondary_url"] = v
		}
		if v := req.GetString("secondary_url_text", ""); v != "" {
			body["secondary_url_text"] = v
		}
		if v := req.GetString("source_id", ""); v != "" {
			body["source_id"] = v
		}
		data, err := c.Post("/platform/content/"+publicID+"/result", body)
		if err != nil {
			return nil, fmt.Errorf("add_result: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateResult(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		resultID, err := req.RequireInt("result_id")
		if err != nil {
			return nil, fmt.Errorf("result_id is required")
		}
		body := map[string]any{}
		if v := req.GetString("title", ""); v != "" {
			body["title"] = v
		}
		if v := req.GetString("desc", ""); v != "" {
			body["desc"] = v
		}
		if v := req.GetString("background", ""); v != "" {
			body["background"] = v
		}
		if v := req.GetString("alt", ""); v != "" {
			body["alt"] = v
		}
		if v := req.GetString("url", ""); v != "" {
			body["url"] = v
		}
		if v := req.GetString("url_text", ""); v != "" {
			body["url_text"] = v
		}
		if v := req.GetInt("min_c", -1); v >= 0 {
			body["min_c"] = v
		}
		if v := req.GetInt("max_c", -1); v >= 0 {
			body["max_c"] = v
		}
		if v := req.GetString("luv", ""); v != "" {
			body["luv"] = v
		}
		if v := req.GetString("search", ""); v != "" {
			body["search"] = v
		}
		if v := req.GetString("search2", ""); v != "" {
			body["search2"] = v
		}
		if v := req.GetInt("is_default", -1); v >= 0 {
			body["is_default"] = v
		}
		if v := req.GetString("secondary_url", ""); v != "" {
			body["secondary_url"] = v
		}
		if v := req.GetString("secondary_url_text", ""); v != "" {
			body["secondary_url_text"] = v
		}
		if v := req.GetString("source_id", ""); v != "" {
			body["source_id"] = v
		}
		path := "/platform/content/" + publicID + "/result/" + strconv.Itoa(resultID)
		data, err := c.Put(path, body)
		if err != nil {
			return nil, fmt.Errorf("update_result: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteResult(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		resultID, err := req.RequireInt("result_id")
		if err != nil {
			return nil, fmt.Errorf("result_id is required")
		}
		path := "/platform/content/" + publicID + "/result/" + strconv.Itoa(resultID)
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("delete_result: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// SetAnswerResultPoint updates the point value linking an answer to a result
// (used in score-based quizzes and calculator tests).
func SetAnswerResultPoint(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		questionID, err := req.RequireInt("question_id")
		if err != nil {
			return nil, fmt.Errorf("question_id is required")
		}
		answerID, err := req.RequireInt("answer_id")
		if err != nil {
			return nil, fmt.Errorf("answer_id is required")
		}
		resultID, err := req.RequireInt("content_result_id")
		if err != nil {
			return nil, fmt.Errorf("content_result_id is required")
		}
		point, err := req.RequireInt("point")
		if err != nil {
			return nil, fmt.Errorf("point is required")
		}
		path := "/platform/content/" + publicID +
			"/question/" + strconv.Itoa(questionID) +
			"/answer/" + strconv.Itoa(answerID) + "/results"
		data, err := c.Put(path, map[string]any{
			"content_result_id": resultID,
			"point":             point,
		})
		if err != nil {
			return nil, fmt.Errorf("set_answer_result_point: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
