// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/hstern/go-activesync/wbxml"
)

func TestEmptyFolderContents(t *testing.T) {
	c, cap, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageItemOperations, "ItemOperations",
				wbxml.E(wbxml.PageItemOperations, "Status", wbxml.Text("1")),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	if err := c.EmptyFolderContents(context.Background(), "trash", true); err != nil {
		t.Fatal(err)
	}
	req, _ := wbxml.Unmarshal(cap.body, wbxml.DefaultRegistry())
	if req.Root.Find("EmptyFolderContents") == nil {
		t.Error("EmptyFolderContents missing")
	}
	if ds := req.Root.Find("DeleteSubFolders"); ds == nil || ds.TextContent() != "1" {
		t.Errorf("DeleteSubFolders = %v", ds)
	}
}

func TestFetchDocumentLibrary(t *testing.T) {
	want := []byte("file contents here")
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageItemOperations, "ItemOperations",
				wbxml.E(wbxml.PageItemOperations, "Status", wbxml.Text("1")),
				wbxml.E(wbxml.PageItemOperations, "Response",
					wbxml.E(wbxml.PageItemOperations, "Fetch",
						wbxml.E(wbxml.PageItemOperations, "Status", wbxml.Text("1")),
						wbxml.E(wbxml.PageItemOperations, "Properties",
							wbxml.E(wbxml.PageItemOperations, "Data", wbxml.Opaque(want)),
						),
					),
				),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	got, err := c.FetchDocumentLibrary(context.Background(), "//share/file.txt", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMoveViaItemOperations(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageItemOperations, "ItemOperations",
				wbxml.E(wbxml.PageItemOperations, "Status", wbxml.Text("1")),
				wbxml.E(wbxml.PageItemOperations, "Response",
					wbxml.E(wbxml.PageItemOperations, "Move",
						wbxml.E(wbxml.PageAirSync, "ServerId", wbxml.Text("dst:42")),
						wbxml.E(wbxml.PageItemOperations, "Status", wbxml.Text("1")),
					),
				),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	id, err := c.MoveViaItemOperations(context.Background(), "src", "src:1", "dst", true)
	if err != nil {
		t.Fatal(err)
	}
	if id != "dst:42" {
		t.Errorf("id = %q", id)
	}
}

func TestItoa64(t *testing.T) {
	cases := map[int64]string{
		0: "0", 1: "1", 9: "9", 10: "10", 100: "100",
		-1: "-1", -42: "-42",
		1 << 32: "4294967296",
	}
	for in, want := range cases {
		if got := itoa64(in); got != want {
			t.Errorf("itoa64(%d) = %q, want %q", in, got, want)
		}
	}
}
