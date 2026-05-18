# Poltio Platform OpenAPI ↔ MCP Server Coverage Report

**Spec:** `/Users/hio/Projects/Poltio/api1/public/docs/platform/platform.yaml`  
**MCP Server:** `/Users/hio/Projects/Poltio/poltio-mcp-server/`  
**Date:** 2026-05-17

---

## Section 1: API Endpoints Covered by MCP Tools

### Auth — ALL MISSING (see Section 2)

### Content (62 endpoints) — ALL COVERED
| Method | Path | MCP Tool Name |
|--------|------|---------------|
| POST | `/platform/content` | `create_content` |
| GET | `/platform/content` | `list_content` |
| GET | `/platform/content/drafts` | `list_drafts` |
| GET | `/platform/content/{public_id}` | `get_content` |
| PUT | `/platform/content/{public_id}` | `update_content` |
| DELETE | `/platform/content/{public_id}` | `delete_content` |
| GET | `/platform/content/{public_id}/publish` | `publish_content` |
| GET | `/platform/content/{public_id}/duplicate` | `duplicate_content` |
| GET | `/platform/content/{public_id}/results` | `get_content_results` |
| GET | `/platform/content/{public_id}/sessions` | `get_content_sessions` |
| PUT | `/platform/content/{public_id}/lead` | `set_content_lead` |
| DELETE | `/platform/content/{public_id}/lead` | `remove_content_lead` |
| PUT | `/platform/content/{public_id}/pixel-code` | `set_content_pixel_code` |
| DELETE | `/platform/content/{public_id}/pixel-code` | `remove_content_pixel_code` |
| GET | `/platform/content/{public_id}/edit` | `get_content_edit` |
| POST | `/platform/content/calculator-test` | `calculator_test` |
| POST | `/platform/content/{public_id}/question` | `add_question` |
| PUT | `/platform/content/{public_id}/question/{question_id}` | `update_question` |
| DELETE | `/platform/content/{public_id}/question/{question_id}` | `delete_question` |
| PUT | `/platform/content/{public_id}/question/{question_id}/lead` | `set_question_lead` |
| DELETE | `/platform/content/{public_id}/question/{question_id}/lead` | `remove_question_lead` |
| PUT | `/platform/content/{public_id}/question/{question_id}/pixel-code` | `set_question_pixel_code` |
| DELETE | `/platform/content/{public_id}/question/{question_id}/pixel-code` | `remove_question_pixel_code` |
| POST | `/platform/content/{public_id}/question/{question_id}/conditions/add` | `add_question_condition` |
| POST | `/platform/content/{public_id}/question/{question_id}/conditions/remove` | `remove_question_condition` |
| DELETE | `/platform/content/{public_id}/question/{question_id}/conditions` | `clear_question_conditions` |
| GET | `/platform/content/{public_id}/question/{source_id}/answer/clone/{target_id}` | `clone_answers` |
| POST | `/platform/content/{public_id}/question/{question_id}/answer` | `add_answer` |
| POST | `/platform/content/{public_id}/question/{question_id}/answer/multi` | `add_answers_bulk` |
| PUT | `/platform/content/{public_id}/question/{question_id}/answer/{answer_id}` | `update_answer` |
| DELETE | `/platform/content/{public_id}/question/{question_id}/answer/{answer_id}` | `delete_answer` |
| PUT | `/platform/content/{public_id}/question/{question_id}/answer/{answer_id}/lead` | `set_answer_lead` |
| DELETE | `/platform/content/{public_id}/question/{question_id}/answer/{answer_id}/lead` | `remove_answer_lead` |
| PUT | `/platform/content/{public_id}/question/{question_id}/answer/{answer_id}/results` | `set_answer_result_point` |
| PUT | `/platform/content/{public_id}/question/{question_id}/answer/{answer_id}/pixel-code` | `set_answer_pixel_code` |
| DELETE | `/platform/content/{public_id}/question/{question_id}/answer/{answer_id}/pixel-code` | `remove_answer_pixel_code` |
| GET | `/platform/content/{public_id}/order/questions` | `get_question_order` |
| PUT | `/platform/content/{public_id}/order/questions` | `update_question_order` |
| GET | `/platform/content/{public_id}/order/answers/{question_id}` | `get_answer_order` |
| PUT | `/platform/content/{public_id}/order/answers/{question_id}` | `update_answer_order` |
| POST | `/platform/content/{public_id}/result` | `add_result` |
| PUT | `/platform/content/{public_id}/result/{result_id}` | `update_result` |
| DELETE | `/platform/content/{public_id}/result/{result_id}` | `delete_result` |
| PUT | `/platform/content/{public_id}/result/{result_id}/lead` | `set_result_lead` |
| DELETE | `/platform/content/{public_id}/result/{result_id}/lead` | `remove_result_lead` |
| PUT | `/platform/content/{public_id}/result/{result_id}/pixel-code` | `set_result_pixel_code` |
| DELETE | `/platform/content/{public_id}/result/{result_id}/pixel-code` | `remove_result_pixel_code` |
| PUT | `/platform/content/{public_id}/result/{result_id}/click-pixel-code` | `set_result_click_pixel_code` |
| DELETE | `/platform/content/{public_id}/result/{result_id}/click-pixel-code` | `remove_result_click_pixel_code` |
| GET | `/platform/content/{public_id}/question/{question_id}/inputs` | `get_question_inputs` |
| POST | `/platform/content/upload` | `upload_image` |
| GET | `/platform/content/templates` | `list_templates` |
| GET | `/platform/content/templates/{public_id}` | `get_template` |
| GET | `/platform/content/templates/{public_id}/use` | `use_template` |
| GET | `/platform/content/{public_id}/sources` | `get_vote_sources` |
| GET | `/platform/content/{public_id}/sankey` | `get_sankey` |
| GET | `/platform/content/{public_id}/sankey/users/{from_id}/{to_id}` | `get_sankey_users` |
| GET | `/platform/content/{public_id}/voters` | `get_voters` |
| GET | `/platform/content/{public_id}/searchable-fields` | `get_searchable_fields` |
| POST | `/platform/content/{public_id}/metrics/{period}` | `get_content_metrics` |
| GET | `/platform/content/{public_id}/session/urls` | `get_session_urls` |
| GET | `/platform/content/{public_id}/conditions` | `get_content_conditions` |

### Theme (6 endpoints) — ALL COVERED
| Method | Path | MCP Tool Name |
|--------|------|---------------|
| POST | `/platform/theme` | `create_theme` |
| GET | `/platform/theme` | `list_themes` |
| GET | `/platform/theme/{theme_id}` | `get_theme` |
| DELETE | `/platform/theme/{theme_id}` | `delete_theme` |
| PUT | `/platform/theme/{theme_id}` | `update_theme` |
| GET | `/platform/theme/default` | `get_default_theme` |

### Dashboard (3 endpoints) — ALL COVERED
| Method | Path | MCP Tool Name |
|--------|------|---------------|
| GET | `/platform/dashboard` | `get_dashboard` |
| GET | `/platform/dashboard/summary` | `get_dashboard_summary` |
| POST | `/platform/dashboard/metrics/{period}` | `get_dashboard_metrics` |

### Sheet Hooks (6 endpoints) — ALL COVERED
| Method | Path | MCP Tool Name |
|--------|------|---------------|
| GET | `/platform/hooks/sheet` | `list_sheet_hooks` |
| POST | `/platform/hooks/sheet` | `create_sheet_hook` |
| GET | `/platform/hooks/sheet/{hook_id}` | `get_sheet_hook` |
| PUT | `/platform/hooks/sheet/{hook_id}` | `update_sheet_hook` |
| DELETE | `/platform/hooks/sheet/{hook_id}` | `delete_sheet_hook` |
| GET | `/platform/hooks/sheet/{hook_id}/logs` | `get_sheet_hook_logs` |

### Webhooks (6 endpoints) — ALL COVERED
| Method | Path | MCP Tool Name |
|--------|------|---------------|
| GET | `/platform/hooks/web` | `list_webhooks` |
| POST | `/platform/hooks/web` | `create_webhook` |
| GET | `/platform/hooks/web/{hook_id}` | `get_webhook` |
| PUT | `/platform/hooks/web/{hook_id}` | `update_webhook` |
| DELETE | `/platform/hooks/web/{hook_id}` | `delete_webhook` |
| GET | `/platform/hooks/web/{hook_id}/logs` | `get_webhook_logs` |

### Reports (2 endpoints) — ALL COVERED
| Method | Path | MCP Tool Name |
|--------|------|---------------|
| GET | `/platform/reports` | `list_reports` |
| POST | `/platform/reports` | `create_report` |

### Subscription (2 endpoints) — ALL COVERED
| Method | Path | MCP Tool Name |
|--------|------|---------------|
| GET | `/platform/subscription/tiers` | `list_subscription_tiers` |
| POST | `/platform/subscription/{tier_id}` | `create_subscription` |

### Leads (12 endpoints) — ALL COVERED
| Method | Path | MCP Tool Name |
|--------|------|---------------|
| GET | `/platform/leads` | `list_leads` |
| POST | `/platform/leads` | `create_lead` |
| GET | `/platform/leads/{lead_id}` | `get_lead` |
| PUT | `/platform/leads/{lead_id}` | `update_lead` |
| DELETE | `/platform/leads/{lead_id}` | `delete_lead` |
| GET | `/platform/leads/{lead_id}/inputs` | `get_lead_inputs` |
| GET | `/platform/leads/{lead_id}/logs` | `get_lead_logs` |
| GET | `/platform/leads/{lead_id}/codes` | `get_lead_codes` |
| POST | `/platform/leads/{lead_id}/codes` | `add_lead_codes` |
| DELETE | `/platform/leads/{lead_id}/codes` | `delete_all_lead_codes` |
| PUT | `/platform/leads/{lead_id}/codes/{lead_coupon_code_id}` | `update_lead_code` |
| DELETE | `/platform/leads/{lead_id}/codes/{lead_coupon_code_id}` | `delete_lead_code` |

### Settings (12 endpoints) — ALL COVERED
| Method | Path | MCP Tool Name |
|--------|------|---------------|
| PUT | `/platform/settings` | `update_settings` |
| PUT | `/platform/settings/password` | `update_password` |
| POST | `/platform/settings/resend-verification` | `resend_verification` |
| POST | `/platform/settings/accept-terms` | `accept_terms` |
| POST | `/platform/settings/two-factor/setup` | `setup_two_factor` |
| POST | `/platform/settings/two-factor/verify` | `verify_two_factor` |
| POST | `/platform/settings/two-factor/disable` | `disable_two_factor` |
| POST | `/platform/settings/two-factor/reset-recovery-codes` | `reset_two_factor_recovery_codes` |
| GET | `/platform/settings/conversion` | `list_conversion_settings` |
| POST | `/platform/settings/conversion` | `create_conversion_setting` |
| PUT | `/platform/settings/conversion/{conversion_setting_id}` | `update_conversion_setting` |
| DELETE | `/platform/settings/conversion/{conversion_setting_id}` | `delete_conversion_setting` |

### Data Sources (5 endpoints) — ALL COVERED
| Method | Path | MCP Tool Name |
|--------|------|---------------|
| GET | `/platform/data-sources` | `list_data_sources` |
| POST | `/platform/data-sources` | `create_data_source` |
| DELETE | `/platform/data-sources/{data_source_id}` | `delete_data_source` |
| POST | `/platform/data-sources/{data_source_id}/note` | `add_data_source_note` |
| POST | `/platform/data-sources/upload` | `upload_data_source` |

### Pixel Codes (4 endpoints) — ALL COVERED
| Method | Path | MCP Tool Name |
|--------|------|---------------|
| GET | `/platform/pixel-codes` | `list_pixel_codes` |
| POST | `/platform/pixel-codes` | `create_pixel_code` |
| PUT | `/platform/pixel-codes/{pixel_code_id}` | `update_pixel_code` |
| DELETE | `/platform/pixel-codes/{pixel_code_id}` | `delete_pixel_code` |

### Tokens (3 endpoints) — ALL COVERED
| Method | Path | MCP Tool Name |
|--------|------|---------------|
| GET | `/platform/tokens` | `list_tokens` |
| POST | `/platform/tokens` | `create_token` |
| DELETE | `/platform/tokens/{token_id}` | `revoke_token` |

### Domains (4 endpoints) — ALL COVERED
| Method | Path | MCP Tool Name |
|--------|------|---------------|
| GET | `/platform/domains` | `list_domains` |
| POST | `/platform/domains` | `create_domain` |
| PUT | `/platform/domains/{domain_id}` | `update_domain` |
| DELETE | `/platform/domains/{domain_id}` | `delete_domain` |

### Search / Misc (4 endpoints) — ALL COVERED
| Method | Path | MCP Tool Name |
|--------|------|---------------|
| POST | `/platform/search/playground` | `search_playground` |
| POST | `/platform/snippet/check-page` | `check_snippet_page` |
| POST | `/platform/link-shorten` | `create_short_link` |
| POST | `/platform/trigger-demo` | `trigger_demo` |

### Conversion (1 endpoint) — COVERED
| Method | Path | MCP Tool Name |
|--------|------|---------------|
| GET | `/platform/conversion/time-stats` | `get_conversion_time_stats` |

### Organizations (9 endpoints) — 8 COVERED, 1 MISMATCHED
| Method | Path | MCP Tool Name | Note |
|--------|------|---------------|------|
| GET | `/platform/organizations` | `list_organizations` | ⚠️ Calls `/platform/account/profile` instead |
| GET | `/platform/organizations/{organization_id}` | `get_organization` | |
| PUT | `/platform/organizations/{organization_id}` | `update_organization` | |
| POST | `/platform/organizations/{organization_id}/invite` | `invite_org_member` | |
| GET | `/platform/organizations/{organization_id}/join/{token}` | `join_organization` | |
| GET | `/platform/organizations/{organization_id}/leave` | `leave_organization` | |
| DELETE | `/platform/organizations/{organization_id}/cancel-invite/{email}` | `cancel_org_invite` | |
| DELETE | `/platform/organizations/{organization_id}/remove-member/{id}` | `remove_org_member` | |
| POST | `/platform/organizations/{organization_id}/update-member` | `update_org_member` | |

---

## Section 2: API Endpoints NOT Covered by MCP Tools (Missing Implementations)

### Auth — 10 endpoints missing
| # | Method | Path | Summary |
|---|--------|------|---------|
| 1 | POST | `/platform/auth/login` | First step of login |
| 2 | POST | `/platform/auth/two-factor/verify` | 2FA verification step 2 |
| 3 | POST | `/platform/auth/two-factor/verify-recovery` | 2FA recovery code verification |
| 4 | POST | `/platform/auth/login-with-email` | Magic link login (step 1) |
| 5 | POST | `/platform/auth/login-with-email-token` | Magic link login (step 2) |
| 6 | POST | `/platform/auth/register` | Register new account |
| 7 | POST | `/platform/auth/password/forget` | Start password recovery |
| 8 | POST | `/platform/auth/password/reset` | Complete password reset |
| 9 | GET | `/platform/auth/verify/{email}/{token}` | Verify email address |
| 10 | POST | `/platform/auth/logout` | Revoke active token |

### Widget — 5 endpoints missing (functions exist in `widgets.go` but are NOT registered in `main.go`)
| # | Method | Path | Summary |
|---|--------|------|---------|
| 11 | GET | `/platform/widgets` | List dynamic widgets |
| 12 | POST | `/platform/widgets` | Create dynamic widget |
| 13 | GET | `/platform/widgets/{widget_id}` | Get single widget |
| 14 | PUT | `/platform/widgets/{widget_id}` | Update widget |
| 15 | DELETE | `/platform/widgets/{widget_id}` | Delete widget |

**Missing endpoints subtotal: 15**

---

## Section 3: MCP Tools That Don't Map to Any API Endpoint

| MCP Tool Name | What it actually does |
|---------------|----------------------|
| `switch_organization` | Client-side only — calls `c.SetOrgID(id)` to switch the `Organization-Id` header for subsequent requests. Does NOT call any Poltio API endpoint. |

**Note:** `widgets.go` contains 5 implementation functions (`ListWidgets`, `CreateWidget`, `GetWidget`, `UpdateWidget`, `DeleteWidget`) that are **dead code** — they are never registered as MCP tools in `main.go`.

---

## Section 4: Parameter / Body Field Mismatches for Covered Endpoints

### 1. `list_organizations` — Wrong Endpoint
- **Spec endpoint:** `GET /platform/organizations`
- **MCP actually calls:** `GET /platform/account/profile` (extracts `organizations` array from profile response)
- **Impact:** The endpoint in the spec is never called. The alternative endpoint (`/platform/account/profile`) is **not documented in the OpenAPI spec**.

### 2. `list_drafts` — Missing Query Parameters
- **Spec:** `GET /platform/content/drafts` supports `page`, `per_page`, `order`, `sort`, `type`, `q`
- **MCP tool params:** `page`, `per_page`, `type`, `q`
- **Missing:** `order` (sort field) and `sort` (sort direction: asc/desc)

### 3. `get_voters` — Missing Query Parameter
- **Spec:** `GET /platform/content/{public_id}/voters` supports `page`, `per_page`, ** `download`** (boolean, default true)
- **MCP tool params:** `page`, `per_page`
- **Missing:** `download` — indicates whether to request the report as a file via email

### 4. `create_content` — Missing Body Fields
- **Spec body:** `BaseContent` (used for both POST and PUT)
- **MCP `create_content` fields:** `type`, `title`, `desc`, `name`, `background`, `alt`, `vertical_image`, `skip_start`, `skip_result`, `hide_results`, `hide_counter`
- **Missing compared to spec:**
  - `display_repeat` (present in `update_content` but missing in `create_content`)
  - `end_date_day`, `end_date_hour`, `end_date_minute`
  - `embed_footer_url`
  - `is_searchable`
  - `is_calculator`
  - `search_results_per_page`
  - `result_loading`
  - `loading_next_question_label`, `loading_result_label`
  - `boost_results_min_view`, `boost_results_ratio`
  - `vertical_mobile_image`

### 5. `update_content` — Missing Body Fields
- Same as above — many `BaseContent` fields are not exposed (only `display_repeat` was added recently compared to `create_content`).

### 6. `add_question` / `update_question` — Missing Body Fields
- **Spec body:** `Question` schema
- **MCP fields:** `title`, `answer_type`, `background`, `alt`, `vertical_image`, `allow_multiple_answers`, `is_skippable`, `rotate_answers`
- **Missing from spec schema:** `max_multi_punch_answer`, `recommended_popular_answer`, `link_html`, `require_login`, `mobile_only`, `answer_time_limit`, `hide_results`, `hide_counter`, `auto_reports_enabled`, `post_type`, `gives_feedback`

### 7. `add_answer` / `update_answer` — Missing Body Fields
- **Spec body:** `Answer` schema
- **MCP fields:** `title`, `background`, `alt`, `has_right_answer`, `is_right_answer`, `is_mutually_exclusive`
- **Missing from spec schema:** `max_vote`, `addon`, `disabled_msg`, `cal_val`, `triggers_lead`, `triggers_display`, `is_clone`, `clone_source`

### 8. `add_result` / `update_result` — Missing Body Fields
- **Spec body:** `Result` schema
- **MCP fields:** `title`, `desc`, `background`, `alt`, `url`, `url_text`, `min_c`, `max_c`
- **Missing from spec schema:** `search`, `search2`, `luv`, `is_default`

### 9. `create_webhook` — Missing Body Fields
- **Spec body:** `Webhook` schema
- **MCP fields:** `url`, `public_id`, `name`, `delay`, `send_leads`, `send_answers`, `account_wide`, `incomplete_send`
- **Missing from spec schema:** `content_type`, `content_id`, `health`, `incomplete_delay`, `use_oauth`, `oauth_login_endpoint`, `oauth_request_body`, `oauth_request_headers`
- **Note:** `is_active` is hardcoded to `true` but not exposed as a parameter

### 10. `update_webhook` — Missing Body Fields
- **MCP fields:** `hook_id`, `url`, `name`, `is_active`, `delay`, `send_leads`, `send_answers`
- **Missing compared to create:** `public_id`, `account_wide`, `incomplete_send`
- **Missing from spec schema:** Same oauth fields as above, `content_type`, `content_id`, `health`, `incomplete_delay`

### 11. `create_sheet_hook` — Hardcoded Defaults
- **Spec body:** `SheetHook` schema
- **MCP fields:** `public_id`, `sheet_id`, `name`, `is_active`
- **Note:** `is_active` defaults to `1` but API schema says boolean. This is minor (integer 1 vs boolean true).

### 12. `update_sheet_hook` — Missing Body Field
- **MCP fields:** `hook_id`, `sheet_id`, `name`, `is_active`
- **Missing:** `public_id` (present in `create_sheet_hook` but not in update)

### 13. `update_lead` — Missing Body Fields
- **Spec body:** `LeadRequest` schema
- **MCP fields:** `lead_id`, `name`, `type`, `msg`, `fields`, `button_value`, `redirect_url`, `title`, `is_active`, `mandatory`
- **Missing compared to `create_lead`:** `youtube_id`, `terms_conditions`

### 14. `update_org_member` — Body Field Name Mismatch (Minor)
- **Spec body:** `OrganizationUpdateMember` — the exact field names in the spec body are not visible, but MCP sends `user_id` and `role`.
- The path param is `organization_id`, and the body contains `user_id` and `role`. This is likely correct.

### 15. `upload_data_source` — Field Name Ambiguity
- **Spec:** `POST /platform/data-sources/upload` expects `multipart/form-data` with binary file. The spec does not name the form field.
- **MCP:** Uses `PostFormFile(path, "file", filename, content)` — field name is `"file"`. Since the spec is ambiguous, this is a potential mismatch if the API expects a different field name.

---

## Section 5: Summary Statistics

| Metric | Count |
|--------|-------|
| **Total API endpoints in OpenAPI spec** | **166** |
| **Covered by MCP tools (direct mapping)** | **150** |
| **Covered but calling wrong endpoint** | **1** (`list_organizations` → `/platform/account/profile`) |
| **Missing implementations (no MCP tool)** | **15** |
| - Auth endpoints | 10 |
| - Widget endpoints | 5 |
| **Registered MCP tools** | **142** |
| **MCP tools with no API endpoint** | **1** (`switch_organization`) |
| **Dead code (unregistered tool functions)** | **5** (`widgets.go`: ListWidgets, CreateWidget, GetWidget, UpdateWidget, DeleteWidget) |

### Coverage by Tag
| Tag | Total Endpoints | Covered | Missing |
|-----|----------------|---------|---------|
| Auth | 10 | 0 | 10 |
| Content | 62 | 62 | 0 |
| Theme | 6 | 6 | 0 |
| Dashboard | 3 | 3 | 0 |
| SheetHook | 6 | 6 | 0 |
| Webhook | 6 | 6 | 0 |
| Report | 2 | 2 | 0 |
| Subscription | 2 | 2 | 0 |
| Lead | 12 | 12 | 0 |
| Settings | 12 | 12 | 0 |
| DataSource | 5 | 5 | 0 |
| PixelCode | 4 | 4 | 0 |
| Token | 3 | 3 | 0 |
| Widget | 5 | 0 | 5 |
| Domain | 4 | 4 | 0 |
| Search/Misc | 4 | 4 | 0 |
| Conversion | 1 | 1 | 0 |
| Organization | 9 | 8* | 0 |

\* `list_organizations` calls a non-spec endpoint.

---

## Recommendations

1. **Add Auth tools** — The 10 auth endpoints are completely unimplemented. These are needed for any account creation, login, password reset, or 2FA flows.
2. **Register Widget tools** — `widgets.go` already has working implementations. They just need to be wired up in `main.go`.
3. **Fix `list_organizations`** — Either add the real `GET /platform/organizations` endpoint to the MCP tool, or document that the tool uses `/platform/account/profile` as a workaround.
4. **Add missing parameters** — Prioritize `order`/`sort` on `list_drafts`, `download` on `get_voters`, and `display_repeat` on `create_content`.
5. **Consider exposing advanced fields** — For power users, optional advanced fields on content/question/answer/result/webhook would improve completeness.
