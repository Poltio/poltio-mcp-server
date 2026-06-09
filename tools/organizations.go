package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Poltio/poltio-mcp-server/client"
)

type OrgClient interface {
	GetOrganizations() ([]byte, error)
	SetOrgID(id string)
}

// OrgOverrideSetter persists a per-session org override (bridge HTTP mode).
// In stdio mode this is nil; SwitchOrganization falls back to SetOrgID.
type OrgOverrideSetter interface {
	SetOrgOverride(ctx context.Context, grantID, orgID string) error
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

func SwitchOrganization(c OrgClient, overrideSetter OrgOverrideSetter) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireInt("id")
		if err != nil {
			return nil, fmt.Errorf("id is required")
		}
		orgID := strconv.Itoa(id)

		if overrideSetter != nil {
			// Bridge HTTP mode: persist org override to DB so all subsequent requests use it.
			grantID := client.GrantIDFromContext(ctx)
			if grantID != "" {
				if err := overrideSetter.SetOrgOverride(ctx, grantID, orgID); err != nil {
					return nil, fmt.Errorf("switch_organization: %w", err)
				}
				return mcp.NewToolResultText(fmt.Sprintf("Switched to organization %d.", id)), nil
			}
		}
		// stdio mode fallback
		c.SetOrgID(orgID)
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
