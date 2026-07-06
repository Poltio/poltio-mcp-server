package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// ContentClient is the interface the tool handlers use to call the Poltio API.
// client.PoltioClient satisfies this interface.
type ContentClient interface {
	Get(path string, query url.Values) ([]byte, error)
	Post(path string, body any) ([]byte, error)
	Put(path string, body any) ([]byte, error)
	Delete(path string) ([]byte, error)
}

func ListContent(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		if t := req.GetString("type", ""); t != "" {
			q.Set("type", t)
		}
		if search := req.GetString("q", ""); search != "" {
			q.Set("q", search)
		}
		if order := req.GetString("order", ""); order != "" {
			q.Set("order", order)
		}
		if sort := req.GetString("sort", ""); sort != "" {
			q.Set("sort", sort)
		}
		data, err := c.Get("/platform/content", q)
		if err != nil {
			return nil, fmt.Errorf("list_content: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetContent(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		data, err := c.Get("/platform/content/"+publicID, nil)
		if err != nil {
			return nil, fmt.Errorf("get_content: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateContent(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		contentType, err := req.RequireString("type")
		if err != nil || contentType == "" {
			return nil, fmt.Errorf("type is required")
		}
		title, err := req.RequireString("title")
		if err != nil || title == "" {
			return nil, fmt.Errorf("title is required")
		}
		body := map[string]any{
			"type":  contentType,
			"title": title,
		}
		if v := req.GetString("desc", ""); v != "" {
			body["desc"] = v
		}
		if v := req.GetString("name", ""); v != "" {
			body["name"] = v
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
		if v := req.GetInt("skip_start", -1); v >= 0 {
			body["skip_start"] = v
		}
		if v := req.GetInt("skip_result", -1); v >= 0 {
			body["skip_result"] = v
		}
		if v := req.GetInt("hide_results", -1); v >= 0 {
			body["hide_results"] = v
		}
		if v := req.GetInt("hide_counter", -1); v >= 0 {
			body["hide_counter"] = v
		}
		if v := req.GetInt("display_repeat", -1); v >= 0 {
			body["display_repeat"] = v
		}
		if v := req.GetString("vertical_mobile_image", ""); v != "" {
			body["vertical_mobile_image"] = v
		}
		if v := req.GetString("embed_footer_url", ""); v != "" {
			body["embed_footer_url"] = v
		}
		if v := req.GetString("embed_background", ""); v != "" {
			body["embed_background"] = v
		}
		if v := req.GetString("theme_id", ""); v != "" {
			body["theme_id"] = v
		}
		if v := req.GetInt("is_searchable", -1); v >= 0 {
			body["is_searchable"] = v
		}
		if v := req.GetInt("is_calculator", -1); v >= 0 {
			body["is_calculator"] = v
		}
		if v := req.GetInt("search_results_per_page", -1); v >= 0 {
			body["search_results_per_page"] = v
		}
		if v := req.GetInt("boost_results_min_view", -1); v >= 0 {
			body["boost_results_min_view"] = v
		}
		if v := req.GetInt("boost_results_ratio", -1); v >= 0 {
			body["boost_results_ratio"] = v
		}
		if v := req.GetInt("result_loading", -1); v >= 0 {
			body["result_loading"] = v
		}
		if v := req.GetString("loading_next_question_label", ""); v != "" {
			body["loading_next_question_label"] = v
		}
		if v := req.GetString("loading_result_label", ""); v != "" {
			body["loading_result_label"] = v
		}
		if v := req.GetInt("play_once", -1); v >= 0 {
			body["play_once"] = v
		}
		if v := req.GetString("play_once_strategy", ""); v != "" {
			body["play_once_strategy"] = v
		}
		if v := req.GetString("play_once_msg", ""); v != "" {
			body["play_once_msg"] = v
		}
		if v := req.GetString("play_once_img", ""); v != "" {
			body["play_once_img"] = v
		}
		if v := req.GetString("play_once_link", ""); v != "" {
			body["play_once_link"] = v
		}
		if v := req.GetString("play_once_btn", ""); v != "" {
			body["play_once_btn"] = v
		}
		if v := req.GetInt("end_date_day", -1); v >= 0 {
			body["end_date_day"] = v
		}
		if v := req.GetInt("end_date_hour", -1); v >= 0 {
			body["end_date_hour"] = v
		}
		if v := req.GetInt("end_date_minute", -1); v >= 0 {
			body["end_date_minute"] = v
		}
		if v := req.GetString("attributes_json", ""); v != "" {
			var attrs map[string]any
			if err := json.Unmarshal([]byte(v), &attrs); err == nil {
				body["attributes"] = attrs
			}
		}
		if v := req.GetString("options_json", ""); v != "" {
			var opts map[string]any
			if err := json.Unmarshal([]byte(v), &opts); err == nil {
				body["options"] = opts
			}
		}
		data, err := c.Post("/platform/content", body)
		if err != nil {
			return nil, fmt.Errorf("create_content: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func PublishContent(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		data, err := c.Get("/platform/content/"+publicID+"/publish", nil)
		if err != nil {
			return nil, fmt.Errorf("publish_content: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func ListDrafts(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		if t := req.GetString("type", ""); t != "" {
			q.Set("type", t)
		}
		if search := req.GetString("q", ""); search != "" {
			q.Set("q", search)
		}
		if order := req.GetString("order", ""); order != "" {
			q.Set("order", order)
		}
		if sort := req.GetString("sort", ""); sort != "" {
			q.Set("sort", sort)
		}
		data, err := c.Get("/platform/content/drafts", q)
		if err != nil {
			return nil, fmt.Errorf("list_drafts: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateContent(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		body := map[string]any{}
		if v := req.GetString("title", ""); v != "" {
			body["title"] = v
		}
		if v := req.GetString("desc", ""); v != "" {
			body["desc"] = v
		}
		if v := req.GetString("name", ""); v != "" {
			body["name"] = v
		}
		if v := req.GetString("type", ""); v != "" {
			body["type"] = v
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
		if v := req.GetInt("skip_start", -1); v >= 0 {
			body["skip_start"] = v
		}
		if v := req.GetInt("skip_result", -1); v >= 0 {
			body["skip_result"] = v
		}
		if v := req.GetInt("hide_results", -1); v >= 0 {
			body["hide_results"] = v
		}
		if v := req.GetInt("hide_counter", -1); v >= 0 {
			body["hide_counter"] = v
		}
		if v := req.GetInt("display_repeat", -1); v >= 0 {
			body["display_repeat"] = v
		}
		if v := req.GetString("vertical_mobile_image", ""); v != "" {
			body["vertical_mobile_image"] = v
		}
		if v := req.GetString("embed_footer_url", ""); v != "" {
			body["embed_footer_url"] = v
		}
		if v := req.GetString("embed_background", ""); v != "" {
			body["embed_background"] = v
		}
		if v := req.GetString("theme_id", ""); v != "" {
			body["theme_id"] = v
		}
		if v := req.GetInt("is_searchable", -1); v >= 0 {
			body["is_searchable"] = v
		}
		if v := req.GetInt("is_calculator", -1); v >= 0 {
			body["is_calculator"] = v
		}
		if v := req.GetInt("search_results_per_page", -1); v >= 0 {
			body["search_results_per_page"] = v
		}
		if v := req.GetInt("boost_results_min_view", -1); v >= 0 {
			body["boost_results_min_view"] = v
		}
		if v := req.GetInt("boost_results_ratio", -1); v >= 0 {
			body["boost_results_ratio"] = v
		}
		if v := req.GetInt("result_loading", -1); v >= 0 {
			body["result_loading"] = v
		}
		if v := req.GetString("loading_next_question_label", ""); v != "" {
			body["loading_next_question_label"] = v
		}
		if v := req.GetString("loading_result_label", ""); v != "" {
			body["loading_result_label"] = v
		}
		if v := req.GetInt("play_once", -1); v >= 0 {
			body["play_once"] = v
		}
		if v := req.GetString("play_once_strategy", ""); v != "" {
			body["play_once_strategy"] = v
		}
		if v := req.GetString("play_once_msg", ""); v != "" {
			body["play_once_msg"] = v
		}
		if v := req.GetString("play_once_img", ""); v != "" {
			body["play_once_img"] = v
		}
		if v := req.GetString("play_once_link", ""); v != "" {
			body["play_once_link"] = v
		}
		if v := req.GetString("play_once_btn", ""); v != "" {
			body["play_once_btn"] = v
		}
		if v := req.GetInt("end_date_day", -1); v >= 0 {
			body["end_date_day"] = v
		}
		if v := req.GetInt("end_date_hour", -1); v >= 0 {
			body["end_date_hour"] = v
		}
		if v := req.GetInt("end_date_minute", -1); v >= 0 {
			body["end_date_minute"] = v
		}
		if v := req.GetString("attributes_json", ""); v != "" {
			var attrs map[string]any
			if err := json.Unmarshal([]byte(v), &attrs); err == nil {
				body["attributes"] = attrs
			}
		}
		if v := req.GetString("options_json", ""); v != "" {
			var opts map[string]any
			if err := json.Unmarshal([]byte(v), &opts); err == nil {
				body["options"] = opts
			}
		}
		data, err := c.Put("/platform/content/"+publicID, body)
		if err != nil {
			return nil, fmt.Errorf("update_content: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteContent(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		data, err := c.Delete("/platform/content/" + publicID)
		if err != nil {
			return nil, fmt.Errorf("delete_content: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DuplicateContent(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		data, err := c.Get("/platform/content/"+publicID+"/duplicate", nil)
		if err != nil {
			return nil, fmt.Errorf("duplicate_content: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetContentResults(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		if v := req.GetString("order_by", ""); v != "" {
			q.Set("order_by", v)
		}
		if v := req.GetString("order_dir", ""); v != "" {
			q.Set("order_dir", v)
		}
		data, err := c.Get("/platform/content/"+publicID+"/results", q)
		if err != nil {
			return nil, fmt.Errorf("get_content_results: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetContentSessions(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		data, err := c.Get("/platform/content/"+publicID+"/sessions", q)
		if err != nil {
			return nil, fmt.Errorf("get_content_sessions: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetContentEdit(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		data, err := c.Get("/platform/content/"+publicID+"/edit", nil)
		if err != nil {
			return nil, fmt.Errorf("get_content_edit: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func ListTemplates(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := c.Get("/platform/content/templates", nil)
		if err != nil {
			return nil, fmt.Errorf("list_templates: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetTemplate(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		data, err := c.Get("/platform/content/templates/"+publicID, nil)
		if err != nil {
			return nil, fmt.Errorf("get_template: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UseTemplate(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		data, err := c.Get("/platform/content/templates/"+publicID+"/use", nil)
		if err != nil {
			return nil, fmt.Errorf("use_template: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetVoteSources(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		data, err := c.Get("/platform/content/"+publicID+"/sources", q)
		if err != nil {
			return nil, fmt.Errorf("get_vote_sources: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetSankey(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		data, err := c.Get("/platform/content/"+publicID+"/sankey", q)
		if err != nil {
			return nil, fmt.Errorf("get_sankey: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetSankeyUsers(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		fromID, err := req.RequireString("from_id")
		if err != nil || fromID == "" {
			return nil, fmt.Errorf("from_id is required")
		}
		toID, err := req.RequireString("to_id")
		if err != nil || toID == "" {
			return nil, fmt.Errorf("to_id is required")
		}
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		path := "/platform/content/" + publicID + "/sankey/users/" + fromID + "/" + toID
		data, err := c.Get(path, q)
		if err != nil {
			return nil, fmt.Errorf("get_sankey_users: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetSearchableFields(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		data, err := c.Get("/platform/content/"+publicID+"/searchable-fields", nil)
		if err != nil {
			return nil, fmt.Errorf("get_searchable_fields: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetSessionUrls(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		data, err := c.Get("/platform/content/"+publicID+"/session/urls", q)
		if err != nil {
			return nil, fmt.Errorf("get_session_urls: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetContentMetrics(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		period, err := req.RequireString("period")
		if err != nil || period == "" {
			return nil, fmt.Errorf("period is required (day, week, month, year)")
		}
		body := map[string]any{}
		if v := req.GetString("start", ""); v != "" {
			body["start"] = v
		}
		if v := req.GetString("end", ""); v != "" {
			body["end"] = v
		}
		if v := req.GetString("metrics", ""); v != "" {
			parts := strings.Split(v, ",")
			trimmed := make([]string, 0, len(parts))
			for _, p := range parts {
				if s := strings.TrimSpace(p); s != "" {
					trimmed = append(trimmed, s)
				}
			}
			if len(trimmed) > 0 {
				body["metrics"] = trimmed
			}
		}
		if v := req.GetString("device_type", ""); v != "" {
			body["device_type"] = v
		}
		data, err := c.Post("/platform/content/"+publicID+"/metrics/"+period, body)
		if err != nil {
			return nil, fmt.Errorf("get_content_metrics: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CalculatorTest(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		formula, err := req.RequireString("formula")
		if err != nil || formula == "" {
			return nil, fmt.Errorf("formula is required (e.g. Q1 + Q2 + Q3)")
		}
		varsRaw, err := req.RequireString("vars")
		if err != nil || varsRaw == "" {
			return nil, fmt.Errorf("vars is required (JSON array of {Qx: value} objects)")
		}
		var vars []map[string]any
		if err := json.Unmarshal([]byte(varsRaw), &vars); err != nil {
			return nil, fmt.Errorf("vars must be a valid JSON array: %w", err)
		}
		data, err := c.Post("/platform/content/calculator-test", map[string]any{
			"formula": formula,
			"vars":    vars,
		})
		if err != nil {
			return nil, fmt.Errorf("calculator_test: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
