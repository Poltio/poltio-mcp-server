package tools

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

func ListPixelCodes(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		data, err := c.Get("/platform/pixel-codes", q)
		if err != nil {
			return nil, fmt.Errorf("list_pixel_codes: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreatePixelCode(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil || name == "" {
			return nil, fmt.Errorf("name is required")
		}
		code, err := req.RequireString("code")
		if err != nil || code == "" {
			return nil, fmt.Errorf("code is required (HTML snippet)")
		}
		data, err := c.Post("/platform/pixel-codes", map[string]any{"name": name, "code": code})
		if err != nil {
			return nil, fmt.Errorf("create_pixel_code: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdatePixelCode(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pixelCodeID, err := req.RequireInt("pixel_code_id")
		if err != nil {
			return nil, fmt.Errorf("pixel_code_id is required")
		}
		body := map[string]any{}
		if v := req.GetString("name", ""); v != "" {
			body["name"] = v
		}
		if v := req.GetString("code", ""); v != "" {
			body["code"] = v
		}
		data, err := c.Put("/platform/pixel-codes/"+strconv.Itoa(pixelCodeID), body)
		if err != nil {
			return nil, fmt.Errorf("update_pixel_code: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeletePixelCode(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pixelCodeID, err := req.RequireInt("pixel_code_id")
		if err != nil {
			return nil, fmt.Errorf("pixel_code_id is required")
		}
		data, err := c.Delete("/platform/pixel-codes/" + strconv.Itoa(pixelCodeID))
		if err != nil {
			return nil, fmt.Errorf("delete_pixel_code: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func SetContentPixelCode(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		pcID, err := req.RequireInt("pixel_code_id")
		if err != nil {
			return nil, fmt.Errorf("pixel_code_id is required")
		}
		data, err := c.Put("/platform/content/"+publicID+"/pixel-code", map[string]any{"pixel_code_id": pcID})
		if err != nil {
			return nil, fmt.Errorf("set_content_pixel_code: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func RemoveContentPixelCode(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		data, err := c.Delete("/platform/content/" + publicID + "/pixel-code")
		if err != nil {
			return nil, fmt.Errorf("remove_content_pixel_code: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func SetQuestionPixelCode(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		questionID, err := req.RequireInt("question_id")
		if err != nil {
			return nil, fmt.Errorf("question_id is required")
		}
		pcID, err := req.RequireInt("pixel_code_id")
		if err != nil {
			return nil, fmt.Errorf("pixel_code_id is required")
		}
		path := "/platform/content/" + publicID + "/question/" + strconv.Itoa(questionID) + "/pixel-code"
		data, err := c.Put(path, map[string]any{"pixel_code_id": pcID})
		if err != nil {
			return nil, fmt.Errorf("set_question_pixel_code: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func RemoveQuestionPixelCode(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		questionID, err := req.RequireInt("question_id")
		if err != nil {
			return nil, fmt.Errorf("question_id is required")
		}
		path := "/platform/content/" + publicID + "/question/" + strconv.Itoa(questionID) + "/pixel-code"
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("remove_question_pixel_code: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func SetAnswerPixelCode(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		pcID, err := req.RequireInt("pixel_code_id")
		if err != nil {
			return nil, fmt.Errorf("pixel_code_id is required")
		}
		path := "/platform/content/" + publicID +
			"/question/" + strconv.Itoa(questionID) +
			"/answer/" + strconv.Itoa(answerID) + "/pixel-code"
		data, err := c.Put(path, map[string]any{"pixel_code_id": pcID})
		if err != nil {
			return nil, fmt.Errorf("set_answer_pixel_code: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func RemoveAnswerPixelCode(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			"/answer/" + strconv.Itoa(answerID) + "/pixel-code"
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("remove_answer_pixel_code: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func SetResultPixelCode(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		resultID, err := req.RequireInt("result_id")
		if err != nil {
			return nil, fmt.Errorf("result_id is required")
		}
		pcID, err := req.RequireInt("pixel_code_id")
		if err != nil {
			return nil, fmt.Errorf("pixel_code_id is required")
		}
		path := "/platform/content/" + publicID + "/result/" + strconv.Itoa(resultID) + "/pixel-code"
		data, err := c.Put(path, map[string]any{"pixel_code_id": pcID})
		if err != nil {
			return nil, fmt.Errorf("set_result_pixel_code: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func RemoveResultPixelCode(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		resultID, err := req.RequireInt("result_id")
		if err != nil {
			return nil, fmt.Errorf("result_id is required")
		}
		path := "/platform/content/" + publicID + "/result/" + strconv.Itoa(resultID) + "/pixel-code"
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("remove_result_pixel_code: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func SetResultClickPixelCode(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		resultID, err := req.RequireInt("result_id")
		if err != nil {
			return nil, fmt.Errorf("result_id is required")
		}
		pcID, err := req.RequireInt("pixel_code_id")
		if err != nil {
			return nil, fmt.Errorf("pixel_code_id is required")
		}
		path := "/platform/content/" + publicID + "/result/" + strconv.Itoa(resultID) + "/click-pixel-code"
		data, err := c.Put(path, map[string]any{"pixel_code_id": pcID})
		if err != nil {
			return nil, fmt.Errorf("set_result_click_pixel_code: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func RemoveResultClickPixelCode(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		resultID, err := req.RequireInt("result_id")
		if err != nil {
			return nil, fmt.Errorf("result_id is required")
		}
		path := "/platform/content/" + publicID + "/result/" + strconv.Itoa(resultID) + "/click-pixel-code"
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("remove_result_click_pixel_code: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
