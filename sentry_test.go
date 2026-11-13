package main

import (
	"strconv"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

func TestThrottle(t *testing.T) {
	var th throttle
	now := time.Now()
	if !th.allow("a", now) || th.allow("a", now.Add(30*time.Second)) {
		t.Fatal("same key twice in a minute: want first only")
	}
	if !th.allow("a", now.Add(time.Minute)) {
		t.Fatal("same key in the next minute: want allowed")
	}
	for i := 1; i < maxEventsPerMinute; i++ {
		if !th.allow(strconv.Itoa(i), now.Add(time.Minute)) {
			t.Fatalf("key %d under the cap: want allowed", i)
		}
	}
	if th.allow("over", now.Add(time.Minute)) {
		t.Fatal("key over the cap: want dropped")
	}
}

func TestEventKeyIgnoresPerRequestText(t *testing.T) {
	gcs := func(op, obj, port string) *sentry.Event {
		return &sentry.Event{Exception: []sentry.Exception{{Type: "*fmt.wrapError", Value: op + `: Get "http://127.0.0.1:1/storage/v1/b/poltio/o/img%2F` + obj + `?alt=media": dial tcp 10.0.0.1:` + port + `: connect: connection refused`}}}
	}
	if eventKey(gcs("gcs read", "494a30704d4f", "443")) != eventKey(gcs("gcs read", "ee5abc9f0d21", "8443")) {
		t.Error("same failure on two objects: want one key")
	}
	if eventKey(gcs("gcs read", "494a30704d4f", "443")) == eventKey(gcs("gcs write", "494a30704d4f", "443")) {
		t.Error("read and write failures: want two keys")
	}
}
