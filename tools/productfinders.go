package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

// finderBody collects the writable fields of a product finder (data source
// content). Every field is optional; create adds the required ones on top.
func finderBody(req mcp.CallToolRequest) (map[string]any, error) {
	body := map[string]any{}
	for _, key := range []string{
		"name", "result_button_text", "result_url", "result_title", "result_desc",
		"filter_desc", "price_text", "old_price_text", "alt",
		"secondary_result_url", "secondary_result_button_text",
		"default_filters", "include_fields", "score_filters",
	} {
		if v := req.GetString(key, ""); v != "" {
			body[key] = v
		}
	}
	for _, key := range []string{
		"price_format", "old_price_text_strike", "display_price_discount_percent",
	} {
		if v := req.GetInt(key, -1); v >= 0 {
			body[key] = v == 1
		}
	}
	// ponytail: ids can be set but not cleared; add explicit nulls if unbinding
	// a content, pixel code or lead is ever asked for.
	for _, key := range []string{
		"content_id", "pixel_code_id", "click_pixel_code_id",
		"secondary_click_pixel_code_id", "lead_id", "per_page",
	} {
		if v := req.GetInt(key, 0); v > 0 {
			body[key] = v
		}
	}
	if v := req.GetString("search_replace_options_json", ""); v != "" {
		var opts map[string]any
		if err := json.Unmarshal([]byte(v), &opts); err != nil {
			return nil, fmt.Errorf("search_replace_options_json must be valid JSON: %w", err)
		}
		body["search_replace_options"] = opts
	}
	return body, nil
}

// finderFieldBody collects the writable fields of a searchable field.
func finderFieldBody(req mcp.CallToolRequest) map[string]any {
	body := map[string]any{}
	for _, key := range []string{"type", "label", "field"} {
		if v := req.GetString(key, ""); v != "" {
			body[key] = v
		}
	}
	if v := req.GetInt("element", 0); v > 0 {
		body["element"] = v
	}
	for _, key := range []string{"normalize", "optional", "index", "use_as_source_id", "is_sortable"} {
		if v := req.GetInt(key, -1); v >= 0 {
			body[key] = v == 1
		}
	}
	return body
}

func ListProductFinders(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		data, err := c.Get("/platform/dsc", q)
		if err != nil {
			return nil, fmt.Errorf("list_product_finders: %w", err)
		}
		return mcp.NewToolResultText(string(data) + publicLinkNote), nil
	}
}

func GetProductFinder(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireInt("product_finder_id")
		if err != nil {
			return nil, fmt.Errorf("product_finder_id is required")
		}
		data, err := c.Get("/platform/dsc/"+strconv.Itoa(id), nil)
		if err != nil {
			return nil, fmt.Errorf("get_product_finder: %w", err)
		}
		return mcp.NewToolResultText(string(data) + publicLinkNote), nil
	}
}

func CreateProductFinder(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil || name == "" {
			return nil, fmt.Errorf("name is required")
		}
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		body, err := finderBody(req)
		if err != nil {
			return nil, err
		}
		body["name"] = name
		body["data_source_id"] = dataSourceID
		data, err := c.Post("/platform/dsc", body)
		if err != nil {
			return nil, fmt.Errorf("create_product_finder: %w", err)
		}
		return mcp.NewToolResultText(string(data) + publicLinkNote), nil
	}
}

func UpdateProductFinder(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireInt("product_finder_id")
		if err != nil {
			return nil, fmt.Errorf("product_finder_id is required")
		}
		body, err := finderBody(req)
		if err != nil {
			return nil, err
		}
		if v := req.GetInt("data_source_id", 0); v > 0 {
			body["data_source_id"] = v
		}
		if len(body) == 0 {
			return nil, fmt.Errorf("nothing to update: pass at least one field")
		}
		data, err := c.Put("/platform/dsc/"+strconv.Itoa(id), body)
		if err != nil {
			return nil, fmt.Errorf("update_product_finder: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteProductFinder(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireInt("product_finder_id")
		if err != nil {
			return nil, fmt.Errorf("product_finder_id is required")
		}
		data, err := c.Delete("/platform/dsc/" + strconv.Itoa(id))
		if err != nil {
			return nil, fmt.Errorf("delete_product_finder: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func AddProductFinderField(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireInt("product_finder_id")
		if err != nil {
			return nil, fmt.Errorf("product_finder_id is required")
		}
		if _, err := req.RequireString("type"); err != nil {
			return nil, fmt.Errorf("type is required (primary, secondary, filter_string, filter_string_multi, filter_numeric)")
		}
		body := finderFieldBody(req)
		if body["field"] == nil && body["element"] == nil {
			return nil, fmt.Errorf("either field or element is required")
		}
		data, err := c.Post("/platform/dsc/"+strconv.Itoa(id)+"/fields", body)
		if err != nil {
			return nil, fmt.Errorf("add_product_finder_field: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateProductFinderField(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireInt("product_finder_id")
		if err != nil {
			return nil, fmt.Errorf("product_finder_id is required")
		}
		fieldID, err := req.RequireInt("field_id")
		if err != nil {
			return nil, fmt.Errorf("field_id is required")
		}
		body := finderFieldBody(req)
		if len(body) == 0 {
			return nil, fmt.Errorf("nothing to update: pass at least one field")
		}
		path := "/platform/dsc/" + strconv.Itoa(id) + "/fields/" + strconv.Itoa(fieldID)
		data, err := c.Put(path, body)
		if err != nil {
			return nil, fmt.Errorf("update_product_finder_field: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteProductFinderField(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireInt("product_finder_id")
		if err != nil {
			return nil, fmt.Errorf("product_finder_id is required")
		}
		fieldID, err := req.RequireInt("field_id")
		if err != nil {
			return nil, fmt.Errorf("field_id is required")
		}
		path := "/platform/dsc/" + strconv.Itoa(id) + "/fields/" + strconv.Itoa(fieldID)
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("delete_product_finder_field: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
