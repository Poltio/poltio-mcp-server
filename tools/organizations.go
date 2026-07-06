package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

type OrgClient interface {
	GetOrganizations() ([]byte, error)
	SetOrgID(id string)
}

func ListOrganizations(c OrgClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := c.GetOrganizations()
		if err != nil {
			return nil, fmt.Errorf("list_organizations: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func SwitchOrganization(c OrgClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireInt("id")
		if err != nil {
			return nil, fmt.Errorf("id is required")
		}
		c.SetOrgID(strconv.Itoa(id))
		return mcp.NewToolResultText(fmt.Sprintf("Switched to organization %d.", id)), nil
	}
}

func GetOrganization(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orgID, err := req.RequireInt("organization_id")
		if err != nil {
			return nil, fmt.Errorf("organization_id is required")
		}
		data, err := c.Get("/platform/organizations/"+strconv.Itoa(orgID), nil)
		if err != nil {
			return nil, fmt.Errorf("get_organization: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateOrganization(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orgID, err := req.RequireInt("organization_id")
		if err != nil {
			return nil, fmt.Errorf("organization_id is required")
		}
		name, err := req.RequireString("name")
		if err != nil || name == "" {
			return nil, fmt.Errorf("name is required")
		}
		data, err := c.Put("/platform/organizations/"+strconv.Itoa(orgID), map[string]any{"name": name})
		if err != nil {
			return nil, fmt.Errorf("update_organization: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func InviteOrgMember(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orgID, err := req.RequireInt("organization_id")
		if err != nil {
			return nil, fmt.Errorf("organization_id is required")
		}
		email, err := req.RequireString("email")
		if err != nil || email == "" {
			return nil, fmt.Errorf("email is required")
		}
		role, err := req.RequireString("role")
		if err != nil || role == "" {
			return nil, fmt.Errorf("role is required (admin, user, viewer)")
		}
		data, err := c.Post("/platform/organizations/"+strconv.Itoa(orgID)+"/invite",
			map[string]any{"email": email, "role": role})
		if err != nil {
			return nil, fmt.Errorf("invite_org_member: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func JoinOrganization(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orgID, err := req.RequireInt("organization_id")
		if err != nil {
			return nil, fmt.Errorf("organization_id is required")
		}
		token, err := req.RequireString("token")
		if err != nil || token == "" {
			return nil, fmt.Errorf("token is required (invite token from the invitation email)")
		}
		path := "/platform/organizations/" + strconv.Itoa(orgID) + "/join/" + token
		data, err := c.Get(path, nil)
		if err != nil {
			return nil, fmt.Errorf("join_organization: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func LeaveOrganization(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orgID, err := req.RequireInt("organization_id")
		if err != nil {
			return nil, fmt.Errorf("organization_id is required")
		}
		data, err := c.Get("/platform/organizations/"+strconv.Itoa(orgID)+"/leave", nil)
		if err != nil {
			return nil, fmt.Errorf("leave_organization: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CancelOrgInvite(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orgID, err := req.RequireInt("organization_id")
		if err != nil {
			return nil, fmt.Errorf("organization_id is required")
		}
		email, err := req.RequireString("email")
		if err != nil || email == "" {
			return nil, fmt.Errorf("email is required")
		}
		path := "/platform/organizations/" + strconv.Itoa(orgID) + "/cancel-invite/" + email
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("cancel_org_invite: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func RemoveOrgMember(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orgID, err := req.RequireInt("organization_id")
		if err != nil {
			return nil, fmt.Errorf("organization_id is required")
		}
		userID, err := req.RequireInt("user_id")
		if err != nil {
			return nil, fmt.Errorf("user_id is required")
		}
		path := "/platform/organizations/" + strconv.Itoa(orgID) + "/remove-member/" + strconv.Itoa(userID)
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("remove_org_member: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateOrgMember(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orgID, err := req.RequireInt("organization_id")
		if err != nil {
			return nil, fmt.Errorf("organization_id is required")
		}
		userID, err := req.RequireInt("user_id")
		if err != nil {
			return nil, fmt.Errorf("user_id is required")
		}
		role, err := req.RequireString("role")
		if err != nil || role == "" {
			return nil, fmt.Errorf("role is required (admin, user, viewer)")
		}
		data, err := c.Post("/platform/organizations/"+strconv.Itoa(orgID)+"/update-member",
			map[string]any{"user_id": userID, "role": role})
		if err != nil {
			return nil, fmt.Errorf("update_org_member: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// ipRuleBody builds the request body for create/update IP rule from
// name, allowed_json and blocked_json params.
func ipRuleBody(req mcp.CallToolRequest) (map[string]any, error) {
	body := map[string]any{}
	if v := req.GetString("name", ""); v != "" {
		body["name"] = v
	}
	for _, key := range []string{"allowed", "blocked"} {
		if v := req.GetString(key+"_json", ""); v != "" {
			var ips []string
			if err := json.Unmarshal([]byte(v), &ips); err != nil {
				return nil, fmt.Errorf("%s_json must be a JSON array of strings: %w", key, err)
			}
			body[key] = ips
		}
	}
	return body, nil
}

func ListIPRules(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orgID, err := req.RequireInt("organization_id")
		if err != nil {
			return nil, fmt.Errorf("organization_id is required")
		}
		data, err := c.Get("/platform/organizations/"+strconv.Itoa(orgID)+"/ip-rules", nil)
		if err != nil {
			return nil, fmt.Errorf("list_ip_rules: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateIPRule(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orgID, err := req.RequireInt("organization_id")
		if err != nil {
			return nil, fmt.Errorf("organization_id is required")
		}
		body, err := ipRuleBody(req)
		if err != nil {
			return nil, err
		}
		data, err := c.Post("/platform/organizations/"+strconv.Itoa(orgID)+"/ip-rules", body)
		if err != nil {
			return nil, fmt.Errorf("create_ip_rule: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateIPRule(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orgID, err := req.RequireInt("organization_id")
		if err != nil {
			return nil, fmt.Errorf("organization_id is required")
		}
		ruleID, err := req.RequireInt("ip_rule_id")
		if err != nil {
			return nil, fmt.Errorf("ip_rule_id is required")
		}
		body, err := ipRuleBody(req)
		if err != nil {
			return nil, err
		}
		path := "/platform/organizations/" + strconv.Itoa(orgID) + "/ip-rules/" + strconv.Itoa(ruleID)
		data, err := c.Put(path, body)
		if err != nil {
			return nil, fmt.Errorf("update_ip_rule: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteIPRule(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orgID, err := req.RequireInt("organization_id")
		if err != nil {
			return nil, fmt.Errorf("organization_id is required")
		}
		ruleID, err := req.RequireInt("ip_rule_id")
		if err != nil {
			return nil, fmt.Errorf("ip_rule_id is required")
		}
		path := "/platform/organizations/" + strconv.Itoa(orgID) + "/ip-rules/" + strconv.Itoa(ruleID)
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("delete_ip_rule: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
