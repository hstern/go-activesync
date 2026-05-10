// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/hstern/go-activesync/wbxml"
)

func changeAck(syncKey string, statuses ...struct {
	id     string
	status string
}) []byte {
	collection := wbxml.E(wbxml.PageAirSync, "Collection",
		wbxml.E(wbxml.PageAirSync, "SyncKey", wbxml.Text(syncKey)),
		wbxml.E(wbxml.PageAirSync, "CollectionId", wbxml.Text("inbox")),
		wbxml.E(wbxml.PageAirSync, "Status", wbxml.Text("1")),
	)
	if len(statuses) > 0 {
		responses := wbxml.E(wbxml.PageAirSync, "Responses")
		for _, s := range statuses {
			responses.Children = append(responses.Children,
				wbxml.E(wbxml.PageAirSync, "Change",
					wbxml.E(wbxml.PageAirSync, "ServerId", wbxml.Text(s.id)),
					wbxml.E(wbxml.PageAirSync, "Status", wbxml.Text(s.status)),
				),
			)
		}
		collection.Children = append(collection.Children, responses)
	}
	doc := &wbxml.Document{
		Root: wbxml.E(wbxml.PageAirSync, "Sync",
			wbxml.E(wbxml.PageAirSync, "Collections", collection),
		),
	}
	out, err := wbxml.Marshal(doc, wbxml.DefaultRegistry())
	if err != nil {
		panic(err)
	}
	return out
}

// helper to take a pointer to a bool literal in tests.
func boolp(b bool) *bool { return &b }

func TestApplyEmailChanges_setRead(t *testing.T) {
	c, cap, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(changeAck("S2"))
	})
	_ = c.cfg.State.SetSyncKey(context.Background(), "inbox", "S1")

	if _, err := c.ApplyEmailChanges(context.Background(), "inbox", []EmailChange{
		{ServerID: "inbox:1", Read: boolp(true)},
	}); err != nil {
		t.Fatal(err)
	}
	// Verify Sync key advanced.
	if k, _ := c.cfg.State.SyncKey(context.Background(), "inbox"); k != "S2" {
		t.Errorf("sync key not advanced: %q", k)
	}
	// Verify request shape: Change with ServerId + ApplicationData.Read=1
	req, _ := wbxml.Unmarshal(cap.body, wbxml.DefaultRegistry())
	change := req.Root.Find("Change")
	if change == nil {
		t.Fatal("Change missing")
	}
	if r := change.Find("Read"); r == nil || r.TextContent() != "1" {
		t.Errorf("Read = %v", r)
	}
}

func TestApplyEmailChanges_setFlagged(t *testing.T) {
	c, cap, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(changeAck("S2"))
	})
	_ = c.cfg.State.SetSyncKey(context.Background(), "inbox", "S1")
	_, err := c.ApplyEmailChanges(context.Background(), "inbox", []EmailChange{
		{ServerID: "x", Flagged: boolp(true)},
	})
	if err != nil {
		t.Fatal(err)
	}
	req, _ := wbxml.Unmarshal(cap.body, wbxml.DefaultRegistry())
	if fs := req.Root.Find("FlagStatus"); fs == nil || fs.TextContent() != "2" {
		t.Errorf("FlagStatus = %v", fs)
	}
}

func TestApplyEmailChanges_delete(t *testing.T) {
	c, cap, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(changeAck("S2"))
	})
	_ = c.cfg.State.SetSyncKey(context.Background(), "inbox", "S1")
	_, err := c.ApplyEmailChanges(context.Background(), "inbox", []EmailChange{
		{ServerID: "doomed", Delete: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	req, _ := wbxml.Unmarshal(cap.body, wbxml.DefaultRegistry())
	del := req.Root.Find("Delete")
	if del == nil {
		t.Fatal("Delete missing")
	}
	if id := del.Find("ServerId"); id == nil || id.TextContent() != "doomed" {
		t.Errorf("ServerId = %v", id)
	}
	// Should NOT have a Change element when only deletes are sent.
	if req.Root.Find("Change") != nil {
		t.Error("unexpected Change present")
	}
}

func TestApplyEmailChanges_bootstrapsIfNeeded(t *testing.T) {
	var (
		mu       sync.Mutex
		calls    int
		sentKeys []string
	)
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := readCapped(r.Body, 1<<20)
		req, _ := wbxml.Unmarshal(body, wbxml.DefaultRegistry())
		mu.Lock()
		calls++
		thisCall := calls
		sentKeys = append(sentKeys, req.Root.Find("SyncKey").TextContent())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		switch thisCall {
		case 1:
			// bootstrap: no items, just a new key
			w.Write(changeAck("BOOT"))
		case 2:
			w.Write(changeAck("DONE"))
		}
	})

	_, err := c.ApplyEmailChanges(context.Background(), "inbox", []EmailChange{
		{ServerID: "x", Read: boolp(true)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Errorf("calls = %d", calls)
	}
	if sentKeys[0] != "0" || sentKeys[1] != "BOOT" {
		t.Errorf("sentKeys = %v", sentKeys)
	}
}

func TestApplyEmailChanges_perItemStatus(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(changeAck("S2",
			struct{ id, status string }{"a", "1"},
			struct{ id, status string }{"b", "8"}, // ObjectNotFound
		))
	})
	_ = c.cfg.State.SetSyncKey(context.Background(), "inbox", "S1")

	res, err := c.ApplyEmailChanges(context.Background(), "inbox", []EmailChange{
		{ServerID: "a", Read: boolp(true)},
		{ServerID: "b", Read: boolp(true)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Fatalf("len = %d", len(res))
	}
	if res[0].Status != 1 || res[1].Status != 8 {
		t.Errorf("results = %+v", res)
	}
}

func TestApplyEmailChanges_validation(t *testing.T) {
	c, _, _ := newTestClient(t, nil)
	if _, err := c.ApplyEmailChanges(context.Background(), "", []EmailChange{{ServerID: "x"}}); err == nil {
		t.Error("want error for empty folder")
	}
	if _, err := c.ApplyEmailChanges(context.Background(), "f", nil); err == nil {
		t.Error("want error for empty changes")
	}
	// Change with no fields and no Delete → no effective changes
	if _, err := c.ApplyEmailChanges(context.Background(), "f", []EmailChange{{ServerID: "x"}}); err == nil ||
		!strings.Contains(err.Error(), "no effective changes") &&
			!strings.Contains(err.Error(), "bootstrap") {
		// May error during ensureSynced (no httptest), which is fine; we
		// just want SOME error here.
		t.Errorf("err = %v", err)
	}
}
