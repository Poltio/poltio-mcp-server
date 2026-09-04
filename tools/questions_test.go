package tools_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Poltio/poltio-mcp-server/tools"
)

func TestAddQuestion_ForwardsSliderOptions(t *testing.T) {
	var got map[string]any
	mock := &mockClient{
		postFunc: func(path string, body any) ([]byte, error) {
			got = body.(map[string]any)
			return []byte(`{}`), nil
		},
	}

	_, err := tools.AddQuestion(mock)(context.Background(), callRequest(map[string]any{
		"public_id":    "abc",
		"answer_type":  "slider",
		"title":        "How much?",
		"options_json": `{"min_label":"Not at all","max_label":"Very much"}`,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	opts, ok := got["options"].(map[string]any)
	if !ok {
		t.Fatalf("options not forwarded, body: %v", got)
	}
	if opts["min_label"] != "Not at all" || opts["max_label"] != "Very much" {
		t.Errorf("wrong labels: %v", opts)
	}
}

func TestAddQuestion_OmitsOptionsWhenAbsent(t *testing.T) {
	var got map[string]any
	mock := &mockClient{
		postFunc: func(path string, body any) ([]byte, error) {
			got = body.(map[string]any)
			return []byte(`{}`), nil
		},
	}

	_, err := tools.AddQuestion(mock)(context.Background(), callRequest(map[string]any{
		"public_id":   "abc",
		"answer_type": "text",
		"title":       "Pick one",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := got["options"]; ok {
		t.Errorf("options sent when not requested: %v", got)
	}
}

func TestUpdateQuestion_RejectsInvalidOptionsJSON(t *testing.T) {
	mock := &mockClient{
		putFunc: func(path string, body any) ([]byte, error) {
			t.Fatal("request sent despite invalid options_json")
			return nil, nil
		},
	}

	_, err := tools.UpdateQuestion(mock)(context.Background(), callRequest(map[string]any{
		"public_id":    "abc",
		"question_id":  1,
		"answer_type":  "slider",
		"title":        "How much?",
		"options_json": `{not json`,
	}))
	if err == nil || !strings.Contains(err.Error(), "options_json") {
		t.Errorf("expected options_json error, got %v", err)
	}
}

func TestAddQuestion_MissingTitleReturnsError(t *testing.T) {
	mock := &mockClient{
		postFunc: func(path string, body any) ([]byte, error) {
			t.Fatal("request sent without a title")
			return nil, nil
		},
	}

	_, err := tools.AddQuestion(mock)(context.Background(), callRequest(map[string]any{
		"public_id":   "abc",
		"answer_type": "text",
	}))
	if err == nil || !strings.Contains(err.Error(), "title is required") {
		t.Errorf("expected title error, got %v", err)
	}
}
