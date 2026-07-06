package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

func AddAnswer(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		questionID, err := req.RequireInt("question_id")
		if err != nil {
			return nil, fmt.Errorf("question_id is required")
		}
		body := map[string]any{}
		if v := req.GetString("title", ""); v != "" {
			body["title"] = v
		}
		if v := req.GetString("background", ""); v != "" {
			body["background"] = v
		}
		if v := req.GetString("alt", ""); v != "" {
			body["alt"] = v
		}
		if v := req.GetInt("has_right_answer", -1); v >= 0 {
			body["has_right_answer"] = v
		}
		if v := req.GetInt("is_right_answer", -1); v >= 0 {
			body["is_right_answer"] = v
		}
		if v := req.GetInt("is_mutually_exclusive", -1); v >= 0 {
			body["is_mutually_exclusive"] = v
		}
		if v := req.GetString("luv", ""); v != "" {
			body["luv"] = v
		}
		if v := req.GetString("cal_val", ""); v != "" {
			body["cal_val"] = v
		}
		if v := req.GetString("search_query", ""); v != "" {
			body["search_query"] = v
		}
		if v := req.GetString("search_filter", ""); v != "" {
			body["search_filter"] = v
		}
		if v := req.GetInt("position", -1); v >= 0 {
			body["position"] = v
		}
		if v := req.GetInt("max_vote", -1); v >= 0 {
			body["max_vote"] = v
		}
		if v := req.GetString("addon", ""); v != "" {
			body["addon"] = v
		}
		if v := req.GetString("disabled_msg", ""); v != "" {
			body["disabled_msg"] = v
		}
		path := "/platform/content/" + publicID + "/question/" + strconv.Itoa(questionID) + "/answer"
		data, err := c.Post(path, body)
		if err != nil {
			return nil, fmt.Errorf("add_answer: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func AddAnswersBulk(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		questionID, err := req.RequireInt("question_id")
		if err != nil {
			return nil, fmt.Errorf("question_id is required")
		}
		answers, err := req.RequireString("answers")
		if err != nil || answers == "" {
			return nil, fmt.Errorf("answers is required (one answer per line)")
		}
		body := map[string]any{
			"answers":         answers,
			"remove_existing": req.GetInt("remove_existing", 0) == 1,
		}
		path := "/platform/content/" + publicID + "/question/" + strconv.Itoa(questionID) + "/answer/multi"
		data, err := c.Post(path, body)
		if err != nil {
			return nil, fmt.Errorf("add_answers_bulk: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateAnswer(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		body := map[string]any{}
		if v := req.GetString("title", ""); v != "" {
			body["title"] = v
		}
		if v := req.GetString("background", ""); v != "" {
			body["background"] = v
		}
		if v := req.GetString("alt", ""); v != "" {
			body["alt"] = v
		}
		if v := req.GetInt("has_right_answer", -1); v >= 0 {
			body["has_right_answer"] = v
		}
		if v := req.GetInt("is_right_answer", -1); v >= 0 {
			body["is_right_answer"] = v
		}
		if v := req.GetInt("is_mutually_exclusive", -1); v >= 0 {
			body["is_mutually_exclusive"] = v
		}
		if v := req.GetString("luv", ""); v != "" {
			body["luv"] = v
		}
		if v := req.GetString("cal_val", ""); v != "" {
			body["cal_val"] = v
		}
		if v := req.GetString("search_query", ""); v != "" {
			body["search_query"] = v
		}
		if v := req.GetString("search_filter", ""); v != "" {
			body["search_filter"] = v
		}
		if v := req.GetInt("position", -1); v >= 0 {
			body["position"] = v
		}
		if v := req.GetInt("max_vote", -1); v >= 0 {
			body["max_vote"] = v
		}
		if v := req.GetString("addon", ""); v != "" {
			body["addon"] = v
		}
		if v := req.GetString("disabled_msg", ""); v != "" {
			body["disabled_msg"] = v
		}
		path := "/platform/content/" + publicID +
			"/question/" + strconv.Itoa(questionID) +
			"/answer/" + strconv.Itoa(answerID)
		data, err := c.Put(path, body)
		if err != nil {
			return nil, fmt.Errorf("update_answer: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteAnswer(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		path := "/platform/content/" + publicID +
			"/question/" + strconv.Itoa(questionID) +
			"/answer/" + strconv.Itoa(answerID)
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("delete_answer: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CloneAnswers(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		sourceID, err := req.RequireInt("source_question_id")
		if err != nil {
			return nil, fmt.Errorf("source_question_id is required")
		}
		targetID, err := req.RequireInt("target_question_id")
		if err != nil {
			return nil, fmt.Errorf("target_question_id is required")
		}
		path := "/platform/content/" + publicID +
			"/question/" + strconv.Itoa(sourceID) +
			"/answer/clone/" + strconv.Itoa(targetID)
		data, err := c.Get(path, nil)
		if err != nil {
			return nil, fmt.Errorf("clone_answers: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetAnswerOrder(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		questionID, err := req.RequireInt("question_id")
		if err != nil {
			return nil, fmt.Errorf("question_id is required")
		}
		path := "/platform/content/" + publicID + "/order/answers/" + strconv.Itoa(questionID)
		data, err := c.Get(path, url.Values{})
		if err != nil {
			return nil, fmt.Errorf("get_answer_order: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// UpdateAnswerOrder accepts a JSON array string: [{"id":1,"position":2},...]
func UpdateAnswerOrder(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		questionID, err := req.RequireInt("question_id")
		if err != nil {
			return nil, fmt.Errorf("question_id is required")
		}
		raw, err := req.RequireString("answers")
		if err != nil || raw == "" {
			return nil, fmt.Errorf("answers is required (JSON array of {id,position} objects)")
		}
		var positions []map[string]any
		if err := json.Unmarshal([]byte(raw), &positions); err != nil {
			return nil, fmt.Errorf("answers must be a valid JSON array: %w", err)
		}
		path := "/platform/content/" + publicID + "/order/answers/" + strconv.Itoa(questionID)
		data, err := c.Put(path, map[string]any{"answers": positions})
		if err != nil {
			return nil, fmt.Errorf("update_answer_order: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
