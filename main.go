package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Poltio/poltio-mcp-server/client"
	"github.com/Poltio/poltio-mcp-server/oauth"
	"github.com/Poltio/poltio-mcp-server/store"
	"github.com/Poltio/poltio-mcp-server/tools"
)

var version = "dev"

type orgEntry struct {
	ID int `json:"id"`
}

// dbOrgSetter persists per-session org overrides to the grant store in bridge mode.
type dbOrgSetter struct{ db *store.Store }

func (d *dbOrgSetter) SetOrgOverride(ctx context.Context, grantID, orgID string) error {
	return d.db.SetOrgOverride(grantID, orgID)
}

func main() {
	token := os.Getenv("POLTIO_API_TOKEN")
	port := os.Getenv("PORT")

	// In bridge mode (PORT set), credentials are resolved per-request from the OAuth store.
	// POLTIO_API_TOKEN is only required for stdio single-tenant mode.
	if token == "" && port == "" {
		log.Fatal("POLTIO_API_TOKEN environment variable is required")
	}

	var c *client.PoltioClient
	if token != "" {
		c = client.New(token)

		data, err := c.GetOrganizations()
		if err != nil {
			log.Fatalf("failed to fetch organizations at startup: %v", err)
		}
		var orgs []orgEntry
		if err := json.Unmarshal(data, &orgs); err != nil {
			log.Fatalf("failed to parse organizations at startup: %v", err)
		}
		if len(orgs) == 0 {
			log.Fatal("no organizations found for this token — ensure your account belongs to at least one organization")
		}
		c.SetOrgID(strconv.Itoa(orgs[0].ID))
	}

	// orgSetter is set to a db-backed implementation inside the bridge block (port != "").
	// Tool registrations capture it by closure; it is read at call time, so the late
	// assignment is visible to all handlers before any requests arrive.
	var orgSetter tools.OrgOverrideSetter

	// mfn is a local type alias to keep the per-request wrapper lines short.
	type mfn = func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)

	// cc wraps a ContentClient tool factory so the per-request client from context is used
	// in bridge mode; falls back to the singleton c in stdio mode.
	cc := func(f func(tools.ContentClient) mfn) mfn {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if pc, err := client.FromContext(ctx); err == nil {
				return f(pc)(ctx, req)
			}
			if c != nil {
				return f(c)(ctx, req)
			}
			return mcp.NewToolResultError("bridge: no client available"), nil
		}
	}

	// sc wraps a StatsClient tool factory.
	sc := func(f func(tools.StatsClient) mfn) mfn {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if pc, err := client.FromContext(ctx); err == nil {
				return f(pc)(ctx, req)
			}
			if c != nil {
				return f(c)(ctx, req)
			}
			return mcp.NewToolResultError("bridge: no client available"), nil
		}
	}

	// uc wraps an UploadClient tool factory.
	uc := func(f func(tools.UploadClient) mfn) mfn {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if pc, err := client.FromContext(ctx); err == nil {
				return f(pc)(ctx, req)
			}
			if c != nil {
				return f(c)(ctx, req)
			}
			return mcp.NewToolResultError("bridge: no client available"), nil
		}
	}

	// oc wraps an OrgClient tool factory.
	oc := func(f func(tools.OrgClient) mfn) mfn {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if pc, err := client.FromContext(ctx); err == nil {
				return f(pc)(ctx, req)
			}
			if c != nil {
				return f(c)(ctx, req)
			}
			return mcp.NewToolResultError("bridge: no client available"), nil
		}
	}

	s := server.NewMCPServer("poltio", version)

	// ── Content ──────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_content",
		mcp.WithDescription("List Poltio content (polls, quizzes, tests) with optional pagination and filtering."),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("per_page", mcp.Description("Results per page (default: 12)")),
		mcp.WithString("type", mcp.Description("Filter by type: poll, set, test, quiz, this-that")),
		mcp.WithString("q", mcp.Description("Search query against title and description")),
		mcp.WithString("order", mcp.Description("Sort field: created_at (default), updated_at, vote_count, voter_count, type, id, end_date")),
		mcp.WithString("sort", mcp.Description("Sort direction: desc (default) or asc")),
	), cc(tools.ListContent))

	s.AddTool(mcp.NewTool(
		"get_content",
		mcp.WithDescription("Get a single Poltio content item with its metrics."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
	), cc(tools.GetContent))

	s.AddTool(mcp.NewTool(
		"create_content",
		mcp.WithDescription("Create a new Poltio content item. The item starts as a draft — call publish_content to make it live."),
		mcp.WithString("type", mcp.Description("Content type: poll, set, test, quiz, this-that"), mcp.Required()),
		mcp.WithString("title", mcp.Description("End-user facing title"), mcp.Required()),
		mcp.WithString("desc", mcp.Description("Cover screen description (optional)")),
		mcp.WithString("name", mcp.Description("Internal non-public name (optional)")),
		mcp.WithString("background", mcp.Description("Cover image path returned by upload_image (optional)")),
		mcp.WithString("alt", mcp.Description("Alt text for the cover image")),
		mcp.WithString("vertical_image", mcp.Description("Wide screen layout cover image path")),
		mcp.WithString("vertical_mobile_image", mcp.Description("Main cover image for single-column mobile view")),
		mcp.WithString("embed_footer_url", mcp.Description("URL for the footer image")),
		mcp.WithNumber("skip_start", mcp.Description("Skip cover card and start from first question: 0 (default) or 1")),
		mcp.WithNumber("skip_result", mcp.Description("Skip result card: 0 (default) or 1")),
		mcp.WithNumber("hide_results", mcp.Description("Hide vote percentages: 0 (default) or 1")),
		mcp.WithNumber("hide_counter", mcp.Description("Hide vote counter: 0 (default) or 1")),
		mcp.WithNumber("display_repeat", mcp.Description("Show play again button: 0 (default) or 1")),
		mcp.WithNumber("is_searchable", mcp.Description("Mark content as searchable to use search/filtering for results: 0 (default) or 1")),
		mcp.WithNumber("is_calculator", mcp.Description("Mark content as calculator to use formulas: 0 (default) or 1")),
		mcp.WithNumber("search_results_per_page", mcp.Description("Result count per page for searchable contents (default: 5)")),
		mcp.WithNumber("result_loading", mcp.Description("Display a loading screen between last question and result: 0 (default) or 1")),
		mcp.WithString("loading_next_question_label", mcp.Description("Custom loading label between questions")),
		mcp.WithString("loading_result_label", mcp.Description("Custom loading label between last question and result")),
		mcp.WithNumber("play_once", mcp.Description("Content playable only once per user: 0 (default) or 1")),
		mcp.WithString("play_once_strategy", mcp.Description("When to consider user as played: start or result (default: result)")),
		mcp.WithString("play_once_msg", mcp.Description("Custom message for play-once error screen")),
		mcp.WithString("play_once_img", mcp.Description("Custom image for play-once error screen")),
		mcp.WithString("play_once_link", mcp.Description("Custom button link for play-once error screen")),
		mcp.WithString("play_once_btn", mcp.Description("Custom button text for play-once error screen")),
		mcp.WithNumber("end_date_day", mcp.Description("Auto-finish content after this many days")),
		mcp.WithNumber("end_date_hour", mcp.Description("Auto-finish content after this many hours")),
		mcp.WithNumber("end_date_minute", mcp.Description("Auto-finish content after this many minutes")),
		mcp.WithString("attributes_json", mcp.Description(`Advanced settings as a JSON object. Fields: cal_formula, gives_feedback, show_timer, display_results, pool_question_count, time_limit, recom_title, noindex, canonical, redirect, keywords`)),
	), cc(tools.CreateContent))

	s.AddTool(mcp.NewTool(
		"publish_content",
		mcp.WithDescription("Publish a draft Poltio content item, making it publicly accessible."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
	), cc(tools.PublishContent))

	s.AddTool(mcp.NewTool(
		"list_drafts",
		mcp.WithDescription("List unpublished (draft) Poltio content items."),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("per_page", mcp.Description("Results per page (default: 12)")),
		mcp.WithString("type", mcp.Description("Filter by type: poll, set, test, quiz, this-that")),
		mcp.WithString("q", mcp.Description("Search query against title and description")),
		mcp.WithString("order", mcp.Description("Sort field: created_at (default), updated_at, vote_count, voter_count, type, id, end_date")),
		mcp.WithString("sort", mcp.Description("Sort direction: desc (default) or asc")),
	), cc(tools.ListDrafts))

	s.AddTool(mcp.NewTool(
		"update_content",
		mcp.WithDescription("Update an existing Poltio content item's metadata and images."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithString("title", mcp.Description("New end-user facing title")),
		mcp.WithString("desc", mcp.Description("New cover screen description")),
		mcp.WithString("name", mcp.Description("New internal non-public name")),
		mcp.WithString("type", mcp.Description("Content type: poll, set, test, quiz, this-that")),
		mcp.WithString("background", mcp.Description("Cover image path returned by upload_image")),
		mcp.WithString("alt", mcp.Description("Alt text for the cover image")),
		mcp.WithString("vertical_image", mcp.Description("Wide screen layout cover image path")),
		mcp.WithString("vertical_mobile_image", mcp.Description("Main cover image for single-column mobile view")),
		mcp.WithString("embed_footer_url", mcp.Description("URL for the footer image")),
		mcp.WithNumber("skip_start", mcp.Description("Skip cover card: 0 or 1")),
		mcp.WithNumber("skip_result", mcp.Description("Skip result card: 0 or 1")),
		mcp.WithNumber("hide_results", mcp.Description("Hide vote percentages: 0 or 1")),
		mcp.WithNumber("hide_counter", mcp.Description("Hide vote counter: 0 or 1")),
		mcp.WithNumber("display_repeat", mcp.Description("Show play again button: 0 or 1")),
		mcp.WithNumber("is_searchable", mcp.Description("Mark content as searchable: 0 or 1")),
		mcp.WithNumber("is_calculator", mcp.Description("Mark content as calculator: 0 or 1")),
		mcp.WithNumber("search_results_per_page", mcp.Description("Result count per page for searchable contents")),
		mcp.WithNumber("result_loading", mcp.Description("Display loading screen before result: 0 or 1")),
		mcp.WithString("loading_next_question_label", mcp.Description("Custom loading label between questions")),
		mcp.WithString("loading_result_label", mcp.Description("Custom loading label before result")),
		mcp.WithNumber("play_once", mcp.Description("Content playable once per user: 0 or 1")),
		mcp.WithString("play_once_strategy", mcp.Description("When to consider user as played: start or result")),
		mcp.WithString("play_once_msg", mcp.Description("Custom play-once message")),
		mcp.WithString("play_once_img", mcp.Description("Custom play-once image path")),
		mcp.WithString("play_once_link", mcp.Description("Custom play-once button link")),
		mcp.WithString("play_once_btn", mcp.Description("Custom play-once button text")),
		mcp.WithNumber("end_date_day", mcp.Description("Auto-finish after this many days")),
		mcp.WithNumber("end_date_hour", mcp.Description("Auto-finish after this many hours")),
		mcp.WithNumber("end_date_minute", mcp.Description("Auto-finish after this many minutes")),
		mcp.WithString("attributes_json", mcp.Description(`Advanced settings as a JSON object. Fields: cal_formula, gives_feedback, show_timer, display_results, pool_question_count, time_limit, recom_title, noindex, canonical, redirect, keywords`)),
	), cc(tools.UpdateContent))

	s.AddTool(mcp.NewTool(
		"delete_content",
		mcp.WithDescription("Permanently delete a Poltio content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
	), cc(tools.DeleteContent))

	s.AddTool(mcp.NewTool(
		"duplicate_content",
		mcp.WithDescription("Duplicate an existing Poltio content item into a new draft."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
	), cc(tools.DuplicateContent))

	s.AddTool(mcp.NewTool(
		"get_content_edit",
		mcp.WithDescription("Get full editable content object including all questions, answers, results, and conditions."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
	), cc(tools.GetContentEdit))

	s.AddTool(mcp.NewTool(
		"list_templates",
		mcp.WithDescription("List available Poltio content templates."),
	), cc(tools.ListTemplates))

	s.AddTool(mcp.NewTool(
		"get_template",
		mcp.WithDescription("Get a single content template with all its data."),
		mcp.WithString("public_id", mcp.Description("Template public identifier"), mcp.Required()),
	), cc(tools.GetTemplate))

	s.AddTool(mcp.NewTool(
		"use_template",
		mcp.WithDescription("Clone a content template into a new draft content item in your account."),
		mcp.WithString("public_id", mcp.Description("Template public identifier"), mcp.Required()),
	), cc(tools.UseTemplate))

	s.AddTool(mcp.NewTool(
		"get_content_results",
		mcp.WithDescription("Get paginated vote results for a content item (per-answer vote counts and stats)."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("per_page", mcp.Description("Results per page, 1-100 (default: 12)")),
		mcp.WithString("order_by", mcp.Description("Sort field: position (default), id, click_count, counter")),
		mcp.WithString("order_dir", mcp.Description("Sort direction: desc (default) or asc")),
	), cc(tools.GetContentResults))

	s.AddTool(mcp.NewTool(
		"get_content_sessions",
		mcp.WithDescription("Get paginated user sessions for a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("per_page", mcp.Description("Results per page (default: 12)")),
	), cc(tools.GetContentSessions))

	s.AddTool(mcp.NewTool(
		"get_content_metrics",
		mcp.WithDescription("Get time-series metrics for a content item grouped by period."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithString("period", mcp.Description("Grouping period: day, week, month, year"), mcp.Required()),
		mcp.WithString("start", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end", mcp.Description("End date (YYYY-MM-DD)")),
		mcp.WithString("metrics", mcp.Description("Comma-separated metric names: view,vote,voter,start,finish,conversion (defaults to all)")),
	), cc(tools.GetContentMetrics))

	s.AddTool(mcp.NewTool(
		"get_vote_sources",
		mcp.WithDescription("Get paginated vote sources (referring URLs) for a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
	), cc(tools.GetVoteSources))

	s.AddTool(mcp.NewTool(
		"get_sankey",
		mcp.WithDescription("Get Sankey diagram data showing user flow through a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
	), cc(tools.GetSankey))

	s.AddTool(mcp.NewTool(
		"get_sankey_users",
		mcp.WithDescription("Get users who took a specific path in the Sankey diagram."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithString("from_id", mcp.Description("Source node ID"), mcp.Required()),
		mcp.WithString("to_id", mcp.Description("Target node ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
	), cc(tools.GetSankeyUsers))

	s.AddTool(mcp.NewTool(
		"get_searchable_fields",
		mcp.WithDescription("Get all searchable and filterable fields defined for a searchable content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
	), cc(tools.GetSearchableFields))

	s.AddTool(mcp.NewTool(
		"get_session_urls",
		mcp.WithDescription("Get session URLs grouped by URL with session counts for a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
	), cc(tools.GetSessionUrls))

	// ── Image Upload ──────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"upload_image",
		mcp.WithDescription("Upload a base64-encoded image to Poltio. Returns a file path to use as the background field in content, questions, answers, or results. IMPORTANT: when creating images for quiz or test questions, the image must be thematic only — it must NOT contain text or visuals that reveal or hint at the correct answer."),
		mcp.WithString("image_base64", mcp.Description("Base64-encoded image data (no data URI prefix, just the raw base64 string)"), mcp.Required()),
		mcp.WithString("ext", mcp.Description("File extension without the dot, e.g. png, jpg, webp"), mcp.Required()),
		mcp.WithString("bucket", mcp.Description("Optional storage bucket name")),
	), uc(tools.UploadImage))

	// ── Questions ─────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"add_question",
		mcp.WithDescription("Add a new question to a draft content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Question text shown to the user")),
		mcp.WithString("answer_type", mcp.Description("Answer type: media, text, score, star_rating, yesno, free_text, free_number, autocomplete"), mcp.Required()),
		mcp.WithString("background", mcp.Description("Question image path returned by upload_image. For quiz/test content, the image must be thematic only — it must NOT contain text or visuals that reveal or hint at the correct answer.")),
		mcp.WithString("alt", mcp.Description("Alt text for the question image")),
		mcp.WithString("vertical_image", mcp.Description("Wide screen layout question image path")),
		mcp.WithNumber("allow_multiple_answers", mcp.Description("Allow selecting multiple answers: 0 (default) or 1")),
		mcp.WithNumber("is_skippable", mcp.Description("Allow skipping this question: 0 (default) or 1")),
		mcp.WithNumber("rotate_answers", mcp.Description("Randomise answer order per user: 0 (default) or 1")),
		mcp.WithString("name", mcp.Description("Question name, only for internal use")),
		mcp.WithNumber("max_multi_punch_answer", mcp.Description("How many answers you like to be voted in one session")),
		mcp.WithNumber("recommended_popular_answer", mcp.Description("How many answers do you want to display on auto complete type answers")),
		mcp.WithString("luv", mcp.Description("lead url variable for the question")),
		mcp.WithNumber("is_searchable", mcp.Description("If you set this as 1 you can use votes to this question to query or filter the search results")),
		mcp.WithString("cal_val_default", mcp.Description("The default value for calculator contents for this question if the answer doesn't have any specific value")),
		mcp.WithString("autocomplete_help", mcp.Description("Autocomplete help text if you want to customize it")),
		mcp.WithString("autocomplete_placeholder", mcp.Description("Autocomplete field placeholder text if you want to customize it")),
		mcp.WithNumber("position", mcp.Description("Numeric value for the Question position in the content")),
		mcp.WithString("conditions", mcp.Description("Comma seperated list of Answer IDs to use as display conditions")),
		mcp.WithNumber("condition_reverse", mcp.Description("Indicates if the conditions should be positive or negative")),
	), cc(tools.AddQuestion))

	s.AddTool(mcp.NewTool(
		"update_question",
		mcp.WithDescription("Update an existing question."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Question text shown to the user")),
		mcp.WithString("answer_type", mcp.Description("Answer type: media, text, score, star_rating, yesno, free_text, free_number, autocomplete"), mcp.Required()),
		mcp.WithString("background", mcp.Description("Question image path returned by upload_image. For quiz/test content, the image must be thematic only — it must NOT contain text or visuals that reveal or hint at the correct answer.")),
		mcp.WithString("alt", mcp.Description("Alt text for the question image")),
		mcp.WithString("vertical_image", mcp.Description("Wide screen layout question image path")),
		mcp.WithNumber("allow_multiple_answers", mcp.Description("Allow selecting multiple answers: 0 or 1")),
		mcp.WithNumber("is_skippable", mcp.Description("Allow skipping this question: 0 or 1")),
		mcp.WithNumber("rotate_answers", mcp.Description("Randomise answer order per user: 0 or 1")),
		mcp.WithString("name", mcp.Description("Question name, only for internal use")),
		mcp.WithNumber("max_multi_punch_answer", mcp.Description("How many answers you like to be voted in one session")),
		mcp.WithNumber("recommended_popular_answer", mcp.Description("How many answers do you want to display on auto complete type answers")),
		mcp.WithString("luv", mcp.Description("lead url variable for the question")),
		mcp.WithNumber("is_searchable", mcp.Description("If you set this as 1 you can use votes to this question to query or filter the search results")),
		mcp.WithString("cal_val_default", mcp.Description("The default value for calculator contents for this question if the answer doesn't have any specific value")),
		mcp.WithString("autocomplete_help", mcp.Description("Autocomplete help text if you want to customize it")),
		mcp.WithString("autocomplete_placeholder", mcp.Description("Autocomplete field placeholder text if you want to customize it")),
		mcp.WithNumber("position", mcp.Description("Numeric value for the Question position in the content")),
		mcp.WithString("conditions", mcp.Description("Comma seperated list of Answer IDs to use as display conditions")),
		mcp.WithNumber("condition_reverse", mcp.Description("Indicates if the conditions should be positive or negative")),
	), cc(tools.UpdateQuestion))

	s.AddTool(mcp.NewTool(
		"delete_question",
		mcp.WithDescription("Delete a question from a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
	), cc(tools.DeleteQuestion))

	// ── Answers ───────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"add_answer",
		mcp.WithDescription("Add a single answer to a question. Use background to set an image answer (upload_image first)."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Answer text (optional when background image is provided)")),
		mcp.WithString("background", mcp.Description("Answer image path returned by upload_image")),
		mcp.WithString("alt", mcp.Description("Alt text for the answer image")),
		mcp.WithString("luv", mcp.Description("Lead URL variable for the answer, e.g. '&color=blue'")),
		mcp.WithNumber("has_right_answer", mcp.Description("Enable right/wrong feedback for this question: 0 (default) or 1")),
		mcp.WithNumber("is_right_answer", mcp.Description("Mark this as the correct answer: 0 (default) or 1")),
		mcp.WithNumber("is_mutually_exclusive", mcp.Description("In multi-answer questions, selecting this deselects others: 0 (default) or 1")),
		mcp.WithString("search_query", mcp.Description("Search index query to run when this answer is selected")),
		mcp.WithString("search_filter", mcp.Description("Search index filter to apply when this answer is selected, e.g. 'color: [blue]'")),
		mcp.WithNumber("position", mcp.Description("Numeric position for this answer in the question")),
		mcp.WithNumber("max_vote", mcp.Description("If set, disables this answer once it reaches this vote count")),
		mcp.WithString("addon", mcp.Description("Additional info shared with GTM and PixelCodes after user selects this answer")),
		mcp.WithString("disabled_msg", mcp.Description("Custom message shown when this answer is disabled")),
	), cc(tools.AddAnswer))

	s.AddTool(mcp.NewTool(
		"add_answers_bulk",
		mcp.WithDescription("Add multiple answers to a question in one call. Provide one answer per line in the answers field."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithString("answers", mcp.Description("Answer texts, one per line"), mcp.Required()),
		mcp.WithNumber("remove_existing", mcp.Description("Remove existing answers before adding: 0 (default) or 1")),
	), cc(tools.AddAnswersBulk))

	s.AddTool(mcp.NewTool(
		"update_answer",
		mcp.WithDescription("Update an existing answer."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID"), mcp.Required()),
		mcp.WithString("title", mcp.Description("New answer text")),
		mcp.WithString("background", mcp.Description("Answer image path returned by upload_image")),
		mcp.WithString("alt", mcp.Description("Alt text for the answer image")),
		mcp.WithString("luv", mcp.Description("Lead URL variable for the answer")),
		mcp.WithNumber("has_right_answer", mcp.Description("Enable right/wrong feedback: 0 or 1")),
		mcp.WithNumber("is_right_answer", mcp.Description("Mark as correct answer: 0 or 1")),
		mcp.WithNumber("is_mutually_exclusive", mcp.Description("Mutually exclusive in multi-answer: 0 or 1")),
		mcp.WithString("search_query", mcp.Description("Search index query to run when this answer is selected")),
		mcp.WithString("search_filter", mcp.Description("Search index filter to apply when this answer is selected")),
		mcp.WithNumber("position", mcp.Description("Numeric position for this answer in the question")),
		mcp.WithNumber("max_vote", mcp.Description("If set, disables this answer once it reaches this vote count")),
		mcp.WithString("addon", mcp.Description("Additional info shared with GTM and PixelCodes after user selects this answer")),
		mcp.WithString("disabled_msg", mcp.Description("Custom message shown when this answer is disabled")),
	), cc(tools.UpdateAnswer))

	s.AddTool(mcp.NewTool(
		"delete_answer",
		mcp.WithDescription("Delete an answer from a question."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID"), mcp.Required()),
	), cc(tools.DeleteAnswer))

	s.AddTool(mcp.NewTool(
		"clone_answers",
		mcp.WithDescription("Copy all answers from a source question to a target question, replacing the target's existing answers."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("source_question_id", mcp.Description("Question to copy answers from"), mcp.Required()),
		mcp.WithNumber("target_question_id", mcp.Description("Question to copy answers to (existing answers will be removed)"), mcp.Required()),
	), cc(tools.CloneAnswers))

	s.AddTool(mcp.NewTool(
		"get_answer_order",
		mcp.WithDescription("Get the current answer order (positions) for a question."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
	), cc(tools.GetAnswerOrder))

	s.AddTool(mcp.NewTool(
		"update_answer_order",
		mcp.WithDescription("Reorder answers in a question. Provide a JSON array of {id, position} objects."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithString("answers", mcp.Description(`JSON array of position objects, e.g. [{"id":1,"position":2},{"id":2,"position":1}]`), mcp.Required()),
	), cc(tools.UpdateAnswerOrder))

	// ── Results ───────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"add_result",
		mcp.WithDescription("Add a result screen to a quiz or test content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Result title"), mcp.Required()),
		mcp.WithString("desc", mcp.Description("Result description")),
		mcp.WithString("background", mcp.Description("Result image path returned by upload_image")),
		mcp.WithString("alt", mcp.Description("Alt text for the result image")),
		mcp.WithString("luv", mcp.Description("Lead URL variable for the result, e.g. '&result=thanks'")),
		mcp.WithString("url", mcp.Description("Optional redirect URL shown on the result screen")),
		mcp.WithString("url_text", mcp.Description("Button label for the redirect URL")),
		mcp.WithString("search", mcp.Description("Main search terms for searchable content")),
		mcp.WithString("search2", mcp.Description("Secondary search terms for searchable content")),
		mcp.WithNumber("min_c", mcp.Description("Minimum score to reach this result (score-based content)")),
		mcp.WithNumber("max_c", mcp.Description("Maximum score for this result (score-based content)")),
		mcp.WithNumber("is_default", mcp.Description("Make this a catch-all default result: 0 (default) or 1")),
	), cc(tools.AddResult))

	s.AddTool(mcp.NewTool(
		"update_result",
		mcp.WithDescription("Update an existing result screen."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("result_id", mcp.Description("Result ID"), mcp.Required()),
		mcp.WithString("title", mcp.Description("New result title")),
		mcp.WithString("desc", mcp.Description("New result description")),
		mcp.WithString("background", mcp.Description("Result image path returned by upload_image")),
		mcp.WithString("alt", mcp.Description("Alt text for the result image")),
		mcp.WithString("luv", mcp.Description("Lead URL variable for the result")),
		mcp.WithString("url", mcp.Description("Redirect URL")),
		mcp.WithString("url_text", mcp.Description("Button label for the redirect URL")),
		mcp.WithString("search", mcp.Description("Main search terms for searchable content")),
		mcp.WithString("search2", mcp.Description("Secondary search terms for searchable content")),
		mcp.WithNumber("min_c", mcp.Description("Minimum score for this result")),
		mcp.WithNumber("max_c", mcp.Description("Maximum score for this result")),
		mcp.WithNumber("is_default", mcp.Description("Make this a catch-all default result: 0 or 1")),
	), cc(tools.UpdateResult))

	s.AddTool(mcp.NewTool(
		"delete_result",
		mcp.WithDescription("Delete a result screen from a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("result_id", mcp.Description("Result ID"), mcp.Required()),
	), cc(tools.DeleteResult))

	s.AddTool(mcp.NewTool(
		"set_answer_result_point",
		mcp.WithDescription("Set the point value that links an answer to a result (used in score-based quizzes and calculator tests)."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID"), mcp.Required()),
		mcp.WithNumber("content_result_id", mcp.Description("Result ID"), mcp.Required()),
		mcp.WithNumber("point", mcp.Description("Point value (≥ 0)"), mcp.Required()),
	), cc(tools.SetAnswerResultPoint))

	// ── Questions — conditions and order ─────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"get_content_conditions",
		mcp.WithDescription("List all questions in a content item that have display conditions attached."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
	), cc(tools.GetContentConditions))

	s.AddTool(mcp.NewTool(
		"add_question_condition",
		mcp.WithDescription("Add an answer as a display condition for a question (the question only shows if the given answer was selected)."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question to add condition to"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID that triggers the condition"), mcp.Required()),
		mcp.WithNumber("condition_reverse", mcp.Description("Invert the condition (hide instead of show): 0 (default) or 1")),
	), cc(tools.AddQuestionCondition))

	s.AddTool(mcp.NewTool(
		"remove_question_condition",
		mcp.WithDescription("Remove a single answer from a question's display conditions."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID to remove from conditions"), mcp.Required()),
	), cc(tools.RemoveQuestionCondition))

	s.AddTool(mcp.NewTool(
		"clear_question_conditions",
		mcp.WithDescription("Remove all display conditions from a question."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
	), cc(tools.ClearQuestionConditions))

	s.AddTool(mcp.NewTool(
		"get_question_order",
		mcp.WithDescription("Get the current question order (positions) for a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
	), cc(tools.GetQuestionOrder))

	s.AddTool(mcp.NewTool(
		"update_question_order",
		mcp.WithDescription("Reorder questions in a content item. Provide a JSON array of {id, position} objects."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithString("questions", mcp.Description(`JSON array of position objects, e.g. [{"id":1,"position":2},{"id":2,"position":1}]`), mcp.Required()),
	), cc(tools.UpdateQuestionOrder))

	s.AddTool(mcp.NewTool(
		"get_question_inputs",
		mcp.WithDescription("Get paginated free-text answer inputs submitted by voters for a question."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("per_page", mcp.Description("Results per page (default: 12)")),
		mcp.WithString("order", mcp.Description("Sort field: created_at (default), voter_id, id")),
		mcp.WithString("sort", mcp.Description("Sort direction: desc (default) or asc")),
	), cc(tools.GetQuestionInputs))

	// ── Lead attachment ───────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"set_content_lead",
		mcp.WithDescription("Attach a lead form to the cover screen of a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("lead_id", mcp.Description("Lead ID to attach"), mcp.Required()),
	), cc(tools.SetContentLead))

	s.AddTool(mcp.NewTool(
		"remove_content_lead",
		mcp.WithDescription("Remove the lead form from the cover screen of a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
	), cc(tools.RemoveContentLead))

	s.AddTool(mcp.NewTool(
		"set_question_lead",
		mcp.WithDescription("Attach a lead form to all answers of a question."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("lead_id", mcp.Description("Lead ID to attach"), mcp.Required()),
	), cc(tools.SetQuestionLead))

	s.AddTool(mcp.NewTool(
		"remove_question_lead",
		mcp.WithDescription("Remove the lead form from all answers of a question."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
	), cc(tools.RemoveQuestionLead))

	s.AddTool(mcp.NewTool(
		"set_answer_lead",
		mcp.WithDescription("Attach a lead form to a specific answer."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID"), mcp.Required()),
		mcp.WithNumber("lead_id", mcp.Description("Lead ID to attach"), mcp.Required()),
	), cc(tools.SetAnswerLead))

	s.AddTool(mcp.NewTool(
		"remove_answer_lead",
		mcp.WithDescription("Remove the lead form from a specific answer."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID"), mcp.Required()),
	), cc(tools.RemoveAnswerLead))

	s.AddTool(mcp.NewTool(
		"set_result_lead",
		mcp.WithDescription("Attach a lead form to a result screen."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("result_id", mcp.Description("Result ID"), mcp.Required()),
		mcp.WithNumber("lead_id", mcp.Description("Lead ID to attach"), mcp.Required()),
	), cc(tools.SetResultLead))

	s.AddTool(mcp.NewTool(
		"remove_result_lead",
		mcp.WithDescription("Remove the lead form from a result screen."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("result_id", mcp.Description("Result ID"), mcp.Required()),
	), cc(tools.RemoveResultLead))

	// ── Lead management ───────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_leads",
		mcp.WithDescription("List lead campaigns for this organization."),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
	), cc(tools.ListLeads))

	s.AddTool(mcp.NewTool(
		"create_lead",
		mcp.WithDescription("Create a new lead campaign."),
		mcp.WithString("name", mcp.Description("Human-readable name for the lead"), mcp.Required()),
		mcp.WithString("type", mcp.Description("Lead type: redirect, video, image, input, none"), mcp.Required()),
		mcp.WithString("msg", mcp.Description("Message displayed to the user")),
		mcp.WithString("fields", mcp.Description("Comma-separated input field names: gsm, email, name, comment")),
		mcp.WithString("title", mcp.Description("Window title")),
		mcp.WithString("button_value", mcp.Description("CTA button label")),
		mcp.WithString("redirect_url", mcp.Description("Redirect URL for redirect-type leads")),
		mcp.WithString("youtube_id", mcp.Description("YouTube video ID for video-type leads")),
		mcp.WithString("terms_conditions", mcp.Description("Terms and conditions text")),
		mcp.WithString("image", mcp.Description("Image path for the lead")),
		mcp.WithNumber("is_active", mcp.Description("Active state: 0 or 1")),
		mcp.WithNumber("mandatory", mcp.Description("Non-dismissable lead: 0 or 1")),
		mcp.WithNumber("tc_optional", mcp.Description("Terms and conditions checkbox optional: 0 or 1 (default: 1)")),
		mcp.WithNumber("tc2_optional", mcp.Description("Second terms and conditions checkbox optional: 0 or 1 (default: 1)")),
		mcp.WithNumber("auto_open", mcp.Description("Auto-redirect to URL: 0 (default) or 1")),
		mcp.WithNumber("auto_open_delay", mcp.Description("Auto-redirect delay in milliseconds (default: 2500)")),
		mcp.WithNumber("open_minimized", mcp.Description("Open lead in minimized state by default: 0 (default) or 1")),
		mcp.WithNumber("delay", mcp.Description("Delay in milliseconds before loading the lead")),
		mcp.WithString("link_target", mcp.Description("Link open target: self, parent, or blank")),
		mcp.WithString("tc_short", mcp.Description("Short text for the terms and conditions line")),
		mcp.WithString("tc2_short", mcp.Description("Short text for the second terms and conditions line")),
		mcp.WithString("tc_approve_button_label", mcp.Description("Custom button label for the accept section")),
		mcp.WithString("tc_reject_button_label", mcp.Description("Custom button label for the reject section")),
		mcp.WithString("tc2_approve_button_label", mcp.Description("Custom button label for the second accept section")),
		mcp.WithString("tc2_reject_button_label", mcp.Description("Custom button label for the second reject section")),
		mcp.WithString("custom_labels_json", mcp.Description(`Custom input field labels as JSON, e.g. {"email":"E-mail","gsm":"Phone"}`)),
	), cc(tools.CreateLead))

	s.AddTool(mcp.NewTool(
		"get_lead",
		mcp.WithDescription("Get a single lead campaign by ID."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
	), cc(tools.GetLead))

	s.AddTool(mcp.NewTool(
		"update_lead",
		mcp.WithDescription("Update a lead campaign."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Human-readable name")),
		mcp.WithString("type", mcp.Description("Lead type: redirect, video, image, input, none")),
		mcp.WithString("msg", mcp.Description("Message displayed to the user")),
		mcp.WithString("fields", mcp.Description("Comma-separated input fields")),
		mcp.WithString("title", mcp.Description("Window title")),
		mcp.WithString("button_value", mcp.Description("CTA button label")),
		mcp.WithString("redirect_url", mcp.Description("Redirect URL")),
		mcp.WithString("youtube_id", mcp.Description("YouTube video ID for video-type leads")),
		mcp.WithString("terms_conditions", mcp.Description("Terms and conditions text")),
		mcp.WithString("image", mcp.Description("Image path for the lead")),
		mcp.WithNumber("is_active", mcp.Description("Active state: 0 or 1")),
		mcp.WithNumber("mandatory", mcp.Description("Non-dismissable: 0 or 1")),
		mcp.WithNumber("tc_optional", mcp.Description("Terms and conditions checkbox optional: 0 or 1")),
		mcp.WithNumber("tc2_optional", mcp.Description("Second terms and conditions checkbox optional: 0 or 1")),
		mcp.WithNumber("auto_open", mcp.Description("Auto-redirect to URL: 0 or 1")),
		mcp.WithNumber("auto_open_delay", mcp.Description("Auto-redirect delay in milliseconds")),
		mcp.WithNumber("open_minimized", mcp.Description("Open lead in minimized state by default: 0 or 1")),
		mcp.WithNumber("delay", mcp.Description("Delay in milliseconds before loading the lead")),
		mcp.WithString("link_target", mcp.Description("Link open target: self, parent, or blank")),
		mcp.WithString("tc_short", mcp.Description("Short text for the terms and conditions line")),
		mcp.WithString("tc2_short", mcp.Description("Short text for the second terms and conditions line")),
		mcp.WithString("tc_approve_button_label", mcp.Description("Custom button label for the accept section")),
		mcp.WithString("tc_reject_button_label", mcp.Description("Custom button label for the reject section")),
		mcp.WithString("tc2_approve_button_label", mcp.Description("Custom button label for the second accept section")),
		mcp.WithString("tc2_reject_button_label", mcp.Description("Custom button label for the second reject section")),
		mcp.WithString("custom_labels_json", mcp.Description(`Custom input field labels as JSON, e.g. {"email":"E-mail","gsm":"Phone"}`)),
	), cc(tools.UpdateLead))

	s.AddTool(mcp.NewTool(
		"delete_lead",
		mcp.WithDescription("Delete a lead campaign."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
	), cc(tools.DeleteLead))

	s.AddTool(mcp.NewTool(
		"get_lead_inputs",
		mcp.WithDescription("Get paginated user inputs submitted through a lead form."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
	), cc(tools.GetLeadInputs))

	s.AddTool(mcp.NewTool(
		"get_lead_logs",
		mcp.WithDescription("Get paginated activation logs for a lead campaign."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
	), cc(tools.GetLeadLogs))

	s.AddTool(mcp.NewTool(
		"get_lead_codes",
		mcp.WithDescription("Get paginated coupon codes for a lead campaign."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
	), cc(tools.GetLeadCodes))

	s.AddTool(mcp.NewTool(
		"add_lead_codes",
		mcp.WithDescription("Add one or more coupon codes to a lead campaign (one code per line)."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		mcp.WithString("codes", mcp.Description("Coupon codes, one per line"), mcp.Required()),
		mcp.WithNumber("single_use", mcp.Description("Each code can only be used once: 0 (default) or 1")),
		mcp.WithNumber("remove_existing", mcp.Description("Remove existing codes first: 0 (default) or 1")),
	), cc(tools.AddLeadCodes))

	s.AddTool(mcp.NewTool(
		"delete_all_lead_codes",
		mcp.WithDescription("Remove ALL coupon codes from a lead campaign."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
	), cc(tools.DeleteAllLeadCodes))

	s.AddTool(mcp.NewTool(
		"update_lead_code",
		mcp.WithDescription("Update a single coupon code in a lead campaign."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		mcp.WithString("lead_coupon_code_id", mcp.Description("Coupon code ID"), mcp.Required()),
		mcp.WithString("code", mcp.Description("New code value"), mcp.Required()),
		mcp.WithNumber("single_use", mcp.Description("Single-use flag: 0 or 1")),
	), cc(tools.UpdateLeadCode))

	s.AddTool(mcp.NewTool(
		"delete_lead_code",
		mcp.WithDescription("Delete a single coupon code from a lead campaign."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		mcp.WithString("lead_coupon_code_id", mcp.Description("Coupon code ID"), mcp.Required()),
	), cc(tools.DeleteLeadCode))

	// ── Pixel codes ───────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_pixel_codes",
		mcp.WithDescription("List pixel code snippets for this organization."),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
	), cc(tools.ListPixelCodes))

	s.AddTool(mcp.NewTool(
		"create_pixel_code",
		mcp.WithDescription("Create a new pixel code snippet (iframe, img, or script tag HTML)."),
		mcp.WithString("name", mcp.Description("Human-readable name"), mcp.Required()),
		mcp.WithString("code", mcp.Description("HTML snippet containing the pixel code"), mcp.Required()),
	), cc(tools.CreatePixelCode))

	s.AddTool(mcp.NewTool(
		"update_pixel_code",
		mcp.WithDescription("Update an existing pixel code snippet."),
		mcp.WithNumber("pixel_code_id", mcp.Description("Pixel code ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Human-readable name")),
		mcp.WithString("code", mcp.Description("HTML snippet")),
	), cc(tools.UpdatePixelCode))

	s.AddTool(mcp.NewTool(
		"delete_pixel_code",
		mcp.WithDescription("Delete a pixel code snippet."),
		mcp.WithNumber("pixel_code_id", mcp.Description("Pixel code ID"), mcp.Required()),
	), cc(tools.DeletePixelCode))

	s.AddTool(mcp.NewTool(
		"set_content_pixel_code",
		mcp.WithDescription("Attach a pixel code to the cover screen of a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("pixel_code_id", mcp.Description("Pixel code ID"), mcp.Required()),
	), cc(tools.SetContentPixelCode))

	s.AddTool(mcp.NewTool(
		"remove_content_pixel_code",
		mcp.WithDescription("Remove the pixel code from the cover screen of a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
	), cc(tools.RemoveContentPixelCode))

	s.AddTool(mcp.NewTool(
		"set_question_pixel_code",
		mcp.WithDescription("Attach a pixel code to all answers of a question."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("pixel_code_id", mcp.Description("Pixel code ID"), mcp.Required()),
	), cc(tools.SetQuestionPixelCode))

	s.AddTool(mcp.NewTool(
		"remove_question_pixel_code",
		mcp.WithDescription("Remove the pixel code from all answers of a question."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
	), cc(tools.RemoveQuestionPixelCode))

	s.AddTool(mcp.NewTool(
		"set_answer_pixel_code",
		mcp.WithDescription("Attach a pixel code to a specific answer."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID"), mcp.Required()),
		mcp.WithNumber("pixel_code_id", mcp.Description("Pixel code ID"), mcp.Required()),
	), cc(tools.SetAnswerPixelCode))

	s.AddTool(mcp.NewTool(
		"remove_answer_pixel_code",
		mcp.WithDescription("Remove the pixel code from a specific answer."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID"), mcp.Required()),
	), cc(tools.RemoveAnswerPixelCode))

	s.AddTool(mcp.NewTool(
		"set_result_pixel_code",
		mcp.WithDescription("Attach a pixel code to a result screen (fires on result view)."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("result_id", mcp.Description("Result ID"), mcp.Required()),
		mcp.WithNumber("pixel_code_id", mcp.Description("Pixel code ID"), mcp.Required()),
	), cc(tools.SetResultPixelCode))

	s.AddTool(mcp.NewTool(
		"remove_result_pixel_code",
		mcp.WithDescription("Remove the view pixel code from a result screen."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("result_id", mcp.Description("Result ID"), mcp.Required()),
	), cc(tools.RemoveResultPixelCode))

	s.AddTool(mcp.NewTool(
		"set_result_click_pixel_code",
		mcp.WithDescription("Attach a pixel code to a result screen's click/CTA action."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("result_id", mcp.Description("Result ID"), mcp.Required()),
		mcp.WithNumber("pixel_code_id", mcp.Description("Pixel code ID"), mcp.Required()),
	), cc(tools.SetResultClickPixelCode))

	s.AddTool(mcp.NewTool(
		"remove_result_click_pixel_code",
		mcp.WithDescription("Remove the click pixel code from a result screen."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("result_id", mcp.Description("Result ID"), mcp.Required()),
	), cc(tools.RemoveResultClickPixelCode))

	// ── Themes ────────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_themes",
		mcp.WithDescription("List themes for this organization."),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
	), cc(tools.ListThemes))

	s.AddTool(mcp.NewTool(
		"get_default_theme",
		mcp.WithDescription("Get the default theme values to use as a base when creating a new theme."),
	), cc(tools.GetDefaultTheme))

	s.AddTool(mcp.NewTool(
		"get_theme",
		mcp.WithDescription("Get a single theme by ID."),
		mcp.WithNumber("theme_id", mcp.Description("Theme ID"), mcp.Required()),
	), cc(tools.GetTheme))

	s.AddTool(mcp.NewTool(
		"create_theme",
		mcp.WithDescription("Create a new theme. Call get_default_theme first to discover available fields, then pass overrides as fields_json."),
		mcp.WithString("name", mcp.Description("Internal name for the theme"), mcp.Required()),
		mcp.WithString("fields_json", mcp.Description("JSON object of theme fields to set (colors, fonts, etc.)")),
	), cc(tools.CreateTheme))

	s.AddTool(mcp.NewTool(
		"update_theme",
		mcp.WithDescription("Update an existing theme's fields."),
		mcp.WithNumber("theme_id", mcp.Description("Theme ID"), mcp.Required()),
		mcp.WithString("fields_json", mcp.Description("JSON object of theme fields to update"), mcp.Required()),
	), cc(tools.UpdateTheme))

	s.AddTool(mcp.NewTool(
		"delete_theme",
		mcp.WithDescription("Delete a theme (fails if the theme is currently in use)."),
		mcp.WithNumber("theme_id", mcp.Description("Theme ID"), mcp.Required()),
	), cc(tools.DeleteTheme))

	// ── Dashboard ─────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"get_dashboard",
		mcp.WithDescription("Get account dashboard data including recent content, profile, and aggregate counters."),
	), cc(tools.GetDashboard))

	s.AddTool(mcp.NewTool(
		"get_dashboard_summary",
		mcp.WithDescription("Get most recently active content stat summary for the dashboard."),
		mcp.WithString("start", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end", mcp.Description("End date (YYYY-MM-DD)")),
		mcp.WithNumber("take", mcp.Description("Number of items to return")),
	), cc(tools.GetDashboardSummary))

	s.AddTool(mcp.NewTool(
		"get_dashboard_metrics",
		mcp.WithDescription("Get account-wide time-series metrics grouped by period."),
		mcp.WithString("period", mcp.Description("Grouping period: day, week, month, year"), mcp.Required()),
		mcp.WithString("start", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end", mcp.Description("End date (YYYY-MM-DD)")),
		mcp.WithString("metrics", mcp.Description("Comma-separated metric names (defaults to all)")),
	), cc(tools.GetDashboardMetrics))

	// ── Sheet Hooks ───────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_sheet_hooks",
		mcp.WithDescription("List Google Sheet hooks for this organization."),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithString("public_id", mcp.Description("Filter by content public_id")),
	), cc(tools.ListSheetHooks))

	s.AddTool(mcp.NewTool(
		"create_sheet_hook",
		mcp.WithDescription("Create a new Google Sheet hook to stream votes into a sheet in real time."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithString("sheet_id", mcp.Description("Google Sheet ID (from the sheet URL)"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Internal name for the hook")),
		mcp.WithNumber("is_active", mcp.Description("Active state: 0 or 1 (default: 1)")),
	), cc(tools.CreateSheetHook))

	s.AddTool(mcp.NewTool(
		"get_sheet_hook",
		mcp.WithDescription("Get details of a Google Sheet hook."),
		mcp.WithNumber("hook_id", mcp.Description("Hook ID"), mcp.Required()),
	), cc(tools.GetSheetHook))

	s.AddTool(mcp.NewTool(
		"update_sheet_hook",
		mcp.WithDescription("Update an existing Google Sheet hook."),
		mcp.WithNumber("hook_id", mcp.Description("Hook ID"), mcp.Required()),
		mcp.WithString("sheet_id", mcp.Description("New Google Sheet ID")),
		mcp.WithString("name", mcp.Description("New internal name")),
		mcp.WithString("public_id", mcp.Description("Content public identifier")),
		mcp.WithNumber("is_active", mcp.Description("Active state: 0 or 1")),
	), cc(tools.UpdateSheetHook))

	s.AddTool(mcp.NewTool(
		"delete_sheet_hook",
		mcp.WithDescription("Delete a Google Sheet hook."),
		mcp.WithNumber("hook_id", mcp.Description("Hook ID"), mcp.Required()),
	), cc(tools.DeleteSheetHook))

	s.AddTool(mcp.NewTool(
		"get_sheet_hook_logs",
		mcp.WithDescription("Get execution logs for a Google Sheet hook."),
		mcp.WithNumber("hook_id", mcp.Description("Hook ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
	), cc(tools.GetSheetHookLogs))

	// ── Webhooks ──────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_webhooks",
		mcp.WithDescription("List webhooks for this organization."),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithString("public_id", mcp.Description("Filter by content public_id")),
	), cc(tools.ListWebhooks))

	s.AddTool(mcp.NewTool(
		"create_webhook",
		mcp.WithDescription("Create a webhook to receive vote/session data in real time."),
		mcp.WithString("url", mcp.Description("Webhook endpoint URL"), mcp.Required()),
		mcp.WithString("public_id", mcp.Description("Content public_id (omit for account-wide hooks)")),
		mcp.WithString("name", mcp.Description("Internal name")),
		mcp.WithNumber("is_active", mcp.Description("Active state: 0 or 1 (default: 1)")),
		mcp.WithNumber("delay", mcp.Description("Delay in seconds before firing")),
		mcp.WithNumber("send_leads", mcp.Description("Include lead data: 0 or 1")),
		mcp.WithNumber("send_answers", mcp.Description("Include answer data: 0 (default) or 1")),
		mcp.WithNumber("account_wide", mcp.Description("Fire for all content in account: 0 (default) or 1")),
		mcp.WithNumber("incomplete_send", mcp.Description("Fire for incomplete sessions: 0 (default) or 1")),
		mcp.WithNumber("incomplete_delay", mcp.Description("Seconds from session start to trigger incomplete webhook")),
		mcp.WithNumber("use_oauth", mcp.Description("Enable OAuth authentication for webhook: 0 or 1")),
		mcp.WithString("oauth_login_endpoint", mcp.Description("OAuth token API endpoint (required with use_oauth)")),
		mcp.WithString("oauth_request_body_json", mcp.Description("Additional OAuth request body fields as JSON")),
		mcp.WithString("oauth_request_headers_json", mcp.Description("Additional OAuth request headers as JSON")),
	), cc(tools.CreateWebhook))

	s.AddTool(mcp.NewTool(
		"get_webhook",
		mcp.WithDescription("Get details of a webhook."),
		mcp.WithNumber("hook_id", mcp.Description("Hook ID"), mcp.Required()),
	), cc(tools.GetWebhook))

	s.AddTool(mcp.NewTool(
		"update_webhook",
		mcp.WithDescription("Update an existing webhook."),
		mcp.WithNumber("hook_id", mcp.Description("Hook ID"), mcp.Required()),
		mcp.WithString("url", mcp.Description("New endpoint URL")),
		mcp.WithString("name", mcp.Description("New internal name")),
		mcp.WithNumber("is_active", mcp.Description("Active state: 0 or 1")),
		mcp.WithNumber("delay", mcp.Description("Delay in seconds")),
		mcp.WithNumber("send_leads", mcp.Description("Include lead data: 0 or 1")),
		mcp.WithNumber("send_answers", mcp.Description("Include answer data: 0 or 1")),
		mcp.WithString("public_id", mcp.Description("Content public_id")),
		mcp.WithNumber("account_wide", mcp.Description("Fire for all content in account: 0 or 1")),
		mcp.WithNumber("incomplete_send", mcp.Description("Fire for incomplete sessions: 0 or 1")),
		mcp.WithNumber("incomplete_delay", mcp.Description("Seconds from session start to trigger incomplete webhook")),
		mcp.WithNumber("use_oauth", mcp.Description("Enable OAuth authentication for webhook: 0 or 1")),
		mcp.WithString("oauth_login_endpoint", mcp.Description("OAuth token API endpoint")),
		mcp.WithString("oauth_request_body_json", mcp.Description("Additional OAuth request body fields as JSON")),
		mcp.WithString("oauth_request_headers_json", mcp.Description("Additional OAuth request headers as JSON")),
	), cc(tools.UpdateWebhook))

	s.AddTool(mcp.NewTool(
		"delete_webhook",
		mcp.WithDescription("Delete a webhook."),
		mcp.WithNumber("hook_id", mcp.Description("Hook ID"), mcp.Required()),
	), cc(tools.DeleteWebhook))

	s.AddTool(mcp.NewTool(
		"get_webhook_logs",
		mcp.WithDescription("Get execution logs for a webhook."),
		mcp.WithNumber("hook_id", mcp.Description("Hook ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
	), cc(tools.GetWebhookLogs))

	// ── Vote / Stats ──────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"get_voters",
		mcp.WithDescription("Get paginated list of voters for a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("per_page", mcp.Description("Results per page (default: 12)")),
		mcp.WithNumber("download", mcp.Description("Request report as a file via email: 0 or 1 (default: 1)")),
	), sc(tools.GetVoters))

	s.AddTool(mcp.NewTool(
		"get_conversion_time_stats",
		mcp.WithDescription("Get conversion time-series stats for the whole account or a specific content item."),
		mcp.WithNumber("content_id", mcp.Description("Filter to a specific content item by its integer ID (optional)")),
		mcp.WithString("start", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end", mcp.Description("End date (YYYY-MM-DD)")),
	), sc(tools.GetConversionTimeStats))

	// ── Reports ───────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_reports",
		mcp.WithDescription("List downloadable report requests."),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
	), cc(tools.ListReports))

	s.AddTool(mcp.NewTool(
		"create_report",
		mcp.WithDescription("Request a new downloadable report (sent to your account email)."),
		mcp.WithString("report", mcp.Description("Report type: content-sessions or content-voters"), mcp.Required()),
		mcp.WithString("public_id", mcp.Description("Content public_id for content-scoped reports")),
		mcp.WithNumber("base_id", mcp.Description("Base ID (required when no public_id is given)")),
	), cc(tools.CreateReport))

	// ── Data Sources ──────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_data_sources",
		mcp.WithDescription("List data sources connected to this account."),
	), cc(tools.ListDataSources))

	s.AddTool(mcp.NewTool(
		"create_data_source",
		mcp.WithDescription("Submit a new data source (XML/JSON feed URL) for review."),
		mcp.WithString("name", mcp.Description("Human-readable name"), mcp.Required()),
		mcp.WithString("source", mcp.Description("Fully qualified feed URL"), mcp.Required()),
		mcp.WithString("type", mcp.Description("Feed format: xml or json"), mcp.Required()),
		mcp.WithString("notes", mcp.Description("Optional notes for the review team")),
	), cc(tools.CreateDataSource))

	s.AddTool(mcp.NewTool(
		"delete_data_source",
		mcp.WithDescription("Remove a data source submission."),
		mcp.WithNumber("data_source_id", mcp.Description("Data source ID"), mcp.Required()),
	), cc(tools.DeleteDataSource))

	s.AddTool(mcp.NewTool(
		"add_data_source_note",
		mcp.WithDescription("Add a note to a data source request (for communication with the review team)."),
		mcp.WithNumber("data_source_id", mcp.Description("Data source ID"), mcp.Required()),
		mcp.WithString("notes", mcp.Description("Note text"), mcp.Required()),
	), cc(tools.AddDataSourceNote))

	s.AddTool(mcp.NewTool(
		"upload_data_source",
		mcp.WithDescription("Upload a file (JSON, XML, CSV, or TXT) as a new data source."),
		mcp.WithString("file_base64", mcp.Description("Base64-encoded file content"), mcp.Required()),
		mcp.WithString("filename", mcp.Description("Filename with extension, e.g. feed.json, data.csv"), mcp.Required()),
	), uc(tools.UploadDataSource))

	// ── Domains ───────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_domains",
		mcp.WithDescription("List custom domains configured for this account."),
	), cc(tools.ListDomains))

	s.AddTool(mcp.NewTool(
		"create_domain",
		mcp.WithDescription("Add a new custom domain. You will need to verify it via DNS after creation."),
		mcp.WithString("domain", mcp.Description("Domain or subdomain, e.g. poltio.yourdomain.com"), mcp.Required()),
		mcp.WithNumber("is_default", mcp.Description("Set as default domain: 0 or 1")),
		mcp.WithNumber("is_active", mcp.Description("Enable the domain: 0 or 1")),
	), cc(tools.CreateDomain))

	s.AddTool(mcp.NewTool(
		"update_domain",
		mcp.WithDescription("Update a custom domain's settings."),
		mcp.WithNumber("domain_id", mcp.Description("Domain ID"), mcp.Required()),
		mcp.WithString("domain", mcp.Description("New domain value")),
		mcp.WithNumber("is_default", mcp.Description("Set as default: 0 or 1")),
		mcp.WithNumber("is_active", mcp.Description("Enable/disable: 0 or 1")),
	), cc(tools.UpdateDomain))

	s.AddTool(mcp.NewTool(
		"delete_domain",
		mcp.WithDescription("Delete a custom domain."),
		mcp.WithNumber("domain_id", mcp.Description("Domain ID"), mcp.Required()),
	), cc(tools.DeleteDomain))

	// ── Widgets ───────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_widgets",
		mcp.WithDescription("List your existing dynamic widgets."),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithString("public_id", mcp.Description("Filter by content public_id")),
	), cc(tools.ListWidgets))

	s.AddTool(mcp.NewTool(
		"create_widget",
		mcp.WithDescription("Create a new dynamic widget."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Widget name")),
		mcp.WithNumber("is_default", mcp.Description("Set as default widget: 0 or 1")),
		mcp.WithNumber("is_active", mcp.Description("Enable the widget: 0 or 1")),
		mcp.WithString("urls", mcp.Description("Comma-separated URLs for the widget")),
	), cc(tools.CreateWidget))

	s.AddTool(mcp.NewTool(
		"get_widget",
		mcp.WithDescription("Get a single Dynamic Widget."),
		mcp.WithNumber("widget_id", mcp.Description("Widget ID"), mcp.Required()),
	), cc(tools.GetWidget))

	s.AddTool(mcp.NewTool(
		"update_widget",
		mcp.WithDescription("Update an existing Dynamic Widget."),
		mcp.WithNumber("widget_id", mcp.Description("Widget ID"), mcp.Required()),
		mcp.WithString("public_id", mcp.Description("Content public identifier")),
		mcp.WithString("name", mcp.Description("Widget name")),
		mcp.WithNumber("is_default", mcp.Description("Set as default widget: 0 or 1")),
		mcp.WithNumber("is_active", mcp.Description("Enable the widget: 0 or 1")),
		mcp.WithString("urls", mcp.Description("Comma-separated URLs for the widget")),
	), cc(tools.UpdateWidget))

	s.AddTool(mcp.NewTool(
		"delete_widget",
		mcp.WithDescription("Delete an existing dynamic widget."),
		mcp.WithNumber("widget_id", mcp.Description("Widget ID"), mcp.Required()),
	), cc(tools.DeleteWidget))

	// ── Settings ──────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"update_settings",
		mcp.WithDescription("Update your account's username, email, or profile photo."),
		mcp.WithString("username", mcp.Description("New unique username")),
		mcp.WithString("email", mcp.Description("New email address (requires re-verification)")),
		mcp.WithString("photo", mcp.Description("Profile photo file path")),
	), cc(tools.UpdateSettings))

	s.AddTool(mcp.NewTool(
		"update_password",
		mcp.WithDescription("Change your account password."),
		mcp.WithString("password", mcp.Description("Current password"), mcp.Required()),
		mcp.WithString("new_password", mcp.Description("New password"), mcp.Required()),
		mcp.WithString("new_password_confirmation", mcp.Description("New password confirmation"), mcp.Required()),
	), cc(tools.UpdatePassword))

	s.AddTool(mcp.NewTool(
		"resend_verification",
		mcp.WithDescription("Resend the account email verification link."),
	), cc(tools.ResendVerification))

	s.AddTool(mcp.NewTool(
		"accept_terms",
		mcp.WithDescription("Accept the Poltio terms and conditions for the current account."),
	), cc(tools.AcceptTerms))

	s.AddTool(mcp.NewTool(
		"setup_two_factor",
		mcp.WithDescription("Begin two-factor authentication setup. Returns a base64 QR code image to scan with an authenticator app."),
	), cc(tools.SetupTwoFactor))

	s.AddTool(mcp.NewTool(
		"verify_two_factor",
		mcp.WithDescription("Confirm 2FA setup with a TOTP verification code. Returns recovery codes on success."),
		mcp.WithString("verification", mcp.Description("6-digit TOTP code from your authenticator app"), mcp.Required()),
	), cc(tools.VerifyTwoFactor))

	s.AddTool(mcp.NewTool(
		"disable_two_factor",
		mcp.WithDescription("Disable two-factor authentication on the current account. Requires a TOTP verification code."),
		mcp.WithString("verification", mcp.Description("6-digit TOTP code from your authenticator app"), mcp.Required()),
	), cc(tools.DisableTwoFactor))

	s.AddTool(mcp.NewTool(
		"reset_two_factor_recovery_codes",
		mcp.WithDescription("Regenerate 2FA recovery codes. Existing codes are invalidated. Requires a TOTP verification code."),
		mcp.WithString("verification", mcp.Description("6-digit TOTP code from your authenticator app"), mcp.Required()),
	), cc(tools.ResetTwoFactorRecoveryCodes))

	s.AddTool(mcp.NewTool(
		"list_conversion_settings",
		mcp.WithDescription("List conversion tracking URLs defined for this account."),
	), cc(tools.ListConversionSettings))

	s.AddTool(mcp.NewTool(
		"create_conversion_setting",
		mcp.WithDescription("Add a new checkout success URL for conversion tracking."),
		mcp.WithString("url", mcp.Description("Checkout success page URL"), mcp.Required()),
		mcp.WithNumber("catch_all", mcp.Description("Report all conversions: 0 or 1")),
	), cc(tools.CreateConversionSetting))

	s.AddTool(mcp.NewTool(
		"update_conversion_setting",
		mcp.WithDescription("Update an existing conversion tracking URL."),
		mcp.WithNumber("conversion_setting_id", mcp.Description("Conversion setting ID"), mcp.Required()),
		mcp.WithString("url", mcp.Description("New URL")),
		mcp.WithNumber("catch_all", mcp.Description("Report all conversions: 0 or 1")),
	), cc(tools.UpdateConversionSetting))

	s.AddTool(mcp.NewTool(
		"delete_conversion_setting",
		mcp.WithDescription("Delete a conversion tracking URL."),
		mcp.WithNumber("conversion_setting_id", mcp.Description("Conversion setting ID"), mcp.Required()),
	), cc(tools.DeleteConversionSetting))

	// ── Organizations ─────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_organizations",
		mcp.WithDescription("List Poltio organizations the current user belongs to, including their role in each."),
	), oc(tools.ListOrganizations))

	s.AddTool(mcp.NewTool(
		"switch_organization",
		mcp.WithDescription("Switch the active organization context. All subsequent tool calls will operate under the selected organization."),
		mcp.WithNumber("id", mcp.Description("Organization ID (from list_organizations)"), mcp.Required()),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if pc, err := client.FromContext(ctx); err == nil {
			return tools.SwitchOrganization(pc, orgSetter)(ctx, req)
		}
		if c != nil {
			return tools.SwitchOrganization(c, orgSetter)(ctx, req)
		}
		return mcp.NewToolResultError("bridge: no client available"), nil
	})

	s.AddTool(mcp.NewTool(
		"get_organization",
		mcp.WithDescription("Get an organization's details including members and pending invites."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
	), cc(tools.GetOrganization))

	s.AddTool(mcp.NewTool(
		"update_organization",
		mcp.WithDescription("Update an organization's name."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("New organization name"), mcp.Required()),
	), cc(tools.UpdateOrganization))

	s.AddTool(mcp.NewTool(
		"invite_org_member",
		mcp.WithDescription("Invite a new member to an organization via email."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithString("email", mcp.Description("Email address of the user to invite"), mcp.Required()),
		mcp.WithString("role", mcp.Description("Role to assign: admin, user, or viewer"), mcp.Required()),
	), cc(tools.InviteOrgMember))

	s.AddTool(mcp.NewTool(
		"join_organization",
		mcp.WithDescription("Join an organization using an invite token from an invitation email."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithString("token", mcp.Description("Invite token from the invitation email"), mcp.Required()),
	), cc(tools.JoinOrganization))

	s.AddTool(mcp.NewTool(
		"leave_organization",
		mcp.WithDescription("Leave an organization (cannot be used by the owner)."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
	), cc(tools.LeaveOrganization))

	s.AddTool(mcp.NewTool(
		"cancel_org_invite",
		mcp.WithDescription("Cancel a pending organization invitation by email."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithString("email", mcp.Description("Email of the pending invite to cancel"), mcp.Required()),
	), cc(tools.CancelOrgInvite))

	s.AddTool(mcp.NewTool(
		"remove_org_member",
		mcp.WithDescription("Remove a member from an organization."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithNumber("user_id", mcp.Description("User ID of the member to remove"), mcp.Required()),
	), cc(tools.RemoveOrgMember))

	s.AddTool(mcp.NewTool(
		"update_org_member",
		mcp.WithDescription("Update a member's role in an organization."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithNumber("user_id", mcp.Description("User ID of the member"), mcp.Required()),
		mcp.WithString("role", mcp.Description("New role: admin, user, or viewer"), mcp.Required()),
	), cc(tools.UpdateOrgMember))

	// ── Misc ──────────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"search_playground",
		mcp.WithDescription("Test search queries and filters against a searchable content item."),
		mcp.WithNumber("content_id", mcp.Description("Content integer ID")),
		mcp.WithString("public_id", mcp.Description("Content public_id (use when content_id is unknown)")),
		mcp.WithString("query_json", mcp.Description(`JSON array of search terms, e.g. ["red shoes","running"]`)),
		mcp.WithString("filter_json", mcp.Description(`JSON array of filter expressions, e.g. ["price: [10...100]"]`)),
	), cc(tools.SearchPlayground))

	s.AddTool(mcp.NewTool(
		"check_snippet_page",
		mcp.WithDescription("Check if a page URL has the Poltio snippet active and receiving requests in the last 48 hours."),
		mcp.WithString("url", mcp.Description("Fully qualified page URL to check"), mcp.Required()),
	), cc(tools.CheckSnippetPage))

	s.AddTool(mcp.NewTool(
		"create_short_link",
		mcp.WithDescription("Create a polt.io shortened URL from any long URL."),
		mcp.WithString("url", mcp.Description("Fully qualified URL to shorten"), mcp.Required()),
	), cc(tools.CreateShortLink))

	// ── Subscription ──────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_subscription_tiers",
		mcp.WithDescription("List available subscription tiers and their features."),
	), cc(tools.ListSubscriptionTiers))

	s.AddTool(mcp.NewTool(
		"create_subscription",
		mcp.WithDescription("Create a new subscription for the current organization."),
		mcp.WithNumber("tier_id", mcp.Description("Subscription tier ID (from list_subscription_tiers)"), mcp.Required()),
		mcp.WithString("period", mcp.Description("Billing period: month or year"), mcp.Required()),
	), cc(tools.CreateSubscription))

	if port != "" {
		serverURL := os.Getenv("SERVER_URL")
		devMode := os.Getenv("BRIDGE_DEV_MODE") == "true"
		if devMode {
			if serverURL == "" {
				log.Fatal("SERVER_URL env var is required in bridge mode")
			}
		} else {
			if err := oauth.ValidateServerURL(serverURL); err != nil {
				log.Fatalf("bridge mode startup: %v", err)
			}
		}

		dbPath := os.Getenv("DATABASE_PATH")
		if dbPath == "" {
			dbPath = "bridge.db"
		}
		db, err := store.Open(dbPath)
		if err != nil {
			log.Fatalf("bridge mode: open store: %v", err)
		}
		defer db.Close()

		encKey, err := store.KeyFromEnv()
		if err != nil {
			log.Fatalf("bridge mode: %v", err)
		}

		// Wire the per-session org-override setter so SwitchOrganization persists to the DB.
		orgSetter = &dbOrgSetter{db: db}

		// Sweep goroutine: hourly cleanup of expired clients, pending grants, and needs_reconnect grants.
		go func() {
			for range time.NewTicker(time.Hour).C {
				n, err := db.SweepGrants(10*time.Minute, 30*24*time.Hour)
				if err != nil {
					log.Printf("sweep error: %v", err)
				} else {
					log.Printf("sweep: deleted %d expired records", n)
				}
			}
		}()

		// Auth middleware: short-circuit with a tool error when the request has no valid grant.
		s.Use(func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
			return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				if oauth.NeedsAuth(ctx) {
					return mcp.NewToolResultError("not authenticated — reconnect your Poltio connector"), nil
				}
				if oauth.NeedsReconnect(ctx) {
					return mcp.NewToolResultError("your Poltio access has expired or was revoked — reconnect at " + serverURL + "/authorize"), nil
				}
				return next(ctx, req)
			}
		})

		prmHandler, asmHandler := oauth.MetadataHandlers(serverURL)

		mcpServer := server.NewStreamableHTTPServer(
			s,
			server.WithEndpointPath("/mcp"),
			server.WithStreamableHTTPCORS(server.WithCORSAllowedOrigins("*")),
			server.WithHTTPContextFunc(oauth.BridgeContextFunc(db, encKey, "")),
		)

		mux := http.NewServeMux()
		mux.HandleFunc("/.well-known/oauth-protected-resource", prmHandler)
		mux.HandleFunc("/.well-known/oauth-authorization-server", asmHandler)
		mux.HandleFunc("/register", oauth.RegisterHandler(db, oauth.RegisterConfig{}))
		mux.HandleFunc("/authorize", oauth.AuthorizeHandler(db, 10*time.Minute))
		mux.HandleFunc("/consent", oauth.ConsentHandler(db, serverURL, encKey, 10*time.Minute, ""))
		mux.HandleFunc("/token", oauth.TokenHandler(db))
		mux.HandleFunc("/revoke", oauth.RevokeHandler(db))
		// /mcp: require Authorization header (pre-check); BridgeContextFunc does full token validation.
		mux.Handle("/mcp", oauth.UnauthorizedMCPMiddleware(serverURL, mcpServer))
		mux.Handle("/mcp/", oauth.UnauthorizedMCPMiddleware(serverURL, mcpServer))

		log.Printf("poltio-mcp-server bridge mode listening on :%s", port)
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
