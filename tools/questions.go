package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

func AddQuestion(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		answerType, err := req.RequireString("answer_type")
		if err != nil || answerType == "" {
			return nil, fmt.Errorf("answer_type is required (media, text, score, star_rating, yesno, free_text, free_number, autocomplete)")
		}
		body := map[string]any{"answer_type": answerType}
		if v := req.GetString("title", ""); v != "" {
			body["title"] = v
		}
		if v := req.GetString("background", ""); v != "" {
			body["background"] = v
		}
		if v := req.GetString("alt", ""); v != "" {
			body["alt"] = v
		}
		if v := req.GetString("vertical_image", ""); v != "" {
			body["vertical_image"] = v
		}
		if v := req.GetInt("allow_multiple_answers", -1); v >= 0 {
			body["allow_multiple_answers"] = v
		}
		if v := req.GetInt("is_skippable", -1); v >= 0 {
			body["is_skippable"] = v
		}
		if v := req.GetInt("rotate_answers", -1); v >= 0 {
			body["rotate_answers"] = v
		}
		if v := req.GetString("name", ""); v != "" {
			body["name"] = v
		}
		if v := req.GetInt("max_multi_punch_answer", -1); v >= 0 {
			body["max_multi_punch_answer"] = v
		}
		if v := req.GetInt("recommended_popular_answer", -1); v >= 0 {
			body["recommended_popular_answer"] = v
		}
		if v := req.GetString("luv", ""); v != "" {
			body["luv"] = v
		}
		if v := req.GetInt("is_searchable", -1); v >= 0 {
			body["is_searchable"] = v
		}
		if v := req.GetString("cal_val_default", ""); v != "" {
			body["cal_val_default"] = v
		}
		if v := req.GetString("autocomplete_help", ""); v != "" {
			body["autocomplete_help"] = v
		}
		if v := req.GetString("autocomplete_placeholder", ""); v != "" {
			body["autocomplete_placeholder"] = v
		}
		if v := req.GetInt("position", -1); v >= 0 {
			body["position"] = v
		}
		if v := req.GetString("conditions", ""); v != "" {
			body["conditions"] = v
		}
		if v := req.GetInt("condition_reverse", -1); v >= 0 {
			body["condition_reverse"] = v
		}
		data, err := c.Post("/platform/content/"+publicID+"/question", body)
		if err != nil {
			return nil, fmt.Errorf("add_question: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateQuestion(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		questionID, err := req.RequireInt("question_id")
		if err != nil {
			return nil, fmt.Errorf("question_id is required")
		}
		answerType, err := req.RequireString("answer_type")
		if err != nil || answerType == "" {
			return nil, fmt.Errorf("answer_type is required (media, text, score, star_rating, yesno, free_text, free_number, autocomplete)")
		}
		body := map[string]any{"answer_type": answerType}
		if v := req.GetString("title", ""); v != "" {
			body["title"] = v
		}
		if v := req.GetString("background", ""); v != "" {
			body["background"] = v
		}
		if v := req.GetString("alt", ""); v != "" {
			body["alt"] = v
		}
		if v := req.GetString("vertical_image", ""); v != "" {
			body["vertical_image"] = v
		}
		if v := req.GetInt("allow_multiple_answers", -1); v >= 0 {
			body["allow_multiple_answers"] = v
		}
		if v := req.GetInt("is_skippable", -1); v >= 0 {
			body["is_skippable"] = v
		}
		if v := req.GetInt("rotate_answers", -1); v >= 0 {
			body["rotate_answers"] = v
		}
		if v := req.GetString("name", ""); v != "" {
			body["name"] = v
		}
		if v := req.GetInt("max_multi_punch_answer", -1); v >= 0 {
			body["max_multi_punch_answer"] = v
		}
		if v := req.GetInt("recommended_popular_answer", -1); v >= 0 {
			body["recommended_popular_answer"] = v
		}
		if v := req.GetString("luv", ""); v != "" {
			body["luv"] = v
		}
		if v := req.GetInt("is_searchable", -1); v >= 0 {
			body["is_searchable"] = v
		}
		if v := req.GetString("cal_val_default", ""); v != "" {
			body["cal_val_default"] = v
		}
		if v := req.GetString("autocomplete_help", ""); v != "" {
			body["autocomplete_help"] = v
		}
		if v := req.GetString("autocomplete_placeholder", ""); v != "" {
			body["autocomplete_placeholder"] = v
		}
		if v := req.GetInt("position", -1); v >= 0 {
			body["position"] = v
		}
		if v := req.GetString("conditions", ""); v != "" {
			body["conditions"] = v
		}
		if v := req.GetInt("condition_reverse", -1); v >= 0 {
			body["condition_reverse"] = v
		}
		path := "/platform/content/" + publicID + "/question/" + strconv.Itoa(questionID)
		data, err := c.Put(path, body)
		if err != nil {
			return nil, fmt.Errorf("update_question: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteQuestion(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		questionID, err := req.RequireInt("question_id")
		if err != nil {
			return nil, fmt.Errorf("question_id is required")
		}
		path := "/platform/content/" + publicID + "/question/" + strconv.Itoa(questionID)
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("delete_question: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetContentConditions(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		data, err := c.Get("/platform/content/"+publicID+"/conditions", nil)
		if err != nil {
			return nil, fmt.Errorf("get_content_conditions: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func AddQuestionCondition(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		body := map[string]any{"answer_id": answerID}
		if v := req.GetInt("condition_reverse", -1); v >= 0 {
			body["condition_reverse"] = v == 1
		}
		path := "/platform/content/" + publicID + "/question/" + strconv.Itoa(questionID) + "/conditions/add"
		data, err := c.Post(path, body)
		if err != nil {
			return nil, fmt.Errorf("add_question_condition: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func RemoveQuestionCondition(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		path := "/platform/content/" + publicID + "/question/" + strconv.Itoa(questionID) + "/conditions/remove"
		data, err := c.Post(path, map[string]any{"answer_id": answerID})
		if err != nil {
			return nil, fmt.Errorf("remove_question_condition: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func ClearQuestionConditions(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		questionID, err := req.RequireInt("question_id")
		if err != nil {
			return nil, fmt.Errorf("question_id is required")
		}
		path := "/platform/content/" + publicID + "/question/" + strconv.Itoa(questionID) + "/conditions"
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("clear_question_conditions: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetQuestionOrder(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		data, err := c.Get("/platform/content/"+publicID+"/order/questions", nil)
		if err != nil {
			return nil, fmt.Errorf("get_question_order: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// UpdateQuestionOrder accepts a JSON array string: [{"id":1,"position":2},...]
func UpdateQuestionOrder(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		raw, err := req.RequireString("questions")
		if err != nil || raw == "" {
			return nil, fmt.Errorf("questions is required (JSON array of {id,position} objects)")
		}
		var positions []map[string]any
		if err := json.Unmarshal([]byte(raw), &positions); err != nil {
			return nil, fmt.Errorf("questions must be a valid JSON array: %w", err)
		}
		data, err := c.Put("/platform/content/"+publicID+"/order/questions", map[string]any{"questions": positions})
		if err != nil {
			return nil, fmt.Errorf("update_question_order: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// GetQuestionInputs returns free-text answer inputs for a question.
func GetQuestionInputs(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		questionID, err := req.RequireInt("question_id")
		if err != nil {
			return nil, fmt.Errorf("question_id is required")
		}
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		if v := req.GetString("order", ""); v != "" {
			q.Set("order", v)
		}
		if v := req.GetString("sort", ""); v != "" {
			q.Set("sort", v)
		}
		path := "/platform/content/" + publicID + "/question/" + strconv.Itoa(questionID) + "/inputs"
		data, err := c.Get(path, q)
		if err != nil {
			return nil, fmt.Errorf("get_question_inputs: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
