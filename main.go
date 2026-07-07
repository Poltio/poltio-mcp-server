package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Poltio/poltio-mcp-server/client"
	"github.com/Poltio/poltio-mcp-server/tools"
)

var version = "dev"

const destructiveWarning = "\n\n⚠️ DESTRUCTIVE: This permanently affects data in the currently selected organization, which may be a live production organization with real content and real users. Ask the user for explicit confirmation before calling this tool."

// destructive appends the shared destructive-operation warning to the tool
// description and sets the MCP destructiveHint annotation. Must be listed
// after WithDescription, since it appends to the description set there.
func destructive() mcp.ToolOption {
	return func(t *mcp.Tool) {
		t.Description += destructiveWarning
		mcp.WithDestructiveHintAnnotation(true)(t)
	}
}

type orgEntry struct {
	ID int `json:"id"`
}

func main() {
	token := os.Getenv("POLTIO_API_TOKEN")
	port := os.Getenv("PORT")
	if port == "" && token == "" {
		log.Fatal("POLTIO_API_TOKEN environment variable is required")
	}

	c := client.New(token)

	if token != "" {
		data, err := c.GetOrganizations()
		if err != nil {
			if port == "" {
				log.Fatalf("failed to fetch organizations at startup: %v", err)
			}
			log.Printf("warning: failed to fetch organizations at startup: %v", err)
		} else {
			var orgs []orgEntry
			if err := json.Unmarshal(data, &orgs); err != nil {
				log.Fatalf("failed to parse organizations at startup: %v", err)
			}
			if len(orgs) == 0 && port == "" {
				log.Fatal("no organizations found for this token — ensure your account belongs to at least one organization")
			}
			if len(orgs) > 0 {
				c.SetOrgID(strconv.Itoa(orgs[0].ID))
			}
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
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.ListContent(c))

	s.AddTool(mcp.NewTool(
		"get_content",
		mcp.WithDescription("Get a single Poltio content item with its metrics."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetContent(c))

	s.AddTool(mcp.NewTool(
		"create_content",
		mcp.WithDescription(`Create a new Poltio content item. It starts as a draft — call publish_content to make it live.

A Poltio content item is an interactive widget made of a cover screen, one or more questions (each with answers), and result screens.

IMPORTANT: Every content type requires at least one result screen (add a default result with is_default=1). Without a result screen, users see nothing after answering the last question.

Pick the type that matches the experience:
- poll: a standalone voting poll that shows live vote percentages and counts.
- set: a multi-question flow. This single API type backs three dashboard presets: "Survey" (a plain set, no result logic), "Calculator"/Product Finder (set with is_calculator=1 — the result is chosen by a math formula over per-answer calculator values), and "Searchable" Product Finder (set with is_searchable=1 — results are matched by search query/filters set on the answers).
- quiz: a right/wrong quiz. Mark one correct answer per question and set attributes_json gives_feedback=1; optionally show a timer (show_timer) and the score (display_results).
- test: a personality/outcome test. Answers carry points (set_answer_result_point) that add up to a matching result screen chosen by its min_c–max_c score range.
- this-that: a single "This or That" question. Only one question is allowed and the answer count should be a power of 2 (2, 4, 8, ...).

After creating, use add_question / add_answer / add_result to build it out. Workflow: create_content → add_question(s) → add_answer(s) → add_result(s) (always add at least one default result) → publish_content.`),
		mcp.WithString("type", mcp.Description("Content type. API values: poll, set, test, quiz, this-that. Note: 'set' backs three dashboard presets — Survey (plain set), Calculator/Product Finder (set + is_calculator), and Searchable Product Finder (set + is_searchable)."), mcp.Required()),
		mcp.WithString("title", mcp.Description("End-user facing title"), mcp.Required()),
		mcp.WithString("desc", mcp.Description("Cover screen description (optional)")),
		mcp.WithString("name", mcp.Description("Internal non-public name (optional)")),
		mcp.WithString("background", mcp.Description("Cover image path returned by upload_image (optional)")),
		mcp.WithString("alt", mcp.Description("Alt text for the cover image")),
		mcp.WithString("vertical_image", mcp.Description("Wide screen layout cover image path")),
		mcp.WithString("vertical_mobile_image", mcp.Description("Main cover image for single-column mobile view")),
		mcp.WithString("embed_footer_url", mcp.Description("URL for the footer image")),
		mcp.WithString("embed_background", mcp.Description("Background image path for the embedded widget frame")),
		mcp.WithString("theme_id", mcp.Description("Theme ID to style the widget with (from list_themes)")),
		mcp.WithNumber("skip_start", mcp.Description("Skip cover card and start from first question: 0 (default) or 1")),
		mcp.WithNumber("skip_result", mcp.Description("Skip result card: 0 (default) or 1")),
		mcp.WithNumber("hide_results", mcp.Description("Hide vote percentages: 0 (default) or 1")),
		mcp.WithNumber("hide_counter", mcp.Description("Hide vote counter: 0 (default) or 1")),
		mcp.WithNumber("display_repeat", mcp.Description("Show play again button: 0 (default) or 1")),
		mcp.WithNumber("is_searchable", mcp.Description("Turns a 'set' into a Searchable Product Finder: results are matched from the answers' search_query/search_filter instead of fixed score ranges. 0 (default) or 1.")),
		mcp.WithNumber("is_calculator", mcp.Description("Turns a 'set' into a Calculator/Product Finder: the result is computed from attributes_json cal_formula over per-answer calculator values. 0 (default) or 1.")),
		mcp.WithNumber("search_results_per_page", mcp.Description("Max number of results shown to the user for searchable content. Range 1-10 (default: 5; recommended 3-5).")),
		mcp.WithNumber("boost_results_min_view", mcp.Description("Searchable result boosting: minimum view count a result needs before its performance affects ranking")),
		mcp.WithNumber("boost_results_ratio", mcp.Description("Searchable result boosting: how strongly result performance affects ranking, 0-100")),
		mcp.WithNumber("result_loading", mcp.Description("Display a loading screen between the last question and the result: 0 (default) or 1")),
		mcp.WithString("loading_next_question_label", mcp.Description("Custom loading label between questions")),
		mcp.WithString("loading_result_label", mcp.Description("Custom loading label between last question and result")),
		mcp.WithNumber("play_once", mcp.Description("Restrict the content to one session per user (tracked by uuid / Poltio session id): 0 (default) or 1")),
		mcp.WithString("play_once_strategy", mcp.Description("When a user counts as having played: 'start' (counted as soon as they begin) or 'result' (counted only once they reach the result screen). Default: result.")),
		mcp.WithString("play_once_msg", mcp.Description("Custom message for play-once error screen")),
		mcp.WithString("play_once_img", mcp.Description("Custom image for play-once error screen")),
		mcp.WithString("play_once_link", mcp.Description("Custom button link for play-once error screen")),
		mcp.WithString("play_once_btn", mcp.Description("Custom button text for play-once error screen")),
		mcp.WithNumber("end_date_day", mcp.Description("Auto-finish content after this many days")),
		mcp.WithNumber("end_date_hour", mcp.Description("Auto-finish content after this many hours")),
		mcp.WithNumber("end_date_minute", mcp.Description("Auto-finish content after this many minutes")),
		mcp.WithString("attributes_json", mcp.Description(`Advanced/behavioral settings as a JSON object. Fields:
- cal_formula (string): formula used to compute the result for calculator contents (is_calculator=1), evaluated over the per-answer/per-question calculator values.
- gives_feedback (0/1): mark answers as correct/wrong and show right-or-wrong feedback after each vote; required for quiz scoring.
- show_timer (0/1): show a running session timer on the widget, for timed competitions/quizzes.
- display_results (0/1, default 1): show the user's correct-answer count on the result screen (requires gives_feedback=1).
- pool_question_count (int): show only this many questions, chosen at random per user, out of all questions added.
- time_limit (int, minutes): countdown timer; the session jumps to the result screen when it reaches 0.
- recom_title (string): heading shown above the results when more than one result is returned, e.g. "Our Recommendations".
- noindex (0/1): ask search engines not to index the hosted widget page directly (does not affect the page you embed it in).
- canonical (string URL): add a canonical rel tag to the widget page.
- redirect (string URL): redirect the widget page to this URL.
- keywords (string): meta keywords for the widget page.`)),
		mcp.WithString("options_json", mcp.Description(`Display options as a JSON object. Fields:
- design (string): widget design version — "" (empty, legacy 2025 design) or "2026-01" (new vertical-image design; enables the options below).
- result_background_blur ("on"/"off"): blur the background image behind result cards (2026 design only).
- share ("off"/""): set "off" to hide the share button (2026 design only).
- list_bullet_points ("true"/""): render result descriptions as bullet points (searchable content, 2026 design only).
- bundle_title (string): heading above bundled/companion results, e.g. "Pairs perfectly with".
- result_button_placement ("top"/"bottom"): where the result CTA button is placed.
- hide_result_session_id (0/1): hide the session ID line on the result screen.`)),
	), tools.CreateContent(c))

	s.AddTool(mcp.NewTool(
		"publish_content",
		mcp.WithDescription("Publish a draft Poltio content item, making it publicly accessible."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
	), tools.PublishContent(c))

	s.AddTool(mcp.NewTool(
		"list_drafts",
		mcp.WithDescription("List unpublished (draft) Poltio content items."),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("per_page", mcp.Description("Results per page (default: 12)")),
		mcp.WithString("type", mcp.Description("Filter by type: poll, set, test, quiz, this-that")),
		mcp.WithString("q", mcp.Description("Search query against title and description")),
		mcp.WithString("order", mcp.Description("Sort field: created_at (default), updated_at, vote_count, voter_count, type, id, end_date")),
		mcp.WithString("sort", mcp.Description("Sort direction: desc (default) or asc")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.ListDrafts(c))

	s.AddTool(mcp.NewTool(
		"update_content",
		mcp.WithDescription("Update an existing Poltio content item's metadata, cover images, and behavioral options. Only the fields you pass are changed. See create_content for the meaning of each content type and option."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithString("title", mcp.Description("New end-user facing title")),
		mcp.WithString("desc", mcp.Description("New cover screen description")),
		mcp.WithString("name", mcp.Description("New internal non-public name")),
		mcp.WithString("type", mcp.Description("Content type. API values: poll, set, test, quiz, this-that. 'set' backs the Survey, Calculator/Product Finder (is_calculator) and Searchable Product Finder (is_searchable) presets.")),
		mcp.WithString("background", mcp.Description("Cover image path returned by upload_image")),
		mcp.WithString("alt", mcp.Description("Alt text for the cover image")),
		mcp.WithString("vertical_image", mcp.Description("Wide screen layout cover image path")),
		mcp.WithString("vertical_mobile_image", mcp.Description("Main cover image for single-column mobile view")),
		mcp.WithString("embed_footer_url", mcp.Description("URL for the footer image")),
		mcp.WithString("embed_background", mcp.Description("Background image path for the embedded widget frame")),
		mcp.WithString("theme_id", mcp.Description("Theme ID to style the widget with (from list_themes)")),
		mcp.WithNumber("skip_start", mcp.Description("Skip cover card: 0 or 1")),
		mcp.WithNumber("skip_result", mcp.Description("Skip result card: 0 or 1")),
		mcp.WithNumber("hide_results", mcp.Description("Hide vote percentages: 0 or 1")),
		mcp.WithNumber("hide_counter", mcp.Description("Hide vote counter: 0 or 1")),
		mcp.WithNumber("display_repeat", mcp.Description("Show play again button: 0 or 1")),
		mcp.WithNumber("is_searchable", mcp.Description("Searchable Product Finder: match results from the answers' search_query/search_filter. 0 or 1.")),
		mcp.WithNumber("is_calculator", mcp.Description("Calculator/Product Finder: compute the result from attributes_json cal_formula over per-answer calculator values. 0 or 1.")),
		mcp.WithNumber("search_results_per_page", mcp.Description("Max results shown for searchable content. Range 1-10 (recommended 3-5).")),
		mcp.WithNumber("boost_results_min_view", mcp.Description("Searchable result boosting: minimum view count a result needs before its performance affects ranking")),
		mcp.WithNumber("boost_results_ratio", mcp.Description("Searchable result boosting: how strongly result performance affects ranking, 0-100")),
		mcp.WithNumber("result_loading", mcp.Description("Display loading screen before result: 0 or 1")),
		mcp.WithString("loading_next_question_label", mcp.Description("Custom loading label between questions")),
		mcp.WithString("loading_result_label", mcp.Description("Custom loading label before result")),
		mcp.WithNumber("play_once", mcp.Description("Restrict to one session per user (tracked by uuid / Poltio session id): 0 or 1")),
		mcp.WithString("play_once_strategy", mcp.Description("When a user counts as played: 'start' (on begin) or 'result' (on reaching the result screen)")),
		mcp.WithString("play_once_msg", mcp.Description("Custom play-once message")),
		mcp.WithString("play_once_img", mcp.Description("Custom play-once image path")),
		mcp.WithString("play_once_link", mcp.Description("Custom play-once button link")),
		mcp.WithString("play_once_btn", mcp.Description("Custom play-once button text")),
		mcp.WithNumber("end_date_day", mcp.Description("Auto-finish after this many days")),
		mcp.WithNumber("end_date_hour", mcp.Description("Auto-finish after this many hours")),
		mcp.WithNumber("end_date_minute", mcp.Description("Auto-finish after this many minutes")),
		mcp.WithString("attributes_json", mcp.Description(`Advanced/behavioral settings as a JSON object. Fields:
- cal_formula (string): formula used to compute the result for calculator contents (is_calculator=1), evaluated over the per-answer/per-question calculator values.
- gives_feedback (0/1): mark answers as correct/wrong and show right-or-wrong feedback after each vote; required for quiz scoring.
- show_timer (0/1): show a running session timer on the widget, for timed competitions/quizzes.
- display_results (0/1, default 1): show the user's correct-answer count on the result screen (requires gives_feedback=1).
- pool_question_count (int): show only this many questions, chosen at random per user, out of all questions added.
- time_limit (int, minutes): countdown timer; the session jumps to the result screen when it reaches 0.
- recom_title (string): heading shown above the results when more than one result is returned, e.g. "Our Recommendations".
- noindex (0/1): ask search engines not to index the hosted widget page directly (does not affect the page you embed it in).
- canonical (string URL): add a canonical rel tag to the widget page.
- redirect (string URL): redirect the widget page to this URL.
- keywords (string): meta keywords for the widget page.`)),
		mcp.WithString("options_json", mcp.Description(`Display options as a JSON object. See create_content options_json for the field list (design, result_background_blur, share, list_bullet_points, bundle_title, result_button_placement, hide_result_session_id).`)),
	), tools.UpdateContent(c))

	s.AddTool(mcp.NewTool(
		"delete_content",
		mcp.WithDescription("Permanently delete a Poltio content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		destructive(),
	), tools.DeleteContent(c))

	s.AddTool(mcp.NewTool(
		"duplicate_content",
		mcp.WithDescription("Duplicate an existing Poltio content item into a new draft."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
	), tools.DuplicateContent(c))

	s.AddTool(mcp.NewTool(
		"get_content_edit",
		mcp.WithDescription("Get full editable content object including all questions, answers, results, and conditions."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetContentEdit(c))

	s.AddTool(mcp.NewTool(
		"list_templates",
		mcp.WithDescription("List available Poltio content templates."),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.ListTemplates(c))

	s.AddTool(mcp.NewTool(
		"get_template",
		mcp.WithDescription("Get a single content template with all its data."),
		mcp.WithString("public_id", mcp.Description("Template public identifier"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetTemplate(c))

	s.AddTool(mcp.NewTool(
		"use_template",
		mcp.WithDescription("Clone a content template into a new draft content item in your account."),
		mcp.WithString("public_id", mcp.Description("Template public identifier"), mcp.Required()),
	), tools.UseTemplate(c))

	s.AddTool(mcp.NewTool(
		"get_content_results",
		mcp.WithDescription("Get paginated vote results for a content item (per-answer vote counts and stats)."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("per_page", mcp.Description("Results per page, 1-100 (default: 12)")),
		mcp.WithString("order_by", mcp.Description("Sort field: position (default), id, click_count, counter")),
		mcp.WithString("order_dir", mcp.Description("Sort direction: desc (default) or asc")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetContentResults(c))

	s.AddTool(mcp.NewTool(
		"get_content_sessions",
		mcp.WithDescription("Get paginated user sessions for a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("per_page", mcp.Description("Results per page (default: 12)")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetContentSessions(c))

	s.AddTool(mcp.NewTool(
		"get_content_metrics",
		mcp.WithDescription("Get time-series metrics for a content item grouped by period."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithString("period", mcp.Description("Grouping period: day, week, month, year"), mcp.Required()),
		mcp.WithString("start", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end", mcp.Description("End date (YYYY-MM-DD)")),
		mcp.WithString("metrics", mcp.Description("Comma-separated metric names (defaults to all): view, vote, voter, start, finish, undo, result_view, result_view_unique, result_click, result_click_unique, result_swipe, result_click_secondary, result_click_compare, result_click_compare_submit, result_click_compare_pdp, conversion, result_click_start_over, result_click_share")),
		mcp.WithString("device_type", mcp.Description("Filter by device: mobile, desktop, tablet, or n/a (unknown). Omit for all devices.")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetContentMetrics(c))

	s.AddTool(mcp.NewTool(
		"get_content_stats",
		mcp.WithDescription("Get aggregate stat totals for a single content item (views, votes, voters, starts, finishes, result clicks, conversion, ...) for a date range. This is the summary shown at the top of the content's stats page; use get_content_metrics for time series."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithString("start", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end", mcp.Description("End date (YYYY-MM-DD)")),
		mcp.WithString("device_type", mcp.Description("Filter by device: mobile, desktop, tablet, or n/a (unknown). Omit for all devices.")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetContentStats(c))

	s.AddTool(mcp.NewTool(
		"get_vote_sources",
		mcp.WithDescription("Get paginated vote sources for a content item: one row per referring URL with its vote count (vote-level granularity; see get_session_urls for session-level counts)."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetVoteSources(c))

	s.AddTool(mcp.NewTool(
		"get_sankey",
		mcp.WithDescription("Get Sankey diagram data showing user flow through a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetSankey(c))

	s.AddTool(mcp.NewTool(
		"get_sankey_users",
		mcp.WithDescription("Get users who took a specific path in the Sankey diagram."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithString("from_id", mcp.Description("Source node ID"), mcp.Required()),
		mcp.WithString("to_id", mcp.Description("Target node ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetSankeyUsers(c))

	s.AddTool(mcp.NewTool(
		"get_searchable_fields",
		mcp.WithDescription("Get all searchable and filterable fields defined for a searchable content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetSearchableFields(c))

	s.AddTool(mcp.NewTool(
		"get_session_urls",
		mcp.WithDescription("Get session URLs grouped by URL with session counts for a content item. This powers the dashboard's 'Vote Sources' report — where sessions came from, one row per URL."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetSessionUrls(c))

	// ── Image Upload ──────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"upload_image",
		mcp.WithDescription("Upload an image to Poltio. Returns a file path to use as the background field in content, questions, answers, or results. Provide the image with EXACTLY ONE of image_path, image_url, or image_base64. PREFER image_path or image_url: the server reads the bytes itself, so the image never passes through the conversation — base64 is slow and fails for larger images because the whole payload has to be generated as the tool argument. Images must not exceed 2 MiB and be one of the supported formats: png, jpg, jpeg, gif, webp. IMPORTANT: when creating images for quiz or test questions, the image must be thematic only — it must NOT contain text or visuals that reveal or hint at the correct answer."),
		mcp.WithString("image_path", mcp.Description("Preferred. Local filesystem path to the image file; the server reads it directly. Use this whenever the image exists on disk where the server runs (e.g. a saved or generated file).")),
		mcp.WithString("image_url", mcp.Description("An http(s) URL to the image; the server fetches it directly. Use this for remotely hosted images instead of downloading and re-encoding them.")),
		mcp.WithString("image_base64", mcp.Description("Fallback for when you only have raw bytes. Base64-encoded image data, either a raw base64 string or a data URI such as data:image/png;base64,.... The decoded image must not exceed 2 MiB. Avoid for large images — the entire payload must be emitted as the tool argument, which is slow and can fail.")),
		mcp.WithString("ext", mcp.Description("Optional file extension without the dot: png, jpg, jpeg, gif, webp. If omitted, it is derived from the image content.")),
	), tools.UploadImage(c))

	// ── Questions ─────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"add_question",
		mcp.WithDescription(`Add a new question to a content item. answer_type determines how the user responds:
- media / text: a normal single- or multiple-choice question. Use 'media' when the answers have images, 'text' for text-only answers (same family). Combine with allow_multiple_answers for multi-select.
- score: a fixed numeric scale (e.g. 1–5) where each option carries a value; used in scored quizzes/tests and calculators.
- star_rating: a star-based rating input.
- yesno: a simple Yes/No question.
- free_text: a free-form text box the user types into (no preset answers; read submissions later with get_question_inputs).
- free_number: a free-form numeric input.
- autocomplete: a typed input that suggests from your predefined answers as the user types; best when you have roughly 15–1000 answers (tune how many suggestions show with recommended_popular_answer).
After adding the question, attach answers with add_answer or add_answers_bulk (free_text/free_number questions take no answers).`),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Question text shown to the user")),
		mcp.WithString("answer_type", mcp.Description("How the user answers: media or text (single/multi choice — media if answers have images), score (numeric scale), star_rating (stars), yesno (Yes/No), free_text (typed text), free_number (typed number), autocomplete (typed input with answer suggestions, for ~15–1000 answers)"), mcp.Required()),
		mcp.WithString("background", mcp.Description("Question image path returned by upload_image. For quiz/test content, the image must be thematic only — it must NOT contain text or visuals that reveal or hint at the correct answer.")),
		mcp.WithString("alt", mcp.Description("Alt text for the question image")),
		mcp.WithString("vertical_image", mcp.Description("Wide screen layout question image path")),
		mcp.WithString("vertical_mobile_image", mcp.Description("Question image for single-column mobile view")),
		mcp.WithString("desc", mcp.Description("Question description shown under the title")),
		mcp.WithNumber("allow_multiple_answers", mcp.Description("Let users pick more than one answer (multi-select / multi-punch): 0 (default) or 1. Pair with max_multi_punch_answer to cap selections.")),
		mcp.WithNumber("is_skippable", mcp.Description("Let users move to the next question without answering: 0 (default) or 1")),
		mcp.WithNumber("rotate_answers", mcp.Description("Shuffle the answer order independently for each user: 0 (default) or 1. Note: passing 1 for quiz and test content ensures each user sees answers in a different order.")),
		mcp.WithNumber("rotate_answers_last", mcp.Description("Shuffle all answers except the last one (keeps e.g. 'None of the above' at the bottom): 0 (default) or 1. Mutually exclusive with rotate_answers.")),
		mcp.WithString("name", mcp.Description("Question name, only for internal use")),
		mcp.WithNumber("max_multi_punch_answer", mcp.Description("Maximum number of answers a user may select. Only meaningful when allow_multiple_answers=1.")),
		mcp.WithNumber("recommended_popular_answer", mcp.Description("For autocomplete questions: how many of the most-voted answers to suggest as the user types.")),
		mcp.WithString("luv", mcp.Description("Lead URL variable: query string appended to the result URL for this question's selection, used to pass data to downstream lead/redirect URLs.")),
		mcp.WithNumber("is_searchable", mcp.Description("Mark this question's votes as usable in search queries/filters for Searchable Product Finders: 0 (default) or 1.")),
		mcp.WithString("cal_val_default", mcp.Description("Calculator contents only: default calculator value applied to every answer in this question that has no value of its own.")),
		mcp.WithString("autocomplete_help", mcp.Description("Custom helper text shown under an autocomplete input.")),
		mcp.WithString("autocomplete_placeholder", mcp.Description("Custom placeholder text for the autocomplete input field.")),
		mcp.WithNumber("position", mcp.Description("Numeric position (order) of this question within the content. Lower shows first.")),
		mcp.WithString("conditions", mcp.Description("Comma-separated Answer IDs (from earlier questions); this question is only shown to users who selected one of them. See condition_reverse to invert.")),
		mcp.WithNumber("condition_reverse", mcp.Description("Invert the display conditions: 0 = show only to users who selected the condition answer(s); 1 = show only to users who did NOT select them.")),
	), tools.AddQuestion(c))

	s.AddTool(mcp.NewTool(
		"update_question",
		mcp.WithDescription("Update an existing question. See add_question for the meaning of each answer_type. Only the fields you pass are changed."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Question text shown to the user")),
		mcp.WithString("answer_type", mcp.Description("How the user answers: media or text (single/multi choice — media if answers have images), score (numeric scale), star_rating (stars), yesno (Yes/No), free_text (typed text), free_number (typed number), autocomplete (typed input with answer suggestions, for ~15–1000 answers)"), mcp.Required()),
		mcp.WithString("background", mcp.Description("Question image path returned by upload_image. For quiz/test content, the image must be thematic only — it must NOT contain text or visuals that reveal or hint at the correct answer.")),
		mcp.WithString("alt", mcp.Description("Alt text for the question image")),
		mcp.WithString("vertical_image", mcp.Description("Wide screen layout question image path")),
		mcp.WithString("vertical_mobile_image", mcp.Description("Question image for single-column mobile view")),
		mcp.WithString("desc", mcp.Description("Question description shown under the title")),
		mcp.WithNumber("allow_multiple_answers", mcp.Description("Let users pick more than one answer (multi-select / multi-punch): 0 or 1. Pair with max_multi_punch_answer to cap selections.")),
		mcp.WithNumber("is_skippable", mcp.Description("Let users move to the next question without answering: 0 or 1")),
		mcp.WithNumber("rotate_answers", mcp.Description("Shuffle the answer order independently for each user: 0 or 1. Always pass 1 for quiz and test content so answer order varies per user.")),
		mcp.WithNumber("rotate_answers_last", mcp.Description("Shuffle all answers except the last one (keeps e.g. 'None of the above' at the bottom): 0 or 1. Mutually exclusive with rotate_answers.")),
		mcp.WithString("name", mcp.Description("Question name, only for internal use")),
		mcp.WithNumber("max_multi_punch_answer", mcp.Description("Maximum number of answers a user may select. Only meaningful when allow_multiple_answers=1.")),
		mcp.WithNumber("recommended_popular_answer", mcp.Description("For autocomplete questions: how many of the most-voted answers to suggest as the user types.")),
		mcp.WithString("luv", mcp.Description("Lead URL variable: query string appended to the result URL for this question's selection, used to pass data to downstream lead/redirect URLs.")),
		mcp.WithNumber("is_searchable", mcp.Description("Mark this question's votes as usable in search queries/filters for Searchable Product Finders: 0 (default) or 1.")),
		mcp.WithString("cal_val_default", mcp.Description("Calculator contents only: default calculator value applied to every answer in this question that has no value of its own.")),
		mcp.WithString("autocomplete_help", mcp.Description("Custom helper text shown under an autocomplete input.")),
		mcp.WithString("autocomplete_placeholder", mcp.Description("Custom placeholder text for the autocomplete input field.")),
		mcp.WithNumber("position", mcp.Description("Numeric position (order) of this question within the content. Lower shows first.")),
		mcp.WithString("conditions", mcp.Description("Comma-separated Answer IDs (from earlier questions); this question is only shown to users who selected one of them. See condition_reverse to invert.")),
		mcp.WithNumber("condition_reverse", mcp.Description("Invert the display conditions: 0 = show only to users who selected the condition answer(s); 1 = show only to users who did NOT select them.")),
	), tools.UpdateQuestion(c))

	s.AddTool(mcp.NewTool(
		"delete_question",
		mcp.WithDescription("Delete a question from a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		destructive(),
	), tools.DeleteQuestion(c))

	// ── Answers ───────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"add_answer",
		mcp.WithDescription("Add a single answer to a question. Use background to set an image answer (upload_image first)."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Answer text (optional when background image is provided)")),
		mcp.WithString("background", mcp.Description("Answer image path returned by upload_image")),
		mcp.WithString("alt", mcp.Description("Alt text for the answer image")),
		mcp.WithString("luv", mcp.Description("Lead URL variable: query string appended to the result URL after a user picks this answer, e.g. '&color=blue'. Carries the selection into downstream lead/redirect URLs.")),
		mcp.WithNumber("has_right_answer", mcp.Description("Enable right/wrong scoring for this answer (works with content attributes_json gives_feedback): 0 (default) or 1")),
		mcp.WithNumber("is_right_answer", mcp.Description("Mark this as the correct answer for a quiz question: 0 (default) or 1")),
		mcp.WithNumber("is_mutually_exclusive", mcp.Description("In a multi-select question, selecting this answer deselects all others (e.g. a 'None of the above' option): 0 (default) or 1")),
		mcp.WithString("cal_val", mcp.Description("Calculator contents only: this answer's calculator value, used by the content's cal_formula. Overrides the question's cal_val_default.")),
		mcp.WithString("search_query", mcp.Description("Searchable Product Finder only: search-index query run to fetch matching results when this answer is selected.")),
		mcp.WithString("search_filter", mcp.Description("Searchable Product Finder only: search-index filter applied when this answer is selected, e.g. 'color: [blue]'.")),
		mcp.WithNumber("position", mcp.Description("Numeric position (order) of this answer within the question. Lower shows first.")),
		mcp.WithNumber("max_vote", mcp.Description("Cap this answer at this many votes; once reached it is disabled and disabled_msg is shown. 0 = unlimited.")),
		mcp.WithString("addon", mcp.Description("Extra metadata attached to this answer; forwarded to GTM events, Pixel events, webhooks and leads when the answer is selected.")),
		mcp.WithString("disabled_msg", mcp.Description("Message shown in place of this answer once it is disabled (e.g. after hitting max_vote).")),
	), tools.AddAnswer(c))

	s.AddTool(mcp.NewTool(
		"add_answers_bulk",
		mcp.WithDescription("Add several text answers to a question at once — the fast way to populate a single/multiple-choice question. Each line becomes its own answer. For images, points, leads or per-answer settings, use add_answer instead."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithString("answers", mcp.Description("Answer texts, one per line. Each non-empty line becomes a separate answer."), mcp.Required()),
		mcp.WithNumber("remove_existing", mcp.Description("Delete the question's existing answers before adding these (replace instead of append): 0 (default) or 1")),
	), tools.AddAnswersBulk(c))

	s.AddTool(mcp.NewTool(
		"update_answer",
		mcp.WithDescription("Update an existing answer."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID"), mcp.Required()),
		mcp.WithString("title", mcp.Description("New answer text")),
		mcp.WithString("background", mcp.Description("Answer image path returned by upload_image")),
		mcp.WithString("alt", mcp.Description("Alt text for the answer image")),
		mcp.WithString("luv", mcp.Description("Lead URL variable: query string appended to the result URL after a user picks this answer, e.g. '&color=blue'. Carries the selection into downstream lead/redirect URLs.")),
		mcp.WithNumber("has_right_answer", mcp.Description("Enable right/wrong scoring for this answer (works with content attributes_json gives_feedback): 0 or 1")),
		mcp.WithNumber("is_right_answer", mcp.Description("Mark this as the correct answer for a quiz question: 0 or 1")),
		mcp.WithNumber("is_mutually_exclusive", mcp.Description("In a multi-select question, selecting this answer deselects all others (e.g. a 'None of the above' option): 0 or 1")),
		mcp.WithString("cal_val", mcp.Description("Calculator contents only: this answer's calculator value, used by the content's cal_formula. Overrides the question's cal_val_default.")),
		mcp.WithString("search_query", mcp.Description("Searchable Product Finder only: search-index query run to fetch matching results when this answer is selected.")),
		mcp.WithString("search_filter", mcp.Description("Searchable Product Finder only: search-index filter applied when this answer is selected, e.g. 'color: [blue]'.")),
		mcp.WithNumber("position", mcp.Description("Numeric position (order) of this answer within the question. Lower shows first.")),
		mcp.WithNumber("max_vote", mcp.Description("Cap this answer at this many votes; once reached it is disabled and disabled_msg is shown. 0 = unlimited.")),
		mcp.WithString("addon", mcp.Description("Extra metadata attached to this answer; forwarded to GTM events, Pixel events, webhooks and leads when the answer is selected.")),
		mcp.WithString("disabled_msg", mcp.Description("Message shown in place of this answer once it is disabled (e.g. after hitting max_vote).")),
	), tools.UpdateAnswer(c))

	s.AddTool(mcp.NewTool(
		"delete_answer",
		mcp.WithDescription("Delete an answer from a question."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID"), mcp.Required()),
		destructive(),
	), tools.DeleteAnswer(c))

	s.AddTool(mcp.NewTool(
		"clone_answers",
		mcp.WithDescription("Copy all answers from a source question to a target question, replacing the target's existing answers."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("source_question_id", mcp.Description("Question to copy answers from"), mcp.Required()),
		mcp.WithNumber("target_question_id", mcp.Description("Question to copy answers to (existing answers will be removed)"), mcp.Required()),
	), tools.CloneAnswers(c))

	s.AddTool(mcp.NewTool(
		"get_answer_order",
		mcp.WithDescription("Get the current answer order (positions) for a question."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetAnswerOrder(c))

	s.AddTool(mcp.NewTool(
		"update_answer_order",
		mcp.WithDescription("Reorder answers in a question. Provide a JSON array of {id, position} objects."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithString("answers", mcp.Description(`JSON array of position objects, e.g. [{"id":1,"position":2},{"id":2,"position":1}]`), mcp.Required()),
	), tools.UpdateAnswerOrder(c))

	// ── Results ───────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"add_result",
		mcp.WithDescription(`Add a result (outcome) screen to a content item. How a result gets selected for a user depends on the content type:
- test / scored quiz: by score range — the user's total points (from set_answer_result_point) fall within this result's min_c–max_c range.
- Calculator/Product Finder (is_calculator): by the value the cal_formula computes.
- Searchable Product Finder (is_searchable): by matching the answers' search_query/search_filter against this result's search / search2 terms.
Add a default result (is_default=1) as a fallback for users no other result matches.`),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Result title shown to the user"), mcp.Required()),
		mcp.WithString("desc", mcp.Description("Result description / body text")),
		mcp.WithString("background", mcp.Description("Result image path returned by upload_image")),
		mcp.WithString("alt", mcp.Description("Alt text for the result image")),
		mcp.WithString("luv", mcp.Description("Lead URL variable: query string appended to the result/redirect URL for this result, e.g. '&result=thanks'.")),
		mcp.WithString("url", mcp.Description("Optional call-to-action URL the result's button links to.")),
		mcp.WithString("url_text", mcp.Description("Label for the call-to-action button (used with url).")),
		mcp.WithString("search", mcp.Description("Searchable Product Finder: primary search terms used to match this result to answer-driven queries.")),
		mcp.WithString("search2", mcp.Description("Searchable Product Finder: secondary search terms for matching this result.")),
		mcp.WithNumber("min_c", mcp.Description("Lowest total score that maps to this result (test/scored content). The result shows when the user's points are between min_c and max_c.")),
		mcp.WithNumber("max_c", mcp.Description("Highest total score that maps to this result (test/scored content).")),
		mcp.WithNumber("is_default", mcp.Description("Make this the catch-all result, shown when no score range or search match applies: 0 (default) or 1.")),
		mcp.WithString("secondary_url", mcp.Description("URL for a secondary action button on the result. Overwritten by the DSC when the content is connected to a data source.")),
		mcp.WithString("secondary_url_text", mcp.Description("Label for the secondary action button (used with secondary_url).")),
		mcp.WithString("source_id", mcp.Description("Product ID for this result, used in GTM ecommerce events, conversion tracking and pixel codes. Overwritten by the DSC when the content is connected to a data source.")),
	), tools.AddResult(c))

	s.AddTool(mcp.NewTool(
		"update_result",
		mcp.WithDescription("Update an existing result screen. See add_result for how results are matched to users. Only the fields you pass are changed."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("result_id", mcp.Description("Result ID"), mcp.Required()),
		mcp.WithString("title", mcp.Description("New result title")),
		mcp.WithString("desc", mcp.Description("New result description / body text")),
		mcp.WithString("background", mcp.Description("Result image path returned by upload_image")),
		mcp.WithString("alt", mcp.Description("Alt text for the result image")),
		mcp.WithString("luv", mcp.Description("Lead URL variable: query string appended to the result/redirect URL for this result, e.g. '&result=thanks'.")),
		mcp.WithString("url", mcp.Description("Call-to-action URL the result's button links to.")),
		mcp.WithString("url_text", mcp.Description("Label for the call-to-action button (used with url).")),
		mcp.WithString("search", mcp.Description("Searchable Product Finder: primary search terms used to match this result to answer-driven queries.")),
		mcp.WithString("search2", mcp.Description("Searchable Product Finder: secondary search terms for matching this result.")),
		mcp.WithNumber("min_c", mcp.Description("Lowest total score that maps to this result (test/scored content). The result shows when the user's points are between min_c and max_c.")),
		mcp.WithNumber("max_c", mcp.Description("Highest total score that maps to this result (test/scored content).")),
		mcp.WithNumber("is_default", mcp.Description("Make this the catch-all result, shown when no score range or search match applies: 0 or 1.")),
		mcp.WithString("secondary_url", mcp.Description("URL for a secondary action button on the result. Overwritten by the DSC when the content is connected to a data source.")),
		mcp.WithString("secondary_url_text", mcp.Description("Label for the secondary action button (used with secondary_url).")),
		mcp.WithString("source_id", mcp.Description("Product ID for this result, used in GTM ecommerce events, conversion tracking and pixel codes. Overwritten by the DSC when the content is connected to a data source.")),
	), tools.UpdateResult(c))

	s.AddTool(mcp.NewTool(
		"delete_result",
		mcp.WithDescription("Delete a result screen from a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("result_id", mcp.Description("Result ID"), mcp.Required()),
		destructive(),
	), tools.DeleteResult(c))

	s.AddTool(mcp.NewTool(
		"set_answer_result_point",
		mcp.WithDescription("Assign how many points an answer contributes toward a specific result. In tests/scored quizzes, points accumulate across the answers a user picks, and the result whose min_c–max_c range contains the total is shown. Call once per (answer, result) pair you want to weight."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID this point value is for"), mcp.Required()),
		mcp.WithNumber("content_result_id", mcp.Description("Result ID the points count toward"), mcp.Required()),
		mcp.WithNumber("point", mcp.Description("Points this answer adds to that result's score (≥ 0)"), mcp.Required()),
	), tools.SetAnswerResultPoint(c))

	// ── Questions — conditions and order ─────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"get_content_conditions",
		mcp.WithDescription("List all questions in a content item that have display conditions attached."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetContentConditions(c))

	s.AddTool(mcp.NewTool(
		"add_question_condition",
		mcp.WithDescription("Add conditional logic / branching: gate a question on an earlier answer. By default the question is shown only to users who selected the given answer ('only people who voted for'); with condition_reverse it is shown only to users who did NOT select it ('only people who did not vote for'). With no conditions a question is shown to everyone. Alternative: update_question's conditions/condition_reverse params set all of a question's conditions in one call (that is how the dashboard editor does it)."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("The question whose visibility is being gated"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("An answer from an earlier question that triggers the condition"), mcp.Required()),
		mcp.WithNumber("condition_reverse", mcp.Description("0 (default) = show the question only to users who selected the answer; 1 = show it only to users who did NOT select the answer.")),
	), tools.AddQuestionCondition(c))

	s.AddTool(mcp.NewTool(
		"remove_question_condition",
		mcp.WithDescription("Remove a single answer from a question's display conditions."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID to remove from conditions"), mcp.Required()),
		destructive(),
	), tools.RemoveQuestionCondition(c))

	s.AddTool(mcp.NewTool(
		"clear_question_conditions",
		mcp.WithDescription("Remove all display conditions from a question."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		destructive(),
	), tools.ClearQuestionConditions(c))

	s.AddTool(mcp.NewTool(
		"get_question_order",
		mcp.WithDescription("Get the current question order (positions) for a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetQuestionOrder(c))

	s.AddTool(mcp.NewTool(
		"update_question_order",
		mcp.WithDescription("Reorder questions in a content item. Provide a JSON array of {id, position} objects."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithString("questions", mcp.Description(`JSON array of position objects, e.g. [{"id":1,"position":2},{"id":2,"position":1}]`), mcp.Required()),
	), tools.UpdateQuestionOrder(c))

	s.AddTool(mcp.NewTool(
		"get_question_inputs",
		mcp.WithDescription("Get paginated free-text answer inputs submitted by voters for a question."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("per_page", mcp.Description("Results per page (default: 12)")),
		mcp.WithString("order", mcp.Description("Sort field: created_at (default), voter_id, id")),
		mcp.WithString("sort", mcp.Description("Sort direction: desc (default) or asc")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetQuestionInputs(c))

	// ── Lead attachment ───────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"set_content_lead",
		mcp.WithDescription("Attach a lead form to the cover screen of a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("lead_id", mcp.Description("Lead ID to attach"), mcp.Required()),
	), tools.SetContentLead(c))

	s.AddTool(mcp.NewTool(
		"remove_content_lead",
		mcp.WithDescription("Remove the lead form from the cover screen of a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		destructive(),
	), tools.RemoveContentLead(c))

	s.AddTool(mcp.NewTool(
		"set_question_lead",
		mcp.WithDescription("Attach a lead form to a question (fires when the question is answered, regardless of which answer is picked)."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("lead_id", mcp.Description("Lead ID to attach"), mcp.Required()),
	), tools.SetQuestionLead(c))

	s.AddTool(mcp.NewTool(
		"remove_question_lead",
		mcp.WithDescription("Remove the lead form from a question."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		destructive(),
	), tools.RemoveQuestionLead(c))

	s.AddTool(mcp.NewTool(
		"set_answer_lead",
		mcp.WithDescription("Attach a lead form to a specific answer."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID"), mcp.Required()),
		mcp.WithNumber("lead_id", mcp.Description("Lead ID to attach"), mcp.Required()),
	), tools.SetAnswerLead(c))

	s.AddTool(mcp.NewTool(
		"remove_answer_lead",
		mcp.WithDescription("Remove the lead form from a specific answer."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID"), mcp.Required()),
		destructive(),
	), tools.RemoveAnswerLead(c))

	s.AddTool(mcp.NewTool(
		"set_result_lead",
		mcp.WithDescription("Attach a lead form to a result screen."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("result_id", mcp.Description("Result ID"), mcp.Required()),
		mcp.WithNumber("lead_id", mcp.Description("Lead ID to attach"), mcp.Required()),
	), tools.SetResultLead(c))

	s.AddTool(mcp.NewTool(
		"remove_result_lead",
		mcp.WithDescription("Remove the lead form from a result screen."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("result_id", mcp.Description("Result ID"), mcp.Required()),
		destructive(),
	), tools.RemoveResultLead(c))

	// ── Lead management ───────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_leads",
		mcp.WithDescription("List lead campaigns for this organization."),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.ListLeads(c))

	s.AddTool(mcp.NewTool(
		"create_lead",
		mcp.WithDescription("Create a new lead campaign."),
		mcp.WithString("name", mcp.Description("Human-readable name for the lead"), mcp.Required()),
		mcp.WithString("type", mcp.Description("Lead type: 'input' (collect form fields from the user), 'redirect' (send the user to a URL via a CTA button), 'empty' (display-only: show an image and/or YouTube video with an optional button), 'internal_redirect' (legacy: jump to another Poltio content, redirect_url holds the target Content ID)"), mcp.Required()),
		mcp.WithString("msg", mcp.Description("Message displayed to the user")),
		mcp.WithString("fields", mcp.Description("Comma-separated input field names: gsm, email, name, comment")),
		mcp.WithString("title", mcp.Description("Window title")),
		mcp.WithString("button_value", mcp.Description("CTA button label")),
		mcp.WithString("redirect_url", mcp.Description("Redirect URL for redirect-type leads. Special values: '#next' advances to the next question instead of leaving; 'https://poltio.com/widget/{public_id}' opens another Poltio content; for internal_redirect leads this holds the target Content ID.")),
		mcp.WithString("ios_link", mcp.Description("Redirect URL override for users on iOS devices")),
		mcp.WithString("android_link", mcp.Description("Redirect URL override for users on Android devices")),
		mcp.WithString("youtube_id", mcp.Description("YouTube video ID to display inside the lead (typically used with the 'empty' display type)")),
		mcp.WithString("terms_conditions", mcp.Description("Terms and conditions text (markdown)")),
		mcp.WithString("terms_conditions2", mcp.Description("Second terms and conditions text (markdown), shown as a separate consent block")),
		mcp.WithString("image", mcp.Description("Image path for the lead")),
		mcp.WithNumber("is_active", mcp.Description("Active state: 0 or 1")),
		mcp.WithNumber("mandatory", mcp.Description("Non-dismissable lead: 0 or 1")),
		mcp.WithNumber("tc_optional", mcp.Description("Terms and conditions checkbox optional: 0 or 1 (default: 1)")),
		mcp.WithNumber("tc2_optional", mcp.Description("Second terms and conditions checkbox optional: 0 or 1 (default: 1)")),
		mcp.WithNumber("auto_open", mcp.Description("Auto-redirect to URL: 0 (default) or 1")),
		mcp.WithNumber("auto_open_delay", mcp.Description("Auto-redirect delay in milliseconds (default: 2500)")),
		mcp.WithNumber("open_minimized", mcp.Description("Open lead in minimized state by default: 0 (default) or 1. Cannot be combined with auto_open.")),
		mcp.WithNumber("delay", mcp.Description("Delay in milliseconds before loading the lead")),
		mcp.WithNumber("stop_set", mcp.Description("Stop the set after this lead's input: 0 (default) or 1. Only works on 'set' contents.")),
		mcp.WithNumber("dont_shorten", mcp.Description("Don't route the redirect URL through the Poltio short-link/click-tracking service: 0 (default) or 1")),
		mcp.WithString("link_target", mcp.Description("Link open target: blank, parent, self, or top")),
		mcp.WithString("tc_short", mcp.Description("Short text for the terms and conditions line")),
		mcp.WithString("tc2_short", mcp.Description("Short text for the second terms and conditions line")),
		mcp.WithString("tc_approve_button_label", mcp.Description("Custom button label for the accept section")),
		mcp.WithString("tc_reject_button_label", mcp.Description("Custom button label for the reject section")),
		mcp.WithString("tc2_approve_button_label", mcp.Description("Custom button label for the second accept section")),
		mcp.WithString("tc2_reject_button_label", mcp.Description("Custom button label for the second reject section")),
		mcp.WithString("custom_labels_json", mcp.Description(`Custom input field labels as JSON, e.g. {"email":"E-mail","gsm":"Phone"}`)),
	), tools.CreateLead(c))

	s.AddTool(mcp.NewTool(
		"get_lead",
		mcp.WithDescription("Get a single lead campaign by ID."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetLead(c))

	s.AddTool(mcp.NewTool(
		"update_lead",
		mcp.WithDescription("Update a lead campaign."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Human-readable name")),
		mcp.WithString("type", mcp.Description("Lead type: 'input' (collect form fields), 'redirect' (CTA button to a URL), 'empty' (display-only image/video), 'internal_redirect' (legacy: jump to another Poltio content)")),
		mcp.WithString("msg", mcp.Description("Message displayed to the user")),
		mcp.WithString("fields", mcp.Description("Comma-separated input field names: gsm, email, name, comment")),
		mcp.WithString("title", mcp.Description("Window title")),
		mcp.WithString("button_value", mcp.Description("CTA button label")),
		mcp.WithString("redirect_url", mcp.Description("Redirect URL. Special values: '#next' advances to the next question; 'https://poltio.com/widget/{public_id}' opens another Poltio content; for internal_redirect leads this holds the target Content ID.")),
		mcp.WithString("ios_link", mcp.Description("Redirect URL override for users on iOS devices")),
		mcp.WithString("android_link", mcp.Description("Redirect URL override for users on Android devices")),
		mcp.WithString("youtube_id", mcp.Description("YouTube video ID to display inside the lead (typically used with the 'empty' display type)")),
		mcp.WithString("terms_conditions", mcp.Description("Terms and conditions text (markdown)")),
		mcp.WithString("terms_conditions2", mcp.Description("Second terms and conditions text (markdown), shown as a separate consent block")),
		mcp.WithString("image", mcp.Description("Image path for the lead")),
		mcp.WithNumber("is_active", mcp.Description("Active state: 0 or 1")),
		mcp.WithNumber("mandatory", mcp.Description("Non-dismissable: 0 or 1")),
		mcp.WithNumber("tc_optional", mcp.Description("Terms and conditions checkbox optional: 0 or 1")),
		mcp.WithNumber("tc2_optional", mcp.Description("Second terms and conditions checkbox optional: 0 or 1")),
		mcp.WithNumber("auto_open", mcp.Description("Auto-redirect to URL: 0 or 1")),
		mcp.WithNumber("auto_open_delay", mcp.Description("Auto-redirect delay in milliseconds")),
		mcp.WithNumber("open_minimized", mcp.Description("Open lead in minimized state by default: 0 or 1. Cannot be combined with auto_open.")),
		mcp.WithNumber("delay", mcp.Description("Delay in milliseconds before loading the lead")),
		mcp.WithNumber("stop_set", mcp.Description("Stop the set after this lead's input: 0 or 1. Only works on 'set' contents.")),
		mcp.WithNumber("dont_shorten", mcp.Description("Don't route the redirect URL through the Poltio short-link/click-tracking service: 0 or 1")),
		mcp.WithString("link_target", mcp.Description("Link open target: blank, parent, self, or top")),
		mcp.WithString("tc_short", mcp.Description("Short text for the terms and conditions line")),
		mcp.WithString("tc2_short", mcp.Description("Short text for the second terms and conditions line")),
		mcp.WithString("tc_approve_button_label", mcp.Description("Custom button label for the accept section")),
		mcp.WithString("tc_reject_button_label", mcp.Description("Custom button label for the reject section")),
		mcp.WithString("tc2_approve_button_label", mcp.Description("Custom button label for the second accept section")),
		mcp.WithString("tc2_reject_button_label", mcp.Description("Custom button label for the second reject section")),
		mcp.WithString("custom_labels_json", mcp.Description(`Custom input field labels as JSON, e.g. {"email":"E-mail","gsm":"Phone"}`)),
	), tools.UpdateLead(c))

	s.AddTool(mcp.NewTool(
		"delete_lead",
		mcp.WithDescription("Delete a lead campaign."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		destructive(),
	), tools.DeleteLead(c))

	s.AddTool(mcp.NewTool(
		"get_lead_inputs",
		mcp.WithDescription("Get paginated user submissions collected through a lead form. Each row holds the submitted field values (name, email, phone/gsm, comment) plus session context such as play time, correct-answer count, and calculator score."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetLeadInputs(c))

	s.AddTool(mcp.NewTool(
		"get_lead_logs",
		mcp.WithDescription("Get paginated activation logs for a lead campaign — when and on which content the lead fired (time, lead id, content type, content id)."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetLeadLogs(c))

	s.AddTool(mcp.NewTool(
		"get_lead_codes",
		mcp.WithDescription("Get paginated coupon codes for a lead campaign. Each code includes redemption state: is_used, single_use, session_id, and used_at."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetLeadCodes(c))

	s.AddTool(mcp.NewTool(
		"add_lead_codes",
		mcp.WithDescription("Add one or more coupon codes to a lead campaign (one code per line)."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		mcp.WithString("codes", mcp.Description("Coupon codes, one per line"), mcp.Required()),
		mcp.WithNumber("single_use", mcp.Description("Each code can only be used once: 0 (default) or 1")),
		mcp.WithNumber("remove_existing", mcp.Description("Remove existing codes first: 0 (default) or 1")),
	), tools.AddLeadCodes(c))

	s.AddTool(mcp.NewTool(
		"delete_all_lead_codes",
		mcp.WithDescription("Remove ALL coupon codes from a lead campaign."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		destructive(),
	), tools.DeleteAllLeadCodes(c))

	s.AddTool(mcp.NewTool(
		"update_lead_code",
		mcp.WithDescription("Update a single coupon code in a lead campaign."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		mcp.WithString("lead_coupon_code_id", mcp.Description("Coupon code ID"), mcp.Required()),
		mcp.WithString("code", mcp.Description("New code value"), mcp.Required()),
		mcp.WithNumber("single_use", mcp.Description("Single-use flag: 0 or 1")),
	), tools.UpdateLeadCode(c))

	s.AddTool(mcp.NewTool(
		"delete_lead_code",
		mcp.WithDescription("Delete a single coupon code from a lead campaign."),
		mcp.WithString("lead_id", mcp.Description("Lead ID"), mcp.Required()),
		mcp.WithString("lead_coupon_code_id", mcp.Description("Coupon code ID"), mcp.Required()),
		destructive(),
	), tools.DeleteLeadCode(c))

	// ── Pixel codes ───────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_pixel_codes",
		mcp.WithDescription("List pixel code snippets for this organization."),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.ListPixelCodes(c))

	s.AddTool(mcp.NewTool(
		"create_pixel_code",
		mcp.WithDescription("Create a reusable pixel/tracking snippet. After creating it, attach it with the set_*_pixel_code tools to a content's cover, a question, an answer, or a result (view or click) so it fires at that exact point in the user's session — used for conversion tracking, retargeting, and analytics (Meta/Google/GTM etc.)."),
		mcp.WithString("name", mcp.Description("Human-readable name"), mcp.Required()),
		mcp.WithString("code", mcp.Description(`HTML snippet (img, iframe, or script tag) fired when this pixel is triggered. You may embed dynamic placeholder tokens that Poltio substitutes at fire time:
[parent_page_url] (URL of the page embedding the widget, escaped), [content_id], [content_title], [q_title], [q_number], [q_id], [a_title], [a_number], [a_id], [r_title], [r_number], [r_id], [r_source_id] (the product's ID from your data-source feed), [voter_id], [puid] (your own UUID passed on the widget URL), [session_id].
Example: <img src="https://t.example.com/e?contentId=[content_id]&answerId=[a_id]"/>.`), mcp.Required()),
	), tools.CreatePixelCode(c))

	s.AddTool(mcp.NewTool(
		"update_pixel_code",
		mcp.WithDescription("Update an existing pixel code snippet."),
		mcp.WithNumber("pixel_code_id", mcp.Description("Pixel code ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Human-readable name")),
		mcp.WithString("code", mcp.Description(`HTML snippet (img, iframe, or script tag) fired when this pixel is triggered. You may embed dynamic placeholder tokens that Poltio substitutes at fire time:
[parent_page_url] (URL of the page embedding the widget, escaped), [content_id], [content_title], [q_title], [q_number], [q_id], [a_title], [a_number], [a_id], [r_title], [r_number], [r_id], [r_source_id] (the product's ID from your data-source feed), [voter_id], [puid] (your own UUID passed on the widget URL), [session_id].
Example: <img src="https://t.example.com/e?contentId=[content_id]&answerId=[a_id]"/>.`)),
	), tools.UpdatePixelCode(c))

	s.AddTool(mcp.NewTool(
		"delete_pixel_code",
		mcp.WithDescription("Delete a pixel code snippet."),
		mcp.WithNumber("pixel_code_id", mcp.Description("Pixel code ID"), mcp.Required()),
		destructive(),
	), tools.DeletePixelCode(c))

	s.AddTool(mcp.NewTool(
		"set_content_pixel_code",
		mcp.WithDescription("Attach a pixel code to the cover screen of a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("pixel_code_id", mcp.Description("Pixel code ID"), mcp.Required()),
	), tools.SetContentPixelCode(c))

	s.AddTool(mcp.NewTool(
		"remove_content_pixel_code",
		mcp.WithDescription("Remove the pixel code from the cover screen of a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		destructive(),
	), tools.RemoveContentPixelCode(c))

	s.AddTool(mcp.NewTool(
		"set_question_pixel_code",
		mcp.WithDescription("Attach a pixel code to all answers of a question."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("pixel_code_id", mcp.Description("Pixel code ID"), mcp.Required()),
	), tools.SetQuestionPixelCode(c))

	s.AddTool(mcp.NewTool(
		"remove_question_pixel_code",
		mcp.WithDescription("Remove the pixel code from all answers of a question."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		destructive(),
	), tools.RemoveQuestionPixelCode(c))

	s.AddTool(mcp.NewTool(
		"set_answer_pixel_code",
		mcp.WithDescription("Attach a pixel code to a specific answer."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID"), mcp.Required()),
		mcp.WithNumber("pixel_code_id", mcp.Description("Pixel code ID"), mcp.Required()),
	), tools.SetAnswerPixelCode(c))

	s.AddTool(mcp.NewTool(
		"remove_answer_pixel_code",
		mcp.WithDescription("Remove the pixel code from a specific answer."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("question_id", mcp.Description("Question ID"), mcp.Required()),
		mcp.WithNumber("answer_id", mcp.Description("Answer ID"), mcp.Required()),
		destructive(),
	), tools.RemoveAnswerPixelCode(c))

	s.AddTool(mcp.NewTool(
		"set_result_pixel_code",
		mcp.WithDescription("Attach a pixel code to a result screen (fires on result view)."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("result_id", mcp.Description("Result ID"), mcp.Required()),
		mcp.WithNumber("pixel_code_id", mcp.Description("Pixel code ID"), mcp.Required()),
	), tools.SetResultPixelCode(c))

	s.AddTool(mcp.NewTool(
		"remove_result_pixel_code",
		mcp.WithDescription("Remove the view pixel code from a result screen."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("result_id", mcp.Description("Result ID"), mcp.Required()),
		destructive(),
	), tools.RemoveResultPixelCode(c))

	s.AddTool(mcp.NewTool(
		"set_result_click_pixel_code",
		mcp.WithDescription("Attach a pixel code to a result screen's click/CTA action."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("result_id", mcp.Description("Result ID"), mcp.Required()),
		mcp.WithNumber("pixel_code_id", mcp.Description("Pixel code ID"), mcp.Required()),
	), tools.SetResultClickPixelCode(c))

	s.AddTool(mcp.NewTool(
		"remove_result_click_pixel_code",
		mcp.WithDescription("Remove the click pixel code from a result screen."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("result_id", mcp.Description("Result ID"), mcp.Required()),
		destructive(),
	), tools.RemoveResultClickPixelCode(c))

	// ── Themes ────────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_themes",
		mcp.WithDescription("List themes for this organization."),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.ListThemes(c))

	s.AddTool(mcp.NewTool(
		"get_default_theme",
		mcp.WithDescription("Get the default theme values to use as a base when creating a new theme."),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetDefaultTheme(c))

	s.AddTool(mcp.NewTool(
		"get_theme",
		mcp.WithDescription("Get a single theme by ID."),
		mcp.WithNumber("theme_id", mcp.Description("Theme ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetTheme(c))

	s.AddTool(mcp.NewTool(
		"create_theme",
		mcp.WithDescription("Create a new theme. Call get_default_theme first to discover available fields, then pass overrides as fields_json."),
		mcp.WithString("name", mcp.Description("Internal name for the theme"), mcp.Required()),
		mcp.WithString("fields_json", mcp.Description("JSON object of theme fields to set (colors, fonts, radii, per-section styles like cover_*, question_*, result_*, gtm_id). Color values must be RGB like 'rgb(255, 0, 0)' or 'r,g,b' as returned by get_default_theme — not hex.")),
	), tools.CreateTheme(c))

	s.AddTool(mcp.NewTool(
		"update_theme",
		mcp.WithDescription("Update an existing theme's fields."),
		mcp.WithNumber("theme_id", mcp.Description("Theme ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("New internal name for the theme")),
		mcp.WithString("fields_json", mcp.Description("JSON object of theme fields to update. Color values must be RGB (as returned by get_theme), not hex."), mcp.Required()),
	), tools.UpdateTheme(c))

	s.AddTool(mcp.NewTool(
		"find_theme",
		mcp.WithDescription("Auto-extract a theme (colors, fonts) from an existing web page URL — useful to match the widget's look to a customer's site. Returns suggested theme fields; save them with create_theme."),
		mcp.WithString("url", mcp.Description("Web page URL to extract theme styles from"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.FindTheme(c))

	s.AddTool(mcp.NewTool(
		"delete_theme",
		mcp.WithDescription("Delete a theme (fails if the theme is currently in use)."),
		mcp.WithNumber("theme_id", mcp.Description("Theme ID"), mcp.Required()),
		destructive(),
	), tools.DeleteTheme(c))

	// ── Dashboard ─────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"get_dashboard",
		mcp.WithDescription("Get account dashboard data including recent content, profile, and aggregate counters."),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetDashboard(c))

	s.AddTool(mcp.NewTool(
		"get_dashboard_summary",
		mcp.WithDescription("Get most recently active content stat summary for the dashboard."),
		mcp.WithString("start", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end", mcp.Description("End date (YYYY-MM-DD)")),
		mcp.WithNumber("take", mcp.Description("Number of items to return")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetDashboardSummary(c))

	s.AddTool(mcp.NewTool(
		"get_dashboard_metrics",
		mcp.WithDescription("Get account-wide time-series metrics grouped by period."),
		mcp.WithString("period", mcp.Description("Grouping period: day, week, month, year"), mcp.Required()),
		mcp.WithString("start", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end", mcp.Description("End date (YYYY-MM-DD)")),
		mcp.WithString("metrics", mcp.Description("Comma-separated metric names (defaults to all): view, vote, voter, start, finish, undo, result_view, result_view_unique, result_click, result_click_unique, result_swipe, result_click_secondary, result_click_compare, result_click_compare_submit, result_click_compare_pdp, conversion, result_click_start_over, result_click_share")),
		mcp.WithString("device_type", mcp.Description("Filter by device: mobile, desktop, tablet, or n/a (unknown). Omit for all devices.")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetDashboardMetrics(c))

	s.AddTool(mcp.NewTool(
		"get_dashboard_stats",
		mcp.WithDescription("Get account-wide aggregate stat totals (views, votes, voters, starts, finishes, result clicks, conversion, ...) for a date range. This is the summary the dashboard's stat cards show; use get_dashboard_metrics for time series."),
		mcp.WithString("start", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end", mcp.Description("End date (YYYY-MM-DD)")),
		mcp.WithString("device_type", mcp.Description("Filter by device: mobile, desktop, tablet, or n/a (unknown). Omit for all devices.")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetDashboardStats(c))

	// ── Sheet Hooks ───────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_sheet_hooks",
		mcp.WithDescription("List Google Sheet hooks for this organization."),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithString("public_id", mcp.Description("Filter by content public_id")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.ListSheetHooks(c))

	s.AddTool(mcp.NewTool(
		"create_sheet_hook",
		mcp.WithDescription("Create a new Google Sheet hook to stream votes into a sheet in real time."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithString("sheet_id", mcp.Description("Google Sheet ID (from the sheet URL)"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Internal name for the hook")),
		mcp.WithNumber("is_active", mcp.Description("Active state: 0 or 1 (default: 1)")),
	), tools.CreateSheetHook(c))

	s.AddTool(mcp.NewTool(
		"get_sheet_hook",
		mcp.WithDescription("Get details of a Google Sheet hook."),
		mcp.WithNumber("hook_id", mcp.Description("Hook ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetSheetHook(c))

	s.AddTool(mcp.NewTool(
		"update_sheet_hook",
		mcp.WithDescription("Update an existing Google Sheet hook."),
		mcp.WithNumber("hook_id", mcp.Description("Hook ID"), mcp.Required()),
		mcp.WithString("sheet_id", mcp.Description("New Google Sheet ID")),
		mcp.WithString("name", mcp.Description("New internal name")),
		mcp.WithString("public_id", mcp.Description("Content public identifier")),
		mcp.WithNumber("is_active", mcp.Description("Active state: 0 or 1")),
	), tools.UpdateSheetHook(c))

	s.AddTool(mcp.NewTool(
		"delete_sheet_hook",
		mcp.WithDescription("Delete a Google Sheet hook."),
		mcp.WithNumber("hook_id", mcp.Description("Hook ID"), mcp.Required()),
		destructive(),
	), tools.DeleteSheetHook(c))

	s.AddTool(mcp.NewTool(
		"get_sheet_hook_logs",
		mcp.WithDescription("Get execution logs for a Google Sheet hook."),
		mcp.WithNumber("hook_id", mcp.Description("Hook ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetSheetHookLogs(c))

	// ── Webhooks ──────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_webhooks",
		mcp.WithDescription("List webhooks for this organization."),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithString("public_id", mcp.Description("Filter by content public_id")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.ListWebhooks(c))

	s.AddTool(mcp.NewTool(
		"create_webhook",
		mcp.WithDescription("Create a webhook to receive vote/session data in real time."),
		mcp.WithString("url", mcp.Description("Webhook endpoint URL"), mcp.Required()),
		mcp.WithString("public_id", mcp.Description("Content public_id (omit for account-wide hooks)")),
		mcp.WithString("name", mcp.Description("Internal name")),
		mcp.WithNumber("is_active", mcp.Description("Active state: 0 or 1 (default: 1)")),
		mcp.WithNumber("delay", mcp.Description("Delay in seconds before firing (dashboard default: 10)")),
		mcp.WithNumber("send_leads", mcp.Description("Include lead data: 0 or 1 (dashboard default: 1)")),
		mcp.WithNumber("send_answers", mcp.Description("Include answer data: 0 or 1 (dashboard default: 1)")),
		mcp.WithNumber("account_wide", mcp.Description("Fire for all content in account: 0 (default) or 1. At most 2 account-wide webhooks are allowed.")),
		mcp.WithNumber("incomplete_send", mcp.Description("Fire for incomplete sessions: 0 (default) or 1")),
		mcp.WithNumber("incomplete_delay", mcp.Description("Seconds from session start to trigger incomplete webhook. Must be at least 60 and not less than delay (dashboard default: 60).")),
		mcp.WithNumber("use_oauth", mcp.Description("Enable OAuth authentication for webhook: 0 or 1")),
		mcp.WithString("oauth_login_endpoint", mcp.Description("OAuth token API endpoint (required with use_oauth)")),
		mcp.WithString("oauth_request_body_json", mcp.Description("Additional OAuth request body fields as JSON")),
		mcp.WithString("oauth_request_headers_json", mcp.Description("Additional OAuth request headers as JSON")),
	), tools.CreateWebhook(c))

	s.AddTool(mcp.NewTool(
		"get_webhook",
		mcp.WithDescription("Get details of a webhook."),
		mcp.WithNumber("hook_id", mcp.Description("Hook ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetWebhook(c))

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
	), tools.UpdateWebhook(c))

	s.AddTool(mcp.NewTool(
		"delete_webhook",
		mcp.WithDescription("Delete a webhook."),
		mcp.WithNumber("hook_id", mcp.Description("Hook ID"), mcp.Required()),
		destructive(),
	), tools.DeleteWebhook(c))

	s.AddTool(mcp.NewTool(
		"get_webhook_logs",
		mcp.WithDescription("Get execution logs for a webhook."),
		mcp.WithNumber("hook_id", mcp.Description("Hook ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetWebhookLogs(c))

	// ── Vote / Stats ──────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"get_voters",
		mcp.WithDescription("Get paginated list of voters for a content item."),
		mcp.WithString("public_id", mcp.Description("Content public identifier"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("per_page", mcp.Description("Results per page (default: 12)")),
		mcp.WithNumber("download", mcp.Description("Pass 1 to request the report as a downloadable file instead of inline data. Omit for normal paginated results.")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetVoters(c))

	s.AddTool(mcp.NewTool(
		"get_conversion_time_stats",
		mcp.WithDescription("Get conversion time-series stats for the whole account or a specific content item."),
		mcp.WithNumber("content_id", mcp.Description("Filter to a specific content item by its integer ID (optional)")),
		mcp.WithString("start", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end", mcp.Description("End date (YYYY-MM-DD)")),
		mcp.WithString("device_type", mcp.Description("Filter by device: mobile, desktop, tablet, or n/a (unknown). Omit for all devices.")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetConversionTimeStats(c))

	// ── Reports ───────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_reports",
		mcp.WithDescription("List downloadable report requests."),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.ListReports(c))

	s.AddTool(mcp.NewTool(
		"create_report",
		mcp.WithDescription("Request a new downloadable report (sent to your account email)."),
		mcp.WithString("report", mcp.Description("Report type: content-sessions or content-voters (pass public_id), voter-leads (pass base_id = the lead ID), answer-voters (pass target_ids = answer IDs)"), mcp.Required()),
		mcp.WithString("public_id", mcp.Description("Content public_id for content-scoped reports (content-sessions, content-voters)")),
		mcp.WithNumber("base_id", mcp.Description("Base entity ID, e.g. the lead ID for voter-leads reports")),
		mcp.WithString("target_ids", mcp.Description("Comma-separated answer IDs for answer-voters reports")),
	), tools.CreateReport(c))

	// ── Data Sources ──────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_data_sources",
		mcp.WithDescription("List the product/catalog data sources connected to this account, with each one's pipeline status (e.g. waiting for approval, in review, processing, up to date, or failed). Data sources are product feeds (e.g. a Shopify catalog) that power Searchable Product Finder content — their items become the results users are matched to."),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.ListDataSources(c))

	s.AddTool(mcp.NewTool(
		"create_data_source",
		mcp.WithDescription("Submit a product/catalog feed URL (e.g. a Shopify XML or JSON feed) as a data source. After creation, configure it with set_data_source_elements and start the import with publish_data_source; once imported, its items can be served as results in Searchable Product Finder content. For a CSV file use create_csv_data_source instead. For an XML feed prefer create_xml_data_source: type xml here currently imports 0 items because the item path cannot be configured via the API."),
		mcp.WithString("name", mcp.Description("Human-readable name"), mcp.Required()),
		mcp.WithString("source", mcp.Description("Fully qualified feed URL"), mcp.Required()),
		mcp.WithString("type", mcp.Description("Feed format: xml or json"), mcp.Required()),
		mcp.WithString("notes", mcp.Description("Optional notes for the review team")),
	), tools.CreateDataSource(c))

	s.AddTool(mcp.NewTool(
		"create_csv_data_source",
		mcp.WithDescription("Create a data source from a CSV file in one step (multipart upload, as the dashboard does). The record is created as type 'csv' so its columns can be inspected with get_data_source_attributes and mapped with set_data_source_elements, then imported with publish_data_source."),
		mcp.WithString("name", mcp.Description("Human-readable name for the data source"), mcp.Required()),
		mcp.WithString("file_base64", mcp.Description("Base64-encoded CSV content. The decoded file must not exceed 2 MiB."), mcp.Required()),
		mcp.WithString("filename", mcp.Description("Filename, defaults to data.csv")),
	), tools.CreateCSVDataSource(c))

	s.AddTool(mcp.NewTool(
		"create_xml_data_source",
		mcp.WithDescription("Create a data source from a remote XML feed by fetching it, flattening each item node one level deep and importing it through the CSV pipeline. Use this instead of create_data_source with type xml (which currently imports 0 items because the item path cannot be configured via the API). The import is a snapshot: it does not auto-sync with the feed; to refresh, delete and recreate. After creation, map columns with set_data_source_elements and run publish_data_source."),
		mcp.WithString("name", mcp.Description("Human-readable name for the data source"), mcp.Required()),
		mcp.WithString("feed_url", mcp.Description("Fully qualified URL of the XML feed"), mcp.Required()),
		mcp.WithString("items_path", mcp.Description("Name of the repeating item node, e.g. 'item' for RSS/Google Shopping feeds or 'product' for custom feeds"), mcp.Required()),
	), tools.CreateXMLDataSource(c))

	s.AddTool(mcp.NewTool(
		"get_data_source",
		mcp.WithDescription("Get a single data source with its status and configured element mappings (data_source_item_elements)."),
		mcp.WithNumber("data_source_id", mcp.Description("Data source ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetDataSource(c))

	s.AddTool(mcp.NewTool(
		"get_data_source_attributes",
		mcp.WithDescription("Discover the columns/fields found in an uploaded or submitted data source feed, with example values per column. Use this before set_data_source_elements to see what can be mapped."),
		mcp.WithNumber("data_source_id", mcp.Description("Data source ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetDataSourceAttributes(c))

	s.AddTool(mcp.NewTool(
		"set_data_source_elements",
		mcp.WithDescription("Map a data source's feed columns to Poltio element types — the configuration step required before publish_data_source. The 'id', 'name', 'url' and 'image' types are mandatory for publishing; unmapped extra columns can be included as 'generic' to keep them as attributes."),
		mcp.WithNumber("data_source_id", mcp.Description("Data source ID"), mcp.Required()),
		mcp.WithString("elements_json", mcp.Description("JSON array of {\"element\": \"<column name from get_data_source_attributes>\", \"type\": \"<element type>\"}. Types: generic, id, gtin, name, condition, description, price, sale_price, image, url, brand, product_type. Example: [{\"element\":\"id\",\"type\":\"id\"},{\"element\":\"title\",\"type\":\"name\"},{\"element\":\"url\",\"type\":\"url\"},{\"element\":\"image\",\"type\":\"image\"},{\"element\":\"price\",\"type\":\"price\"}]"), mcp.Required()),
	), tools.SetDataSourceElements(c))

	s.AddTool(mcp.NewTool(
		"publish_data_source",
		mcp.WithDescription("Publish a configured data source so its import pipeline starts and its items become available as Searchable Product Finder results. Requires element mappings set first via set_data_source_elements — publishing without an 'id'-typed element is rejected (\"You can not publish a data source without id element\"). Check progress with list_data_sources or get_data_source_items."),
		mcp.WithNumber("data_source_id", mcp.Description("Data source ID (from list_data_sources or create_data_source)"), mcp.Required()),
	), tools.PublishDataSource(c))

	s.AddTool(mcp.NewTool(
		"get_data_source_items",
		mcp.WithDescription("Get paginated imported items of a published data source, to verify the import worked."),
		mcp.WithNumber("data_source_id", mcp.Description("Data source ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetDataSourceItems(c))

	s.AddTool(mcp.NewTool(
		"delete_data_source",
		mcp.WithDescription("Remove a data source submission."),
		mcp.WithNumber("data_source_id", mcp.Description("Data source ID"), mcp.Required()),
		destructive(),
	), tools.DeleteDataSource(c))

	s.AddTool(mcp.NewTool(
		"add_data_source_note",
		mcp.WithDescription("Add a note to a data source request (for communication with the review team)."),
		mcp.WithNumber("data_source_id", mcp.Description("Data source ID"), mcp.Required()),
		mcp.WithString("notes", mcp.Description("Note text"), mcp.Required()),
	), tools.AddDataSourceNote(c))

	s.AddTool(mcp.NewTool(
		"upload_data_source",
		mcp.WithDescription("Upload a file (JSON, XML, CSV, or TXT) as a new data source."),
		mcp.WithString("file_base64", mcp.Description("Base64-encoded file content. The decoded file must not exceed 2 MiB."), mcp.Required()),
		mcp.WithString("filename", mcp.Description("Filename with extension, e.g. feed.json, data.csv"), mcp.Required()),
	), tools.UploadDataSource(c))

	// ── Domains ───────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_domains",
		mcp.WithDescription("List custom domains configured for this account."),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.ListDomains(c))

	s.AddTool(mcp.NewTool(
		"create_domain",
		mcp.WithDescription("Add a new custom domain. You will need to verify it via DNS after creation."),
		mcp.WithString("domain", mcp.Description("Domain or subdomain, e.g. poltio.yourdomain.com"), mcp.Required()),
		mcp.WithNumber("is_default", mcp.Description("Set as default domain: 0 or 1")),
		mcp.WithNumber("is_active", mcp.Description("Enable the domain: 0 or 1")),
	), tools.CreateDomain(c))

	s.AddTool(mcp.NewTool(
		"update_domain",
		mcp.WithDescription("Update a custom domain's settings. The domain name itself cannot be changed after creation — delete and re-create instead."),
		mcp.WithNumber("domain_id", mcp.Description("Domain ID"), mcp.Required()),
		mcp.WithNumber("is_default", mcp.Description("Set as default: 0 or 1")),
		mcp.WithNumber("is_active", mcp.Description("Enable/disable: 0 or 1")),
	), tools.UpdateDomain(c))

	s.AddTool(mcp.NewTool(
		"delete_domain",
		mcp.WithDescription("Delete a custom domain."),
		mcp.WithNumber("domain_id", mcp.Description("Domain ID"), mcp.Required()),
		destructive(),
	), tools.DeleteDomain(c))

	// ── Widgets ───────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_widgets",
		mcp.WithDescription("List your existing dynamic widgets."),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
		mcp.WithString("public_id", mcp.Description("Filter by content public_id")),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.ListWidgets(c))

	s.AddTool(mcp.NewTool(
		"create_widget",
		mcp.WithDescription("Create a Dynamic Widget: a placement rule that auto-loads a chosen content item on your site through the Poltio embed snippet, either on every page or only on specific page URLs. Lets you swap which content shows where without editing your site code."),
		mcp.WithString("public_id", mcp.Description("Public ID of the content this widget displays"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Internal name for the widget")),
		mcp.WithNumber("is_default", mcp.Description("Make this the default widget shown on every page with the snippet: 0 or 1. When 1, omit urls.")),
		mcp.WithNumber("is_active", mcp.Description("Enable the widget: 0 or 1")),
		mcp.WithString("urls", mcp.Description("Comma-separated page URLs where this widget should appear (Specific Page targeting). For all pages, pass is_default=1 instead.")),
		mcp.WithString("starts_at", mcp.Description("Schedule start: only show the widget from this moment, format 'YYYY-MM-DD HH:MM:SS'")),
		mcp.WithString("ends_at", mcp.Description("Schedule end: stop showing the widget after this moment, format 'YYYY-MM-DD HH:MM:SS'")),
		mcp.WithString("overlay_options_json", mcp.Description("Appearance/behavior config as a JSON object (as produced by the dashboard widget editor): trigger type (card, pill, box, product_card, iframe), trigger position, colors, collapsed state, show-on-load/show-on-scroll, delay, and per-device (mobile) overrides. Read an existing widget with get_widget to see the shape before setting this.")),
	), tools.CreateWidget(c))

	s.AddTool(mcp.NewTool(
		"get_widget",
		mcp.WithDescription("Get a single Dynamic Widget."),
		mcp.WithNumber("widget_id", mcp.Description("Widget ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetWidget(c))

	s.AddTool(mcp.NewTool(
		"update_widget",
		mcp.WithDescription("Update an existing Dynamic Widget."),
		mcp.WithNumber("widget_id", mcp.Description("Widget ID"), mcp.Required()),
		mcp.WithString("public_id", mcp.Description("Content public identifier")),
		mcp.WithString("name", mcp.Description("Widget name")),
		mcp.WithNumber("is_default", mcp.Description("Set as default widget shown on every page: 0 or 1. When 1, omit urls.")),
		mcp.WithNumber("is_active", mcp.Description("Enable the widget: 0 or 1")),
		mcp.WithString("urls", mcp.Description("Comma-separated page URLs where this widget should appear (Specific Page targeting). For all pages, pass is_default=1 instead.")),
		mcp.WithString("starts_at", mcp.Description("Schedule start: only show the widget from this moment, format 'YYYY-MM-DD HH:MM:SS'")),
		mcp.WithString("ends_at", mcp.Description("Schedule end: stop showing the widget after this moment, format 'YYYY-MM-DD HH:MM:SS'")),
		mcp.WithString("overlay_options_json", mcp.Description("Appearance/behavior config as a JSON object (as produced by the dashboard widget editor): trigger type (card, pill, box, product_card, iframe), trigger position, colors, collapsed state, show-on-load/show-on-scroll, delay, and per-device (mobile) overrides. Read the widget with get_widget first and send back the modified object.")),
	), tools.UpdateWidget(c))

	s.AddTool(mcp.NewTool(
		"delete_widget",
		mcp.WithDescription("Delete an existing dynamic widget."),
		mcp.WithNumber("widget_id", mcp.Description("Widget ID"), mcp.Required()),
		destructive(),
	), tools.DeleteWidget(c))

	// ── Settings ──────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"update_settings",
		mcp.WithDescription("Update your account's username, email, or profile photo."),
		mcp.WithString("username", mcp.Description("New unique username")),
		mcp.WithString("email", mcp.Description("New email address (requires re-verification)")),
		mcp.WithString("photo", mcp.Description("Profile photo file path")),
	), tools.UpdateSettings(c))

	s.AddTool(mcp.NewTool(
		"update_password",
		mcp.WithDescription("Change your account password."),
		mcp.WithString("password", mcp.Description("Current password"), mcp.Required()),
		mcp.WithString("new_password", mcp.Description("New password"), mcp.Required()),
		mcp.WithString("new_password_confirmation", mcp.Description("New password confirmation"), mcp.Required()),
	), tools.UpdatePassword(c))

	s.AddTool(mcp.NewTool(
		"resend_verification",
		mcp.WithDescription("Resend the account email verification link."),
	), tools.ResendVerification(c))

	s.AddTool(mcp.NewTool(
		"accept_terms",
		mcp.WithDescription("Accept the Poltio terms and conditions for the current account."),
	), tools.AcceptTerms(c))

	s.AddTool(mcp.NewTool(
		"setup_two_factor",
		mcp.WithDescription("Begin two-factor authentication setup. Returns a base64 QR code image to scan with an authenticator app."),
	), tools.SetupTwoFactor(c))

	s.AddTool(mcp.NewTool(
		"verify_two_factor",
		mcp.WithDescription("Confirm 2FA setup with a TOTP verification code. Returns recovery codes on success."),
		mcp.WithString("verification", mcp.Description("6-digit TOTP code from your authenticator app"), mcp.Required()),
	), tools.VerifyTwoFactor(c))

	s.AddTool(mcp.NewTool(
		"disable_two_factor",
		mcp.WithDescription("Disable two-factor authentication on the current account. Requires a TOTP verification code."),
		mcp.WithString("verification", mcp.Description("6-digit TOTP code from your authenticator app"), mcp.Required()),
		destructive(),
	), tools.DisableTwoFactor(c))

	s.AddTool(mcp.NewTool(
		"reset_two_factor_recovery_codes",
		mcp.WithDescription("Regenerate 2FA recovery codes. Existing codes are invalidated. Requires a TOTP verification code."),
		mcp.WithString("verification", mcp.Description("6-digit TOTP code from your authenticator app"), mcp.Required()),
	), tools.ResetTwoFactorRecoveryCodes(c))

	s.AddTool(mcp.NewTool(
		"list_conversion_settings",
		mcp.WithDescription("List the checkout/success page URLs registered for conversion tracking. When a user who interacted with a Poltio widget later lands on one of these pages, Poltio attributes a conversion — letting you measure widget-driven sales/sign-ups."),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.ListConversionSettings(c))

	s.AddTool(mcp.NewTool(
		"create_conversion_setting",
		mcp.WithDescription("Register a checkout/order-success page URL so Poltio can count it as a conversion when reached by users who engaged with a widget. Add the same success page your site shows after a purchase or sign-up."),
		mcp.WithString("url", mcp.Description("The checkout/order-success page URL on your site, e.g. https://shop.example.com/order/complete. Use ^ as a wildcard matching any single path segment, e.g. https://shop.example.com/^/order/^; without ^ the URL must match exactly."), mcp.Required()),
		mcp.WithNumber("catch_all", mcp.Description("If 1, count every visit to this URL as a conversion; if 0, only count visits attributable to a prior widget interaction. Default: 1.")),
	), tools.CreateConversionSetting(c))

	s.AddTool(mcp.NewTool(
		"update_conversion_setting",
		mcp.WithDescription("Update a registered conversion success URL."),
		mcp.WithNumber("conversion_setting_id", mcp.Description("Conversion setting ID"), mcp.Required()),
		mcp.WithString("url", mcp.Description("New checkout/order-success page URL, e.g. https://shop.example.com/order/complete. Use ^ as a wildcard matching any single path segment.")),
		mcp.WithNumber("catch_all", mcp.Description("If 1, count every visit to this URL as a conversion; if 0, only those attributable to a widget interaction.")),
	), tools.UpdateConversionSetting(c))

	s.AddTool(mcp.NewTool(
		"delete_conversion_setting",
		mcp.WithDescription("Delete a conversion tracking URL."),
		mcp.WithNumber("conversion_setting_id", mcp.Description("Conversion setting ID"), mcp.Required()),
		destructive(),
	), tools.DeleteConversionSetting(c))

	// ── Organizations ─────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_organizations",
		mcp.WithDescription("List Poltio organizations the current user belongs to, including their role in each."),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.ListOrganizations(c))

	s.AddTool(mcp.NewTool(
		"switch_organization",
		mcp.WithDescription("Switch the active organization context. All subsequent tool calls will operate under the selected organization."),
		mcp.WithNumber("id", mcp.Description("Organization ID (from list_organizations)"), mcp.Required()),
	), tools.SwitchOrganization(c))

	s.AddTool(mcp.NewTool(
		"get_organization",
		mcp.WithDescription("Get an organization's details including members and pending invites."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.GetOrganization(c))

	s.AddTool(mcp.NewTool(
		"update_organization",
		mcp.WithDescription("Update an organization's name."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("New organization name"), mcp.Required()),
	), tools.UpdateOrganization(c))

	s.AddTool(mcp.NewTool(
		"invite_org_member",
		mcp.WithDescription("Invite a new member to an organization via email."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithString("email", mcp.Description("Email address of the user to invite"), mcp.Required()),
		mcp.WithString("role", mcp.Description("Role to assign: admin, user, or viewer. (The 'owner' role exists but cannot be assigned.)"), mcp.Required()),
	), tools.InviteOrgMember(c))

	s.AddTool(mcp.NewTool(
		"join_organization",
		mcp.WithDescription("Join an organization using an invite token from an invitation email."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithString("token", mcp.Description("Invite token from the invitation email"), mcp.Required()),
	), tools.JoinOrganization(c))

	s.AddTool(mcp.NewTool(
		"leave_organization",
		mcp.WithDescription("Leave an organization (cannot be used by the owner)."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		destructive(),
	), tools.LeaveOrganization(c))

	s.AddTool(mcp.NewTool(
		"cancel_org_invite",
		mcp.WithDescription("Cancel a pending organization invitation by email."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithString("email", mcp.Description("Email of the pending invite to cancel"), mcp.Required()),
		destructive(),
	), tools.CancelOrgInvite(c))

	s.AddTool(mcp.NewTool(
		"remove_org_member",
		mcp.WithDescription("Remove a member from an organization."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithNumber("user_id", mcp.Description("User ID of the member to remove"), mcp.Required()),
		destructive(),
	), tools.RemoveOrgMember(c))

	s.AddTool(mcp.NewTool(
		"update_org_member",
		mcp.WithDescription("Update a member's role in an organization."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithNumber("user_id", mcp.Description("User ID of the member"), mcp.Required()),
		mcp.WithString("role", mcp.Description("New role: admin, user, or viewer. (The 'owner' role exists but cannot be assigned.)"), mcp.Required()),
	), tools.UpdateOrgMember(c))

	s.AddTool(mcp.NewTool(
		"list_ip_rules",
		mcp.WithDescription("List the IP access rules of an organization."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.ListIPRules(c))

	s.AddTool(mcp.NewTool(
		"create_ip_rule",
		mcp.WithDescription("Create an IP access rule for an organization, allowing or blocking IP addresses or IPv4 CIDR blocks."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Rule name, e.g. 'Office Network & VPN'")),
		mcp.WithString("allowed_json", mcp.Description(`JSON array of allowed IPs or IPv4 CIDR blocks, e.g. ["192.168.1.0/24"]`)),
		mcp.WithString("blocked_json", mcp.Description(`JSON array of blocked IPs or IPv4 CIDR blocks, e.g. ["203.0.113.50"]`)),
	), tools.CreateIPRule(c))

	s.AddTool(mcp.NewTool(
		"update_ip_rule",
		mcp.WithDescription("Update an existing IP access rule of an organization."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithNumber("ip_rule_id", mcp.Description("IP rule ID (from list_ip_rules)"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Rule name, e.g. 'Office Network & VPN'")),
		mcp.WithString("allowed_json", mcp.Description(`JSON array of allowed IPs or IPv4 CIDR blocks, e.g. ["192.168.1.0/24"]`)),
		mcp.WithString("blocked_json", mcp.Description(`JSON array of blocked IPs or IPv4 CIDR blocks, e.g. ["203.0.113.50"]`)),
	), tools.UpdateIPRule(c))

	s.AddTool(mcp.NewTool(
		"delete_ip_rule",
		mcp.WithDescription("Delete an IP access rule from an organization."),
		mcp.WithNumber("organization_id", mcp.Description("Organization ID"), mcp.Required()),
		mcp.WithNumber("ip_rule_id", mcp.Description("IP rule ID (from list_ip_rules)"), mcp.Required()),
		destructive(),
	), tools.DeleteIPRule(c))

	// ── Misc ──────────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"search_playground",
		mcp.WithDescription("Test search queries and filters against a searchable content item."),
		mcp.WithNumber("content_id", mcp.Description("Content integer ID")),
		mcp.WithString("public_id", mcp.Description("Content public_id (use when content_id is unknown)")),
		mcp.WithString("query_json", mcp.Description(`JSON array of search terms, e.g. ["red shoes","running"]`)),
		mcp.WithString("filter_json", mcp.Description(`JSON array of filter expressions, e.g. ["price: [10...100]"]`)),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.SearchPlayground(c))

	s.AddTool(mcp.NewTool(
		"check_snippet_page",
		mcp.WithDescription("Check if a page URL has the Poltio snippet active and receiving requests in the last 48 hours."),
		mcp.WithString("url", mcp.Description("Fully qualified page URL to check"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.CheckSnippetPage(c))

	s.AddTool(mcp.NewTool(
		"create_short_link",
		mcp.WithDescription("Create a polt.io shortened URL from any long URL."),
		mcp.WithString("url", mcp.Description("Fully qualified URL to shorten"), mcp.Required()),
	), tools.CreateShortLink(c))

	// ── Subscription ──────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool(
		"list_subscription_tiers",
		mcp.WithDescription("List available subscription tiers and their features."),
		mcp.WithReadOnlyHintAnnotation(true),
	), tools.ListSubscriptionTiers(c))

	s.AddTool(mcp.NewTool(
		"create_subscription",
		mcp.WithDescription("Create a new subscription for the current organization."),
		mcp.WithNumber("tier_id", mcp.Description("Subscription tier ID (from list_subscription_tiers)"), mcp.Required()),
		mcp.WithString("period", mcp.Description("Billing period: month or year"), mcp.Required()),
	), tools.CreateSubscription(c))

	s.AddTool(mcp.NewTool(
		"cancel_subscription",
		mcp.WithDescription("Cancel the current organization's subscription. This is a billing-affecting action — only call it when the user explicitly asks to cancel."),
		destructive(),
	), tools.CancelSubscription(c))

	if port != "" {
		httpServer := server.NewStreamableHTTPServer(
			s,
			server.WithEndpointPath("/mcp"),
			server.WithStreamableHTTPCORS(server.WithCORSAllowedOrigins("*")),
		)
		log.Printf("poltio-mcp-server listening on :%s/mcp", port)
		if err := http.ListenAndServe(":"+port, httpServer); err != nil {
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
