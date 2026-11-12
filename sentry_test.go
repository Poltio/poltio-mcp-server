package main

import (
	"strconv"
	"testing"
	"time"
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
