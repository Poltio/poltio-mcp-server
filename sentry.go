package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// initSentry reports to SENTRY_DSN; without it every capture is a no-op.
func initSentry(release string) error {
	t := &throttle{}
	return sentry.Init(sentry.ClientOptions{
		Dsn:     os.Getenv("SENTRY_DSN"),
		Release: release,
		// No IPs, cookies, headers, bodies or query strings: they can carry lead PII (Poltio/issues#6483).
		DataCollection: &sentry.DataCollection{
			UserInfo: sentry.Set(false),
			Cookies:  &sentry.KeyValueCollectionBehavior{Mode: sentry.CollectionOff},
			HTTPHeaders: &sentry.HeaderCollectionConfig{
				Request:  &sentry.KeyValueCollectionBehavior{Mode: sentry.CollectionOff},
				Response: &sentry.KeyValueCollectionBehavior{Mode: sentry.CollectionOff},
			},
			HTTPBodies:  []sentry.BodyType{},
			QueryParams: &sentry.KeyValueCollectionBehavior{Mode: sentry.CollectionOff},
		},
		BeforeSend: func(e *sentry.Event, h *sentry.EventHint) *sentry.Event {
			// A client hanging up, or a shutdown, is not a bug.
			if h != nil && errors.Is(h.OriginalException, context.Canceled) {
				return nil
			}
			if !t.allow(eventKey(e), time.Now()) {
				return nil
			}
			return e
		},
	})
}

// maxEventsPerMinute caps what one pod sends, so an outage that fails every
// request costs a handful of events instead of the quota.
const maxEventsPerMinute = 20

// throttle lets each distinct event through once a minute, and at most
// maxEventsPerMinute in total.
type throttle struct {
	mu     sync.Mutex
	window time.Time
	sent   map[string]bool
}

func (t *throttle) allow(key string, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if now.Sub(t.window) >= time.Minute {
		t.window, t.sent = now, map[string]bool{}
	}
	if t.sent[key] || len(t.sent) >= maxEventsPerMinute {
		return false
	}
	t.sent[key] = true
	return true
}

// keyNoise is the part of an error's text that changes on every request (URLs,
// object names, ids, ports), so one outage stays one key.
var keyNoise = regexp.MustCompile(`\S+://\S+|\d+`)

func eventKey(e *sentry.Event) string {
	key := e.Message
	if key == "" && len(e.Exception) > 0 {
		x := e.Exception[len(e.Exception)-1]
		key = x.Type + ": " + x.Value
	}
	return keyNoise.ReplaceAllString(key, "#")
}

// reportPanics turns a panicking tool into an error result and a Sentry event.
func reportPanics(next server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (res *mcp.CallToolResult, err error) {
		defer func() {
			if p := recover(); p != nil {
				hub := sentry.CurrentHub().Clone()
				hub.Scope().SetTag("tool", req.Params.Name)
				hub.Recover(p)
				err = fmt.Errorf("panic in %s tool: %v", req.Params.Name, p)
			}
		}()
		return next(ctx, req)
	}
}
