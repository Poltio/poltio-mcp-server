package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

func ListLeads(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		data, err := c.Get("/platform/leads", q)
		if err != nil {
			return nil, fmt.Errorf("list_leads: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateLead(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil || name == "" {
			return nil, fmt.Errorf("name is required")
		}
		leadType, err := req.RequireString("type")
		if err != nil || leadType == "" {
			return nil, fmt.Errorf("type is required (input, redirect, empty, internal_redirect)")
		}
		body := map[string]any{"name": name, "type": leadType}
		if v := req.GetString("msg", ""); v != "" {
			body["msg"] = v
		}
		if v := req.GetString("fields", ""); v != "" {
			body["fields"] = v
		}
		if v := req.GetString("button_value", ""); v != "" {
			body["button_value"] = v
		}
		if v := req.GetString("redirect_url", ""); v != "" {
			body["redirect_url"] = v
		}
		if v := req.GetString("title", ""); v != "" {
			body["title"] = v
		}
		if v := req.GetString("youtube_id", ""); v != "" {
			body["youtube_id"] = v
		}
		if v := req.GetString("terms_conditions", ""); v != "" {
			body["terms_conditions"] = v
		}
		if v := req.GetString("terms_conditions2", ""); v != "" {
			body["terms_conditions2"] = v
		}
		if v := req.GetString("ios_link", ""); v != "" {
			body["ios_link"] = v
		}
		if v := req.GetString("android_link", ""); v != "" {
			body["android_link"] = v
		}
		if v := req.GetInt("is_active", -1); v >= 0 {
			body["is_active"] = v
		}
		if v := req.GetInt("mandatory", -1); v >= 0 {
			body["mandatory"] = v == 1
		}
		if v := req.GetString("image", ""); v != "" {
			body["image"] = v
		}
		if v := req.GetInt("tc_optional", -1); v >= 0 {
			body["tc_optional"] = v
		}
		if v := req.GetInt("tc2_optional", -1); v >= 0 {
			body["tc2_optional"] = v
		}
		if v := req.GetInt("auto_open", -1); v >= 0 {
			body["auto_open"] = v
		}
		if v := req.GetInt("auto_open_delay", -1); v >= 0 {
			body["auto_open_delay"] = v
		}
		if v := req.GetInt("open_minimized", -1); v >= 0 {
			body["open_minimized"] = v
		}
		if v := req.GetInt("delay", -1); v >= 0 {
			body["delay"] = v
		}
		if v := req.GetInt("stop_set", -1); v >= 0 {
			body["stop_set"] = v
		}
		if v := req.GetInt("dont_shorten", -1); v >= 0 {
			body["dont_shorten"] = v
		}
		if v := req.GetString("link_target", ""); v != "" {
			body["link_target"] = v
		}
		if v := req.GetString("tc_short", ""); v != "" {
			body["tc_short"] = v
		}
		if v := req.GetString("tc2_short", ""); v != "" {
			body["tc2_short"] = v
		}
		if v := req.GetString("tc_approve_button_label", ""); v != "" {
			body["tc_approve_button_label"] = v
		}
		if v := req.GetString("tc_reject_button_label", ""); v != "" {
			body["tc_reject_button_label"] = v
		}
		if v := req.GetString("tc2_approve_button_label", ""); v != "" {
			body["tc2_approve_button_label"] = v
		}
		if v := req.GetString("tc2_reject_button_label", ""); v != "" {
			body["tc2_reject_button_label"] = v
		}
		if v := req.GetString("custom_labels_json", ""); v != "" {
			var labels map[string]any
			if err := json.Unmarshal([]byte(v), &labels); err == nil {
				body["custom_labels"] = labels
			}
		}
		data, err := c.Post("/platform/leads", body)
		if err != nil {
			return nil, fmt.Errorf("create_lead: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetLead(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		leadID, err := req.RequireString("lead_id")
		if err != nil || leadID == "" {
			return nil, fmt.Errorf("lead_id is required")
		}
		data, err := c.Get("/platform/leads/"+leadID, nil)
		if err != nil {
			return nil, fmt.Errorf("get_lead: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateLead(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		leadID, err := req.RequireString("lead_id")
		if err != nil || leadID == "" {
			return nil, fmt.Errorf("lead_id is required")
		}
		body := map[string]any{}
		if v := req.GetString("name", ""); v != "" {
			body["name"] = v
		}
		if v := req.GetString("type", ""); v != "" {
			body["type"] = v
		}
		if v := req.GetString("msg", ""); v != "" {
			body["msg"] = v
		}
		if v := req.GetString("fields", ""); v != "" {
			body["fields"] = v
		}
		if v := req.GetString("button_value", ""); v != "" {
			body["button_value"] = v
		}
		if v := req.GetString("redirect_url", ""); v != "" {
			body["redirect_url"] = v
		}
		if v := req.GetString("title", ""); v != "" {
			body["title"] = v
		}
		if v := req.GetString("youtube_id", ""); v != "" {
			body["youtube_id"] = v
		}
		if v := req.GetString("terms_conditions", ""); v != "" {
			body["terms_conditions"] = v
		}
		if v := req.GetString("terms_conditions2", ""); v != "" {
			body["terms_conditions2"] = v
		}
		if v := req.GetString("ios_link", ""); v != "" {
			body["ios_link"] = v
		}
		if v := req.GetString("android_link", ""); v != "" {
			body["android_link"] = v
		}
		if v := req.GetInt("is_active", -1); v >= 0 {
			body["is_active"] = v
		}
		if v := req.GetInt("mandatory", -1); v >= 0 {
			body["mandatory"] = v == 1
		}
		if v := req.GetString("image", ""); v != "" {
			body["image"] = v
		}
		if v := req.GetInt("tc_optional", -1); v >= 0 {
			body["tc_optional"] = v
		}
		if v := req.GetInt("tc2_optional", -1); v >= 0 {
			body["tc2_optional"] = v
		}
		if v := req.GetInt("auto_open", -1); v >= 0 {
			body["auto_open"] = v
		}
		if v := req.GetInt("auto_open_delay", -1); v >= 0 {
			body["auto_open_delay"] = v
		}
		if v := req.GetInt("open_minimized", -1); v >= 0 {
			body["open_minimized"] = v
		}
		if v := req.GetInt("delay", -1); v >= 0 {
			body["delay"] = v
		}
		if v := req.GetInt("stop_set", -1); v >= 0 {
			body["stop_set"] = v
		}
		if v := req.GetInt("dont_shorten", -1); v >= 0 {
			body["dont_shorten"] = v
		}
		if v := req.GetString("link_target", ""); v != "" {
			body["link_target"] = v
		}
		if v := req.GetString("tc_short", ""); v != "" {
			body["tc_short"] = v
		}
		if v := req.GetString("tc2_short", ""); v != "" {
			body["tc2_short"] = v
		}
		if v := req.GetString("tc_approve_button_label", ""); v != "" {
			body["tc_approve_button_label"] = v
		}
		if v := req.GetString("tc_reject_button_label", ""); v != "" {
			body["tc_reject_button_label"] = v
		}
		if v := req.GetString("tc2_approve_button_label", ""); v != "" {
			body["tc2_approve_button_label"] = v
		}
		if v := req.GetString("tc2_reject_button_label", ""); v != "" {
			body["tc2_reject_button_label"] = v
		}
		if v := req.GetString("custom_labels_json", ""); v != "" {
			var labels map[string]any
			if err := json.Unmarshal([]byte(v), &labels); err == nil {
				body["custom_labels"] = labels
			}
		}
		data, err := c.Put("/platform/leads/"+leadID, body)
		if err != nil {
			return nil, fmt.Errorf("update_lead: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteLead(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		leadID, err := req.RequireString("lead_id")
		if err != nil || leadID == "" {
			return nil, fmt.Errorf("lead_id is required")
		}
		data, err := c.Delete("/platform/leads/" + leadID)
		if err != nil {
			return nil, fmt.Errorf("delete_lead: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetLeadInputs(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		leadID, err := req.RequireString("lead_id")
		if err != nil || leadID == "" {
			return nil, fmt.Errorf("lead_id is required")
		}
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		data, err := c.Get("/platform/leads/"+leadID+"/inputs", q)
		if err != nil {
			return nil, fmt.Errorf("get_lead_inputs: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetLeadLogs(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		leadID, err := req.RequireString("lead_id")
		if err != nil || leadID == "" {
			return nil, fmt.Errorf("lead_id is required")
		}
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		data, err := c.Get("/platform/leads/"+leadID+"/logs", q)
		if err != nil {
			return nil, fmt.Errorf("get_lead_logs: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetLeadCodes(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		leadID, err := req.RequireString("lead_id")
		if err != nil || leadID == "" {
			return nil, fmt.Errorf("lead_id is required")
		}
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		data, err := c.Get("/platform/leads/"+leadID+"/codes", q)
		if err != nil {
			return nil, fmt.Errorf("get_lead_codes: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func AddLeadCodes(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		leadID, err := req.RequireString("lead_id")
		if err != nil || leadID == "" {
			return nil, fmt.Errorf("lead_id is required")
		}
		codes, err := req.RequireString("codes")
		if err != nil || codes == "" {
			return nil, fmt.Errorf("codes is required (one code per line)")
		}
		body := map[string]any{"codes": codes}
		if v := req.GetInt("single_use", -1); v >= 0 {
			body["single_use"] = v == 1
		}
		if v := req.GetInt("remove_existing", -1); v >= 0 {
			body["remove_existing"] = v == 1
		}
		data, err := c.Post("/platform/leads/"+leadID+"/codes", body)
		if err != nil {
			return nil, fmt.Errorf("add_lead_codes: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteAllLeadCodes(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		leadID, err := req.RequireString("lead_id")
		if err != nil || leadID == "" {
			return nil, fmt.Errorf("lead_id is required")
		}
		data, err := c.Delete("/platform/leads/" + leadID + "/codes")
		if err != nil {
			return nil, fmt.Errorf("delete_all_lead_codes: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateLeadCode(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		leadID, err := req.RequireString("lead_id")
		if err != nil || leadID == "" {
			return nil, fmt.Errorf("lead_id is required")
		}
		codeID, err := req.RequireString("lead_coupon_code_id")
		if err != nil || codeID == "" {
			return nil, fmt.Errorf("lead_coupon_code_id is required")
		}
		code, err := req.RequireString("code")
		if err != nil || code == "" {
			return nil, fmt.Errorf("code is required")
		}
		body := map[string]any{"code": code}
		if v := req.GetInt("single_use", -1); v >= 0 {
			body["single_use"] = v
		}
		path := "/platform/leads/" + leadID + "/codes/" + codeID
		data, err := c.Put(path, body)
		if err != nil {
			return nil, fmt.Errorf("update_lead_code: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteLeadCode(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		leadID, err := req.RequireString("lead_id")
		if err != nil || leadID == "" {
			return nil, fmt.Errorf("lead_id is required")
		}
		codeID, err := req.RequireString("lead_coupon_code_id")
		if err != nil || codeID == "" {
			return nil, fmt.Errorf("lead_coupon_code_id is required")
		}
		path := "/platform/leads/" + leadID + "/codes/" + codeID
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("delete_lead_code: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
