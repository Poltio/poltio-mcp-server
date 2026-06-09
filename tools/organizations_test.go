package tools_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Poltio/poltio-mcp-server/client"
	"github.com/Poltio/poltio-mcp-server/tools"
)

// mockOrgClient implements tools.OrgClient for testing SwitchOrganization.
type mockOrgClient struct {
	orgsData []byte
	orgsErr  error
	setOrgID func(id string)
}

func (m *mockOrgClient) GetOrganizations() ([]byte, error) {
	return m.orgsData, m.orgsErr
}

func (m *mockOrgClient) SetOrgID(id string) {
	if m.setOrgID != nil {
		m.setOrgID(id)
	}
}

// mockOverrideSetter implements tools.OrgOverrideSetter for testing bridge mode.
type mockOverrideSetter struct {
	calledWithGrantID string
	calledWithOrgID   string
	err               error
}

func (m *mockOverrideSetter) SetOrgOverride(ctx context.Context, grantID, orgID string) error {
	m.calledWithGrantID = grantID
	m.calledWithOrgID = orgID
	return m.err
}

// Test 7: SwitchOrganization with nil overrideSetter calls SetOrgID (stdio path).
func TestSwitchOrganization_NilOverrideSetter_CallsSetOrgID(t *testing.T) {
	var gotOrgID string
	orgClient := &mockOrgClient{
		setOrgID: func(id string) {
			gotOrgID = id
		},
	}

	handler := tools.SwitchOrganization(orgClient, nil)
	result, err := handler(context.Background(), callRequest(map[string]any{"id": float64(42)}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if gotOrgID != "42" {
		t.Errorf("SetOrgID: want %q, got %q", "42", gotOrgID)
	}
}

// Test 8: SwitchOrganization with overrideSetter calls SetOrgOverride with correct grantID.
func TestSwitchOrganization_WithOverrideSetter_CallsSetOrgOverride(t *testing.T) {
	orgClient := &mockOrgClient{}
	setter := &mockOverrideSetter{}

	// Seed the grantID in context.
	ctx := client.WithGrantID(context.Background(), "grant123")

	handler := tools.SwitchOrganization(orgClient, setter)
	result, err := handler(ctx, callRequest(map[string]any{"id": float64(99)}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if setter.calledWithGrantID != "grant123" {
		t.Errorf("grantID: want %q, got %q", "grant123", setter.calledWithGrantID)
	}
	if setter.calledWithOrgID != "99" {
		t.Errorf("orgID: want %q, got %q", "99", setter.calledWithOrgID)
	}
}

// Test 8b: When overrideSetter is non-nil but grantID is empty, falls back to SetOrgID.
func TestSwitchOrganization_WithOverrideSetter_NoGrantID_FallsBackToSetOrgID(t *testing.T) {
	var gotOrgID string
	orgClient := &mockOrgClient{
		setOrgID: func(id string) {
			gotOrgID = id
		},
	}
	setter := &mockOverrideSetter{}

	// No grantID in context (empty string from GrantIDFromContext).
	handler := tools.SwitchOrganization(orgClient, setter)
	_, err := handler(context.Background(), callRequest(map[string]any{"id": float64(7)}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotOrgID != "7" {
		t.Errorf("SetOrgID: want %q, got %q", "7", gotOrgID)
	}
	// overrideSetter should NOT have been called.
	if setter.calledWithGrantID != "" {
		t.Errorf("expected SetOrgOverride not called, but got grantID %q", setter.calledWithGrantID)
	}
}

// Test 8c: When overrideSetter.SetOrgOverride returns an error, SwitchOrganization propagates it.
func TestSwitchOrganization_OverrideSetterError_Propagated(t *testing.T) {
	orgClient := &mockOrgClient{}
	setter := &mockOverrideSetter{err: errors.New("db write failed")}

	ctx := client.WithGrantID(context.Background(), "grant-err")
	handler := tools.SwitchOrganization(orgClient, setter)
	_, err := handler(ctx, callRequest(map[string]any{"id": float64(5)}))
	if err == nil {
		t.Fatal("expected error from SetOrgOverride, got nil")
	}
	if !errors.Is(err, setter.err) {
		t.Errorf("error should wrap setter.err: got %v", err)
	}
}
