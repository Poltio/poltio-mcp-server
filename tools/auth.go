package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func Login(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		email, err := req.RequireString("email")
		if err != nil || email == "" {
			return nil, fmt.Errorf("email is required")
		}
		password, err := req.RequireString("password")
		if err != nil || password == "" {
			return nil, fmt.Errorf("password is required")
		}
		body := map[string]any{"email": email, "password": password}
		if v := req.GetInt("client_id", 0); v > 0 {
			body["client_id"] = v
		}
		if v := req.GetString("client_secret", ""); v != "" {
			body["client_secret"] = v
		}
		data, err := c.Post("/platform/auth/login", body)
		if err != nil {
			return nil, fmt.Errorf("login: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func LoginVerifyTwoFactor(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tempToken, err := req.RequireString("temp_token")
		if err != nil || tempToken == "" {
			return nil, fmt.Errorf("temp_token is required")
		}
		verification, err := req.RequireString("verification")
		if err != nil || verification == "" {
			return nil, fmt.Errorf("verification is required")
		}
		body := map[string]any{"temp_token": tempToken, "verification": verification}
		if v := req.GetInt("client_id", 0); v > 0 {
			body["client_id"] = v
		}
		if v := req.GetString("client_secret", ""); v != "" {
			body["client_secret"] = v
		}
		data, err := c.Post("/platform/auth/two-factor/verify", body)
		if err != nil {
			return nil, fmt.Errorf("login_verify_two_factor: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func LoginVerifyTwoFactorRecovery(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tempToken, err := req.RequireString("temp_token")
		if err != nil || tempToken == "" {
			return nil, fmt.Errorf("temp_token is required")
		}
		verification, err := req.RequireString("verification")
		if err != nil || verification == "" {
			return nil, fmt.Errorf("verification is required")
		}
		body := map[string]any{"temp_token": tempToken, "verification": verification}
		if v := req.GetInt("client_id", 0); v > 0 {
			body["client_id"] = v
		}
		if v := req.GetString("client_secret", ""); v != "" {
			body["client_secret"] = v
		}
		data, err := c.Post("/platform/auth/two-factor/verify-recovery", body)
		if err != nil {
			return nil, fmt.Errorf("login_verify_two_factor_recovery: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func LoginWithEmail(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		email, err := req.RequireString("email")
		if err != nil || email == "" {
			return nil, fmt.Errorf("email is required")
		}
		body := map[string]any{"email": email}
		if v := req.GetInt("client_id", 0); v > 0 {
			body["client_id"] = v
		}
		if v := req.GetString("client_secret", ""); v != "" {
			body["client_secret"] = v
		}
		data, err := c.Post("/platform/auth/login-with-email", body)
		if err != nil {
			return nil, fmt.Errorf("login_with_email: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func LoginWithEmailToken(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		email, err := req.RequireString("email")
		if err != nil || email == "" {
			return nil, fmt.Errorf("email is required")
		}
		token, err := req.RequireString("token")
		if err != nil || token == "" {
			return nil, fmt.Errorf("token is required")
		}
		body := map[string]any{"email": email, "token": token}
		if v := req.GetInt("client_id", 0); v > 0 {
			body["client_id"] = v
		}
		if v := req.GetString("client_secret", ""); v != "" {
			body["client_secret"] = v
		}
		data, err := c.Post("/platform/auth/login-with-email-token", body)
		if err != nil {
			return nil, fmt.Errorf("login_with_email_token: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func Register(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		email, err := req.RequireString("email")
		if err != nil || email == "" {
			return nil, fmt.Errorf("email is required")
		}
		password, err := req.RequireString("password")
		if err != nil || password == "" {
			return nil, fmt.Errorf("password is required")
		}
		body := map[string]any{"email": email, "password": password}
		if v := req.GetString("first_name", ""); v != "" {
			body["first_name"] = v
		}
		if v := req.GetString("last_name", ""); v != "" {
			body["last_name"] = v
		}
		if v := req.GetString("password_confirmation", ""); v != "" {
			body["password_confirmation"] = v
		}
		if v := req.GetInt("accepted", -1); v >= 0 {
			body["accepted"] = v
		}
		if v := req.GetInt("client_id", 0); v > 0 {
			body["client_id"] = v
		}
		if v := req.GetString("client_secret", ""); v != "" {
			body["client_secret"] = v
		}
		data, err := c.Post("/platform/auth/register", body)
		if err != nil {
			return nil, fmt.Errorf("register: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func ForgetPassword(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		email, err := req.RequireString("email")
		if err != nil || email == "" {
			return nil, fmt.Errorf("email is required")
		}
		body := map[string]any{"email": email}
		if v := req.GetInt("client_id", 0); v > 0 {
			body["client_id"] = v
		}
		if v := req.GetString("client_secret", ""); v != "" {
			body["client_secret"] = v
		}
		data, err := c.Post("/platform/auth/password/forget", body)
		if err != nil {
			return nil, fmt.Errorf("forget_password: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func ResetPassword(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		email, err := req.RequireString("email")
		if err != nil || email == "" {
			return nil, fmt.Errorf("email is required")
		}
		token, err := req.RequireString("token")
		if err != nil || token == "" {
			return nil, fmt.Errorf("token is required")
		}
		password, err := req.RequireString("password")
		if err != nil || password == "" {
			return nil, fmt.Errorf("password is required")
		}
		body := map[string]any{"email": email, "token": token, "password": password}
		if v := req.GetString("password_confirmation", ""); v != "" {
			body["password_confirmation"] = v
		}
		if v := req.GetInt("client_id", 0); v > 0 {
			body["client_id"] = v
		}
		if v := req.GetString("client_secret", ""); v != "" {
			body["client_secret"] = v
		}
		data, err := c.Post("/platform/auth/password/reset", body)
		if err != nil {
			return nil, fmt.Errorf("reset_password: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func VerifyEmail(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		email, err := req.RequireString("email")
		if err != nil || email == "" {
			return nil, fmt.Errorf("email is required")
		}
		token, err := req.RequireString("token")
		if err != nil || token == "" {
			return nil, fmt.Errorf("token is required")
		}
		data, err := c.Get("/platform/auth/verify/"+email+"/"+token, nil)
		if err != nil {
			return nil, fmt.Errorf("verify_email: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func Logout(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := c.Post("/platform/auth/logout", nil)
		if err != nil {
			return nil, fmt.Errorf("logout: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
