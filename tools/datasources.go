package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func ListDataSources(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := c.Get("/platform/data-sources", nil)
		if err != nil {
			return nil, fmt.Errorf("list_data_sources: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateDataSource(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil || name == "" {
			return nil, fmt.Errorf("name is required")
		}
		source, err := req.RequireString("source")
		if err != nil || source == "" {
			return nil, fmt.Errorf("source is required (fully qualified URL for the feed, or the shop ID when type is shopify)")
		}
		feedType, err := req.RequireString("type")
		if err != nil || feedType == "" {
			return nil, fmt.Errorf("type is required (xml, json, csv, shopify)")
		}
		body := map[string]any{"name": name, "source": source, "type": feedType}
		if v := req.GetString("items_path", ""); v != "" {
			body["items_path"] = v
		}
		if v := req.GetString("user_agent", ""); v != "" {
			body["user_agent"] = v
		}
		if v := req.GetString("notes", ""); v != "" {
			body["notes"] = v
		}
		data, err := c.Post("/platform/data-sources", body)
		if err != nil {
			return nil, fmt.Errorf("create_data_source: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteDataSource(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		data, err := c.Delete("/platform/data-sources/" + strconv.Itoa(dataSourceID))
		if err != nil {
			return nil, fmt.Errorf("delete_data_source: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateCSVDataSource(c UploadClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil || name == "" {
			return nil, fmt.Errorf("name is required")
		}
		fileBase64, err := req.RequireString("file_base64")
		if err != nil || fileBase64 == "" {
			return nil, fmt.Errorf("file_base64 is required (base64-encoded CSV content)")
		}
		content, err := base64.StdEncoding.DecodeString(fileBase64)
		if err != nil {
			return nil, fmt.Errorf("file_base64 is not valid base64: %w", err)
		}
		filename := req.GetString("filename", "data.csv")
		fields := map[string]string{"type": "csv", "name": name}
		data, err := c.PostFormFileFields("/platform/data-sources", "source_file", filename, content, fields)
		if err != nil {
			return nil, fmt.Errorf("create_csv_data_source: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// CreateXMLDataSource creates a native xml data source. The API accepts
// items_path on create, so the feed is read by the importer itself and stays in
// sync — no client-side flattening needed.
func CreateXMLDataSource(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil || name == "" {
			return nil, fmt.Errorf("name is required")
		}
		feedURL, err := req.RequireString("feed_url")
		if err != nil || feedURL == "" {
			return nil, fmt.Errorf("feed_url is required")
		}
		itemsPath, err := req.RequireString("items_path")
		if err != nil || itemsPath == "" {
			return nil, fmt.Errorf("items_path is required (item node name, e.g. item, product, entry)")
		}
		body := map[string]any{
			"name":       name,
			"source":     feedURL,
			"type":       "xml",
			"items_path": itemsPath,
		}
		if v := req.GetString("user_agent", ""); v != "" {
			body["user_agent"] = v
		}
		data, err := c.Post("/platform/data-sources", body)
		if err != nil {
			return nil, fmt.Errorf("create_xml_data_source: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetDataSource(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		data, err := c.Get("/platform/data-sources/"+strconv.Itoa(dataSourceID), nil)
		if err != nil {
			return nil, fmt.Errorf("get_data_source: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetDataSourceAttributes(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		data, err := c.Get("/platform/data-sources/"+strconv.Itoa(dataSourceID)+"/format", nil)
		if err != nil {
			return nil, fmt.Errorf("get_data_source_attributes: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func SetDataSourceElements(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		raw, err := req.RequireString("elements_json")
		if err != nil || raw == "" {
			return nil, fmt.Errorf("elements_json is required")
		}
		var elements []map[string]any
		if err := json.Unmarshal([]byte(raw), &elements); err != nil {
			return nil, fmt.Errorf("elements_json is not a valid JSON array: %w", err)
		}
		// The endpoint validates one element per request: a body wrapping them in
		// an "elements" array is rejected for missing element and type.
		path := "/platform/data-sources/" + strconv.Itoa(dataSourceID) + "/elements"
		var created []string
		var failures []string
		for _, element := range elements {
			data, err := c.Post(path, element)
			if err != nil {
				failures = append(failures, fmt.Sprintf("%v: %v", element["element"], err))
				continue
			}
			created = append(created, string(data))
		}
		result := fmt.Sprintf("Created %d of %d elements.\n[%s]", len(created), len(elements), strings.Join(created, ",\n"))
		if len(failures) > 0 {
			result += "\n\nFailed:\n" + strings.Join(failures, "\n")
		}
		return mcp.NewToolResultText(result), nil
	}
}

func GetDataSourceItems(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		// The endpoint paginates by a fixed 25; per_page is not read.
		data, err := c.Get("/platform/data-sources/"+strconv.Itoa(dataSourceID)+"/items", q)
		if err != nil {
			return nil, fmt.Errorf("get_data_source_items: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func PublishDataSource(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		data, err := c.Post("/platform/data-sources/"+strconv.Itoa(dataSourceID)+"/mark-ready", nil)
		if err != nil {
			// An import already pending or running answers 400 with this message.
			// The source is in the state the caller asked for, so it is not a
			// failure. Matched on the message, not the status, so that any other
			// 400 still surfaces as the error it is.
			if strings.Contains(err.Error(), "already pending or in progress") {
				return mcp.NewToolResultText("The data source is already importing (nothing to do)."), nil
			}
			return nil, fmt.Errorf("publish_data_source: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UploadDataSource(c UploadClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileBase64, err := req.RequireString("file_base64")
		if err != nil || fileBase64 == "" {
			return nil, fmt.Errorf("file_base64 is required (base64-encoded file content)")
		}
		filename, err := req.RequireString("filename")
		if err != nil || filename == "" {
			return nil, fmt.Errorf("filename is required (e.g. feed.json, data.csv)")
		}
		content, err := base64.StdEncoding.DecodeString(fileBase64)
		if err != nil {
			return nil, fmt.Errorf("file_base64 is not valid base64: %w", err)
		}
		data, err := c.PostFormFile("/platform/data-sources/upload", "file", filename, content)
		if err != nil {
			return nil, fmt.Errorf("upload_data_source: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateDataSource(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		body := map[string]any{}
		for _, key := range []string{"name", "type", "source", "items_path", "user_agent"} {
			if v := req.GetString(key, ""); v != "" {
				body[key] = v
			}
		}
		if len(body) == 0 {
			return nil, fmt.Errorf("nothing to update: pass at least one of name, type, source, items_path, user_agent")
		}
		data, err := c.Put("/platform/data-sources/"+strconv.Itoa(dataSourceID), body)
		if err != nil {
			return nil, fmt.Errorf("update_data_source: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func RefreshDataSourceFormat(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		data, err := c.Post("/platform/data-sources/"+strconv.Itoa(dataSourceID)+"/refresh-format", nil)
		if err != nil {
			return nil, fmt.Errorf("refresh_data_source_format: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetDataSourceElements(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		data, err := c.Get("/platform/data-sources/"+strconv.Itoa(dataSourceID)+"/elements", nil)
		if err != nil {
			return nil, fmt.Errorf("get_data_source_elements: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateDataSourceElement(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		elementID, err := req.RequireInt("element_id")
		if err != nil {
			return nil, fmt.Errorf("element_id is required")
		}
		body := map[string]any{}
		for _, key := range []string{"type", "element", "slug", "namespace"} {
			if v := req.GetString(key, ""); v != "" {
				body[key] = v
			}
		}
		if len(body) == 0 {
			return nil, fmt.Errorf("nothing to update: pass at least one of type, element, slug, namespace")
		}
		path := "/platform/data-sources/" + strconv.Itoa(dataSourceID) + "/elements/" + strconv.Itoa(elementID)
		data, err := c.Put(path, body)
		if err != nil {
			return nil, fmt.Errorf("update_data_source_element: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteDataSourceElement(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		elementID, err := req.RequireInt("element_id")
		if err != nil {
			return nil, fmt.Errorf("element_id is required")
		}
		path := "/platform/data-sources/" + strconv.Itoa(dataSourceID) + "/elements/" + strconv.Itoa(elementID)
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("delete_data_source_element: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// GetDataSourceItem reads one item. The API resolves the identifier against
// both the internal id and the feed's own source_id, so either works.
func GetDataSourceItem(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		item, err := req.RequireString("item")
		if err != nil || item == "" {
			return nil, fmt.Errorf("item is required (the item's id or its source_id)")
		}
		path := "/platform/data-sources/" + strconv.Itoa(dataSourceID) + "/items/" + url.PathEscape(item)
		data, err := c.Get(path, nil)
		if err != nil {
			return nil, fmt.Errorf("get_data_source_item: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// CreateDataSourceItems adds items to a manual or Shopify source. items_json
// takes either one object or an array — the API accepts a batch of up to 100
// under an "items" key, and reports per-item failures without failing the rest.
func CreateDataSourceItems(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		raw, err := req.RequireString("items_json")
		if err != nil || raw == "" {
			return nil, fmt.Errorf("items_json is required")
		}
		body, err := itemsPayload(raw)
		if err != nil {
			return nil, err
		}
		path := "/platform/data-sources/" + strconv.Itoa(dataSourceID) + "/items"
		data, err := c.Post(path, body)
		if err != nil {
			return nil, fmt.Errorf("create_data_source_items: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// itemsPayload turns the caller's JSON into the shape the API expects: an
// array becomes a batch under "items", a single object is sent as the item
// itself. Decoded rather than forwarded verbatim so malformed JSON fails here
// with a clear message instead of as an opaque API 422.
func itemsPayload(raw string) (map[string]any, error) {
	var arr []any
	if err := json.Unmarshal([]byte(raw), &arr); err == nil {
		if len(arr) == 0 {
			return nil, fmt.Errorf("items_json is an empty array: pass at least one item")
		}
		if len(arr) > 100 {
			return nil, fmt.Errorf("items_json holds %d items: the API accepts at most 100 per call", len(arr))
		}
		return map[string]any{"items": arr}, nil
	}
	var one map[string]any
	if err := json.Unmarshal([]byte(raw), &one); err != nil {
		return nil, fmt.Errorf("items_json must be a JSON object or array of objects: %w", err)
	}
	return map[string]any{"item": one}, nil
}

// UpdateDataSourceItem replaces one item. The API fills source_id, name and url
// from the stored item when they are absent, so a partial payload is fine.
func UpdateDataSourceItem(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		item, err := req.RequireString("item")
		if err != nil || item == "" {
			return nil, fmt.Errorf("item is required (the item's id or its source_id)")
		}
		raw, err := req.RequireString("item_json")
		if err != nil || raw == "" {
			return nil, fmt.Errorf("item_json is required")
		}
		var one map[string]any
		if err := json.Unmarshal([]byte(raw), &one); err != nil {
			return nil, fmt.Errorf("item_json must be a JSON object: %w", err)
		}
		path := "/platform/data-sources/" + strconv.Itoa(dataSourceID) + "/items/" + url.PathEscape(item)
		data, err := c.Put(path, map[string]any{"item": one})
		if err != nil {
			return nil, fmt.Errorf("update_data_source_item: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteDataSourceItem(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		item, err := req.RequireString("item")
		if err != nil || item == "" {
			return nil, fmt.Errorf("item is required (the item's id or its source_id)")
		}
		path := "/platform/data-sources/" + strconv.Itoa(dataSourceID) + "/items/" + url.PathEscape(item)
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("delete_data_source_item: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
