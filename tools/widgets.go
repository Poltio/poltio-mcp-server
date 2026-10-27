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

func ListWidgets(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		if v := req.GetString("public_id", ""); v != "" {
			q.Set("public_id", v)
		}
		data, err := c.Get("/platform/widgets", q)
		if err != nil {
			return nil, fmt.Errorf("list_widgets: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateWidget(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		name, err := req.RequireString("name")
		if err != nil || name == "" {
			return nil, fmt.Errorf("name is required")
		}
		// The API requires is_default on every create.
		body := map[string]any{"public_id": publicID, "name": name, "is_default": req.GetInt("is_default", 0) == 1}
		if v := req.GetInt("is_active", -1); v >= 0 {
			body["is_active"] = v == 1
		}
		if v := req.GetString("urls", ""); v != "" {
			parts := strings.Split(v, ",")
			filtered := make([]map[string]any, 0, len(parts))
			for _, p := range parts {
				if s := strings.TrimSpace(p); s != "" {
					filtered = append(filtered, map[string]any{"url": s})
				}
			}
			if len(filtered) > 0 {
				body["urls"] = filtered
			}
		}
		if v := req.GetString("starts_at", ""); v != "" {
			body["starts_at"] = v
		}
		if v := req.GetString("ends_at", ""); v != "" {
			body["ends_at"] = v
		}
		if v := req.GetString("overlay_options_json", ""); v != "" {
			var opts map[string]any
			if err := json.Unmarshal([]byte(v), &opts); err != nil {
				return nil, fmt.Errorf("overlay_options_json must be valid JSON: %w", err)
			}
			body["overlay_options"] = opts
		}
		if err := setAB(req, body); err != nil {
			return nil, err
		}
		data, err := c.Post("/platform/widgets", body)
		if err != nil {
			return nil, fmt.Errorf("create_widget: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetWidget(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		widgetID, err := req.RequireInt("widget_id")
		if err != nil {
			return nil, fmt.Errorf("widget_id is required")
		}
		data, err := c.Get("/platform/widgets/"+strconv.Itoa(widgetID), nil)
		if err != nil {
			return nil, fmt.Errorf("get_widget: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetWidgetStats(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		widgetID, err := req.RequireInt("widget_id")
		if err != nil {
			return nil, fmt.Errorf("widget_id is required")
		}
		q := url.Values{}
		for _, k := range []string{"start", "end", "device_type"} {
			if v := req.GetString(k, ""); v != "" {
				q.Set(k, v)
			}
		}
		data, err := c.Get("/platform/widgets/"+strconv.Itoa(widgetID)+"/stats", q)
		if err != nil {
			return nil, fmt.Errorf("get_widget_stats: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateWidget(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		widgetID, err := req.RequireInt("widget_id")
		if err != nil {
			return nil, fmt.Errorf("widget_id is required")
		}
		body := map[string]any{}
		if v := req.GetString("public_id", ""); v != "" {
			body["public_id"] = v
		}
		if v := req.GetString("name", ""); v != "" {
			body["name"] = v
		}
		if v := req.GetInt("is_default", -1); v >= 0 {
			body["is_default"] = v == 1
		}
		if v := req.GetInt("is_active", -1); v >= 0 {
			body["is_active"] = v == 1
		}
		if v := req.GetString("urls", ""); v != "" {
			parts := strings.Split(v, ",")
			filtered := make([]map[string]any, 0, len(parts))
			for _, p := range parts {
				if s := strings.TrimSpace(p); s != "" {
					filtered = append(filtered, map[string]any{"url": s})
				}
			}
			if len(filtered) > 0 {
				body["urls"] = filtered
			}
		}
		if v := req.GetString("starts_at", ""); v != "" {
			body["starts_at"] = v
		}
		if v := req.GetString("ends_at", ""); v != "" {
			body["ends_at"] = v
		}
		if v := req.GetString("overlay_options_json", ""); v != "" {
			var opts map[string]any
			if err := json.Unmarshal([]byte(v), &opts); err != nil {
				return nil, fmt.Errorf("overlay_options_json must be valid JSON: %w", err)
			}
			body["overlay_options"] = opts
		}
		if err := setAB(req, body); err != nil {
			return nil, err
		}
		if err := fillRequiredWidgetFields(c, widgetID, body); err != nil {
			return nil, fmt.Errorf("update_widget: %w", err)
		}
		data, err := c.Put("/platform/widgets/"+strconv.Itoa(widgetID), body)
		if err != nil {
			return nil, fmt.Errorf("update_widget: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// setAB copies ab_json into body; the literal "null" clears the A/B test.
func setAB(req mcp.CallToolRequest, body map[string]any) error {
	v := req.GetString("ab_json", "")
	if v == "" {
		return nil
	}
	var ab map[string]any
	if err := json.Unmarshal([]byte(v), &ab); err != nil {
		return fmt.Errorf("ab_json must be a JSON object or null: %w", err)
	}
	body["ab"] = ab
	return nil
}

// fillRequiredWidgetFields fills the fields the API requires on every PUT —
// public_id and name, plus is_default and urls on a parent widget — from the
// stored widget, so update_widget can change just one field.
func fillRequiredWidgetFields(c ContentClient, widgetID int, body map[string]any) error {
	data, err := c.Get("/platform/widgets/"+strconv.Itoa(widgetID), nil)
	if err != nil {
		return fmt.Errorf("read current widget: %w", err)
	}
	var w struct {
		ParentID  *int   `json:"parent_id"`
		PublicID  string `json:"public_id"`
		Name      string `json:"name"`
		IsDefault bool   `json:"is_default"`
		URLs      []struct {
			URL string `json:"url"`
		} `json:"urls"`
	}
	if err := json.Unmarshal(data, &w); err != nil {
		return fmt.Errorf("read current widget: %w", err)
	}
	if _, ok := body["public_id"]; !ok {
		body["public_id"] = w.PublicID
	}
	if _, ok := body["name"]; !ok {
		body["name"] = w.Name
	}
	if w.ParentID != nil { // a variation: is_default and urls are prohibited
		return nil
	}
	if _, ok := body["is_default"]; !ok {
		body["is_default"] = w.IsDefault
	}
	if _, ok := body["urls"]; !ok && len(w.URLs) > 0 {
		urls := make([]map[string]any, len(w.URLs))
		for i, u := range w.URLs {
			urls[i] = map[string]any{"url": u.URL}
		}
		body["urls"] = urls
	}
	return nil
}

func DeleteWidget(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		widgetID, err := req.RequireInt("widget_id")
		if err != nil {
			return nil, fmt.Errorf("widget_id is required")
		}
		data, err := c.Delete("/platform/widgets/" + strconv.Itoa(widgetID))
		if err != nil {
			return nil, fmt.Errorf("delete_widget: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// CreateWidgetVariation adds an alternative content to an existing widget. The
// variation is a widget in its own right, parented to the one given here.
func CreateWidgetVariation(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		widgetID, err := req.RequireInt("widget_id")
		if err != nil {
			return nil, fmt.Errorf("widget_id is required")
		}
		name, err := req.RequireString("name")
		if err != nil || name == "" {
			return nil, fmt.Errorf("name is required")
		}
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required (the content this variation shows)")
		}
		body := map[string]any{"name": name, "public_id": publicID}
		if v := req.GetInt("is_active", -1); v >= 0 {
			body["is_active"] = v == 1
		}
		if v := req.GetString("overlay_options_json", ""); v != "" {
			var opts map[string]any
			if err := json.Unmarshal([]byte(v), &opts); err != nil {
				return nil, fmt.Errorf("overlay_options_json must be valid JSON: %w", err)
			}
			body["overlay_options"] = opts
		}
		path := "/platform/widgets/" + strconv.Itoa(widgetID) + "/variations"
		data, err := c.Post(path, body)
		if err != nil {
			return nil, fmt.Errorf("create_widget_variation: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteWidgetVariation(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		widgetID, err := req.RequireInt("widget_id")
		if err != nil {
			return nil, fmt.Errorf("widget_id is required")
		}
		variationID, err := req.RequireInt("variation_id")
		if err != nil {
			return nil, fmt.Errorf("variation_id is required")
		}
		path := "/platform/widgets/" + strconv.Itoa(widgetID) + "/variations/" + strconv.Itoa(variationID)
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("delete_widget_variation: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
