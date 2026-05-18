package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

func UpdateSettings(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		body := map[string]any{}
		if v := req.GetString("username", ""); v != "" {
			body["username"] = v
		}
		if v := req.GetString("email", ""); v != "" {
			body["email"] = v
		}
		if v := req.GetString("photo", ""); v != "" {
			body["photo"] = v
		}
		if len(body) == 0 {
			return nil, fmt.Errorf("at least one field (username, email, photo) is required")
		}
		data, err := c.Put("/platform/settings", body)
		if err != nil {
			return nil, fmt.Errorf("update_settings: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdatePassword(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		password, err := req.RequireString("password")
		if err != nil || password == "" {
			return nil, fmt.Errorf("password (current) is required")
		}
		newPassword, err := req.RequireString("new_password")
		if err != nil || newPassword == "" {
			return nil, fmt.Errorf("new_password is required")
		}
		newPasswordConfirm, err := req.RequireString("new_password_confirmation")
		if err != nil || newPasswordConfirm == "" {
			return nil, fmt.Errorf("new_password_confirmation is required")
		}
		data, err := c.Put("/platform/settings/password", map[string]any{
			"password":                  password,
			"new_password":              newPassword,
			"new_password_confirmation": newPasswordConfirm,
		})
		if err != nil {
			return nil, fmt.Errorf("update_password: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func ListConversionSettings(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := c.Get("/platform/settings/conversion", nil)
		if err != nil {
			return nil, fmt.Errorf("list_conversion_settings: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateConversionSetting(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		convURL, err := req.RequireString("url")
		if err != nil || convURL == "" {
			return nil, fmt.Errorf("url is required (your checkout success page URL)")
		}
		body := map[string]any{"url": convURL}
		if v := req.GetInt("catch_all", -1); v >= 0 {
			body["catch_all"] = v == 1
		}
		data, err := c.Post("/platform/settings/conversion", body)
		if err != nil {
			return nil, fmt.Errorf("create_conversion_setting: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateConversionSetting(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		settingID, err := req.RequireInt("conversion_setting_id")
		if err != nil {
			return nil, fmt.Errorf("conversion_setting_id is required")
		}
		body := map[string]any{}
		if v := req.GetString("url", ""); v != "" {
			body["url"] = v
		}
		if v := req.GetInt("catch_all", -1); v >= 0 {
			body["catch_all"] = v == 1
		}
		data, err := c.Put("/platform/settings/conversion/"+strconv.Itoa(settingID), body)
		if err != nil {
			return nil, fmt.Errorf("update_conversion_setting: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteConversionSetting(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		settingID, err := req.RequireInt("conversion_setting_id")
		if err != nil {
			return nil, fmt.Errorf("conversion_setting_id is required")
		}
		data, err := c.Delete("/platform/settings/conversion/" + strconv.Itoa(settingID))
		if err != nil {
			return nil, fmt.Errorf("delete_conversion_setting: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func ResendVerification(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := c.Post("/platform/settings/resend-verification", nil)
		if err != nil {
			return nil, fmt.Errorf("resend_verification: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func AcceptTerms(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := c.Post("/platform/settings/accept-terms", nil)
		if err != nil {
			return nil, fmt.Errorf("accept_terms: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func SetupTwoFactor(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := c.Post("/platform/settings/two-factor/setup", nil)
		if err != nil {
			return nil, fmt.Errorf("setup_two_factor: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func VerifyTwoFactor(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		verification, err := req.RequireString("verification")
		if err != nil || verification == "" {
			return nil, fmt.Errorf("verification is required (TOTP code from authenticator app)")
		}
		data, err := c.Post("/platform/settings/two-factor/verify", map[string]any{"verification": verification})
		if err != nil {
			return nil, fmt.Errorf("verify_two_factor: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DisableTwoFactor(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		verification, err := req.RequireString("verification")
		if err != nil || verification == "" {
			return nil, fmt.Errorf("verification is required (TOTP code from authenticator app)")
		}
		data, err := c.Post("/platform/settings/two-factor/disable", map[string]any{"verification": verification})
		if err != nil {
			return nil, fmt.Errorf("disable_two_factor: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func ResetTwoFactorRecoveryCodes(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		verification, err := req.RequireString("verification")
		if err != nil || verification == "" {
			return nil, fmt.Errorf("verification is required (TOTP code from authenticator app)")
		}
		data, err := c.Post("/platform/settings/two-factor/reset-recovery-codes", map[string]any{"verification": verification})
		if err != nil {
			return nil, fmt.Errorf("reset_two_factor_recovery_codes: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
