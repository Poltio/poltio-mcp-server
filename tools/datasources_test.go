package tools

import (
	"strings"
	"testing"
)

func TestXMLItemsToRows(t *testing.T) {
	feed := `<?xml version="1.0"?>
<rss><channel>
  <title>ignored</title>
  <item>
    <g:id>KG-001</g:id>
    <title>AeroBrew X1</title>
    <link>https://example.com/p/1</link>
    <price>1200 <currency>TRY</currency></price>
  </item>
  <item>
    <g:id>KG-002</g:id>
    <title>Grinder Pro</title>
    <extra>only here</extra>
  </item>
</channel></rss>`

	headers, rows, err := xmlItemsToRows(strings.NewReader(feed), "item")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if got := strings.Join(headers, ","); got != "id,title,link,price,extra" {
		t.Fatalf("unexpected headers: %s", got)
	}
	if rows[0]["id"] != "KG-001" || rows[1]["id"] != "KG-002" {
		t.Fatalf("id values wrong: %v", rows)
	}
	if rows[0]["price"] != "1200 TRY" { // nested text flattened
		t.Fatalf("price = %q", rows[0]["price"])
	}
	if rows[1]["extra"] != "only here" || rows[0]["extra"] != "" {
		t.Fatalf("extra column wrong: %v", rows)
	}

	if _, rows, _ := xmlItemsToRows(strings.NewReader(feed), "product"); len(rows) != 0 {
		t.Fatalf("wrong items_path should yield 0 rows, got %d", len(rows))
	}
}
