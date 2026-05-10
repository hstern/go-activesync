// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hstern/go-activesync/wbxml"
)

func searchResponse(total int, hits ...*wbxml.Element) []byte {
	store := wbxml.E(wbxml.PageSearch, "Store",
		wbxml.E(wbxml.PageSearch, "Status", wbxml.Text("1")),
	)
	for _, h := range hits {
		store.Children = append(store.Children, h)
	}
	store.Children = append(store.Children,
		wbxml.E(wbxml.PageSearch, "Range", wbxml.Text("0-49")),
		wbxml.E(wbxml.PageSearch, "Total", wbxml.Text(itoa(total))),
	)
	doc := &wbxml.Document{
		Root: wbxml.E(wbxml.PageSearch, "Search",
			wbxml.E(wbxml.PageSearch, "Status", wbxml.Text("1")),
			wbxml.E(wbxml.PageSearch, "Response", store),
		),
	}
	out, err := wbxml.Marshal(doc, wbxml.DefaultRegistry())
	if err != nil {
		panic(err)
	}
	return out
}

func searchHit(longID, subject, from string) *wbxml.Element {
	return wbxml.E(wbxml.PageSearch, "Result",
		wbxml.E(wbxml.PageAirSync, "Class", wbxml.Text("Email")),
		wbxml.E(wbxml.PageSearch, "LongId", wbxml.Text(longID)),
		wbxml.E(wbxml.PageAirSync, "CollectionId", wbxml.Text("inbox")),
		wbxml.E(wbxml.PageSearch, "Properties",
			wbxml.E(wbxml.PageEmail, "Subject", wbxml.Text(subject)),
			wbxml.E(wbxml.PageEmail, "From", wbxml.Text(from)),
		),
	)
}

func TestSearchEmail_basic(t *testing.T) {
	c, cap, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(searchResponse(2,
			searchHit("hit-1", "Quarterly numbers", "alice@x"),
			searchHit("hit-2", "Q numbers reply", "bob@x"),
		))
	})

	res, err := c.SearchEmail(context.Background(), "quarterly", EmailSearchOptions{
		FolderID: "inbox",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 2 {
		t.Errorf("Total = %d", res.Total)
	}
	if res.Range != "0-49" {
		t.Errorf("Range = %q", res.Range)
	}
	if len(res.Items) != 2 {
		t.Fatalf("len(Items) = %d", len(res.Items))
	}
	if res.Items[0].Subject != "Quarterly numbers" || res.Items[0].ServerID != "hit-1" {
		t.Errorf("hit 0 = %+v", res.Items[0])
	}

	// Verify request shape: Cmd=Search, contains FreeText "quarterly".
	if cap.url.Query().Get("Cmd") != "Search" {
		t.Errorf("Cmd: %q", cap.url.Query().Get("Cmd"))
	}
	req, _ := wbxml.Unmarshal(cap.body, wbxml.DefaultRegistry())
	if ft := req.Root.Find("FreeText"); ft == nil || ft.TextContent() != "quarterly" {
		t.Errorf("FreeText = %v", ft)
	}
	if cid := req.Root.Find("CollectionId"); cid == nil || cid.TextContent() != "inbox" {
		t.Errorf("CollectionId = %v", cid)
	}
}

func TestSearchEmail_emptyQuery(t *testing.T) {
	c, _, _ := newTestClient(t, nil)
	_, err := c.SearchEmail(context.Background(), "  ", EmailSearchOptions{})
	if err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Errorf("err = %v", err)
	}
}

func TestSearchEmail_serverError(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageSearch, "Search",
				wbxml.E(wbxml.PageSearch, "Status", wbxml.Text("1")),
				wbxml.E(wbxml.PageSearch, "Response",
					wbxml.E(wbxml.PageSearch, "Store",
						wbxml.E(wbxml.PageSearch, "Status", wbxml.Text("3")),
						wbxml.E(wbxml.PageSearch, "Range", wbxml.Text("0-0")),
						wbxml.E(wbxml.PageSearch, "Total", wbxml.Text("0")),
					),
				),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	_, err := c.SearchEmail(context.Background(), "x", EmailSearchOptions{})
	if !IsStatusCode(err, 3) {
		t.Errorf("err = %v", err)
	}
}

func TestSearchEmailQuery_structuredAnd(t *testing.T) {
	c, cap, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(searchResponse(0))
	})
	q := And(
		EmailClass(),
		EqualTo(PropEmailFrom, "alice@x"),
		GreaterThan(PropEmailDateReceived, "2026-01-01T00:00:00.000Z"),
	)
	if _, err := c.SearchEmailQuery(context.Background(), q, EmailSearchOptions{}); err != nil {
		t.Fatal(err)
	}
	req, _ := wbxml.Unmarshal(cap.body, wbxml.DefaultRegistry())
	if req.Root.Find("And") == nil {
		t.Error("And missing")
	}
	if req.Root.Find("EqualTo") == nil {
		t.Error("EqualTo missing")
	}
	if req.Root.Find("GreaterThan") == nil {
		t.Error("GreaterThan missing")
	}
}

func TestSearchEmailQuery_nilRejected(t *testing.T) {
	c, _, _ := newTestClient(t, nil)
	if _, err := c.SearchEmailQuery(context.Background(), nil, EmailSearchOptions{}); err == nil {
		t.Error("want error for nil query")
	}
}

func TestSearchEmail_emptyResults(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(searchResponse(0))
	})
	res, err := c.SearchEmail(context.Background(), "x", EmailSearchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 0 || len(res.Items) != 0 {
		t.Errorf("res = %+v", res)
	}
}
