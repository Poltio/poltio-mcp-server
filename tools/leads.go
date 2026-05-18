package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

func SetContentLead(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		leadID, err := req.RequireInt("lead_id")
		if err != nil {
			return nil, fmt.Errorf("lead_id is required")
		}
		data, err := c.Put("/platform/content/"+publicID+"/lead", map[string]any{"lead_id": leadID})
		if err != nil {
			return nil, fmt.Errorf("set_content_lead: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func RemoveContentLead(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		data, err := c.Delete("/platform/content/" + publicID + "/lead")
		if err != nil {
			return nil, fmt.Errorf("remove_content_lead: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func SetQuestionLead(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		questionID, err := req.RequireInt("question_id")
		if err != nil {
			return nil, fmt.Errorf("question_id is required")
		}
		leadID, err := req.RequireInt("lead_id")
		if err != nil {
			return nil, fmt.Errorf("lead_id is required")
		}
		path := "/platform/content/" + publicID + "/question/" + strconv.Itoa(questionID) + "/lead"
		data, err := c.Put(path, map[string]any{"lead_id": leadID})
		if err != nil {
			return nil, fmt.Errorf("set_question_lead: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func RemoveQuestionLead(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		questionID, err := req.RequireInt("question_id")
		if err != nil {
			return nil, fmt.Errorf("question_id is required")
		}
		path := "/platform/content/" + publicID + "/question/" + strconv.Itoa(questionID) + "/lead"
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("remove_question_lead: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func SetAnswerLead(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		leadID, err := req.RequireInt("lead_id")
		if err != nil {
			return nil, fmt.Errorf("lead_id is required")
		}
		path := "/platform/content/" + publicID +
			"/question/" + strconv.Itoa(questionID) +
			"/answer/" + strconv.Itoa(answerID) + "/lead"
		data, err := c.Put(path, map[string]any{"lead_id": leadID})
		if err != nil {
			return nil, fmt.Errorf("set_answer_lead: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func RemoveAnswerLead(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			"/answer/" + strconv.Itoa(answerID) + "/lead"
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("remove_answer_lead: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func SetResultLead(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		resultID, err := req.RequireInt("result_id")
		if err != nil {
			return nil, fmt.Errorf("result_id is required")
		}
		leadID, err := req.RequireInt("lead_id")
		if err != nil {
			return nil, fmt.Errorf("lead_id is required")
		}
		path := "/platform/content/" + publicID + "/result/" + strconv.Itoa(resultID) + "/lead"
		data, err := c.Put(path, map[string]any{"lead_id": leadID})
		if err != nil {
			return nil, fmt.Errorf("set_result_lead: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func RemoveResultLead(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		resultID, err := req.RequireInt("result_id")
		if err != nil {
			return nil, fmt.Errorf("result_id is required")
		}
		path := "/platform/content/" + publicID + "/result/" + strconv.Itoa(resultID) + "/lead"
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("remove_result_lead: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
