package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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
			return nil, fmt.Errorf("source is required (fully qualified URL for the feed)")
		}
		feedType, err := req.RequireString("type")
		if err != nil || feedType == "" {
			return nil, fmt.Errorf("type is required (xml, json)")
		}
		body := map[string]any{"name": name, "source": source, "type": feedType}
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

// CreateXMLDataSource fetches a remote XML feed, flattens its items to CSV and
// creates the data source through the working CSV pipeline. This exists because
// the platform API cannot set items_path on xml-type sources, so a native xml
// source always imports 0 items.
// ponytail: snapshot import, no auto-sync from the feed; refresh = delete + recreate.
func CreateXMLDataSource(c UploadClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create_xml_data_source: invalid feed_url: %w", err)
		}
		httpReq.Header.Set("User-Agent", "PoltioMCP")
		resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("create_xml_data_source: fetching feed: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("create_xml_data_source: feed returned HTTP %d", resp.StatusCode)
		}

		headers, rows, err := xmlItemsToRows(resp.Body, itemsPath)
		if err != nil {
			return nil, fmt.Errorf("create_xml_data_source: parsing feed: %w", err)
		}
		if len(rows) == 0 {
			return nil, fmt.Errorf("create_xml_data_source: no <%s> items found in feed; check items_path", itemsPath)
		}

		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		_ = w.Write(headers)
		for _, row := range rows {
			rec := make([]string, len(headers))
			for i, h := range headers {
				rec[i] = row[h]
			}
			_ = w.Write(rec)
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return nil, fmt.Errorf("create_xml_data_source: writing csv: %w", err)
		}

		fields := map[string]string{"type": "csv", "name": name}
		data, err := c.PostFormFileFields("/platform/data-sources", "source_file", "feed.csv", buf.Bytes(), fields)
		if err != nil {
			return nil, fmt.Errorf("create_xml_data_source: %w", err)
		}
		return mcp.NewToolResultText(fmt.Sprintf("Imported %d items with columns [%s] from the XML feed.\n%s",
			len(rows), strings.Join(headers, ", "), string(data))), nil
	}
}

// xmlItemsToRows scans the stream for elements named itemsPath and flattens each
// one level deep: direct child element name -> concatenated text of its subtree.
// ponytail: attributes and nested structure are dropped; repeated child names keep the first value.
func xmlItemsToRows(r io.Reader, itemsPath string) ([]string, []map[string]string, error) {
	dec := xml.NewDecoder(r)
	var headers []string
	seen := map[string]bool{}
	var rows []map[string]string

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != itemsPath {
			continue
		}

		row := map[string]string{}
		depth := 0
		field := ""
		var val strings.Builder
	item:
		for {
			t, err := dec.Token()
			if err != nil {
				return nil, nil, err
			}
			switch tt := t.(type) {
			case xml.StartElement:
				depth++
				if depth == 1 {
					field = tt.Name.Local
					val.Reset()
				}
			case xml.CharData:
				if depth >= 1 {
					val.Write(tt)
				}
			case xml.EndElement:
				if depth == 0 {
					break item // closing tag of the item itself
				}
				if depth == 1 {
					if _, dup := row[field]; !dup {
						row[field] = strings.TrimSpace(val.String())
						if !seen[field] {
							seen[field] = true
							headers = append(headers, field)
						}
					}
				}
				depth--
			}
		}
		rows = append(rows, row)
	}
	return headers, rows, nil
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
		data, err := c.Get("/platform/data-sources/"+strconv.Itoa(dataSourceID)+"/attributes", nil)
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
		path := "/platform/data-sources/" + strconv.Itoa(dataSourceID) + "/elements"
		data, err := c.Post(path, map[string]any{"elements": elements})
		if err != nil {
			return nil, fmt.Errorf("set_data_source_elements: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
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
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
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
		data, err := c.Post("/platform/data-sources/"+strconv.Itoa(dataSourceID)+"/publish", nil)
		if err != nil {
			return nil, fmt.Errorf("publish_data_source: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func AddDataSourceNote(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		notes, err := req.RequireString("notes")
		if err != nil || notes == "" {
			return nil, fmt.Errorf("notes is required")
		}
		path := "/platform/data-sources/" + strconv.Itoa(dataSourceID) + "/note"
		data, err := c.Post(path, map[string]any{"notes": notes})
		if err != nil {
			return nil, fmt.Errorf("add_data_source_note: %w", err)
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
