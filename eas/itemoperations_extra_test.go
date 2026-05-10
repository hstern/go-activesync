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

func TestFetchDocumentLibrary_emptyLink(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("server should not be hit")
	})
	if _, err := c.FetchDocumentLibrary(context.Background(), "", 0, 0); err == nil {
		t.Error("want error for empty linkID")
	}
}

func TestFetchDocumentLibrary_serverError(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{Root: wbxml.E(wbxml.PageItemOperations, "ItemOperations",
			wbxml.E(wbxml.PageItemOperations, "Status", wbxml.Text("3")),
		)}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	if _, err := c.FetchDocumentLibrary(context.Background(), "link-1", 0, 0); err == nil {
		t.Error("want error for non-OK status")
	}
}

func TestFetchDocumentLibrary_withRange(t *testing.T) {
	c, cap, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{Root: wbxml.E(wbxml.PageItemOperations, "ItemOperations",
			wbxml.E(wbxml.PageItemOperations, "Status", wbxml.Text("1")),
			wbxml.E(wbxml.PageItemOperations, "Response",
				wbxml.E(wbxml.PageItemOperations, "Fetch",
					wbxml.E(wbxml.PageItemOperations, "Status", wbxml.Text("1")),
					wbxml.E(wbxml.PageItemOperations, "Properties",
						wbxml.E(wbxml.PageItemOperations, "Data", wbxml.Opaque([]byte("partial"))),
					),
				),
			),
		)}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	got, err := c.FetchDocumentLibrary(context.Background(), "link-1", 0, 1023)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte("partial")) {
		t.Errorf("data = %q", got)
	}
	req, _ := wbxml.Unmarshal(cap.body, wbxml.DefaultRegistry())
	if rng := req.Root.Find("Range"); rng == nil || rng.TextContent() != "0-1023" {
		t.Errorf("Range = %v", rng)
	}
}

func TestFetchAttachment_returnsBytes(t *testing.T) {
	want := []byte("attachment payload")
	c, cap, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageItemOperations, "ItemOperations",
				wbxml.E(wbxml.PageItemOperations, "Status", wbxml.Text("1")),
				wbxml.E(wbxml.PageItemOperations, "Response",
					wbxml.E(wbxml.PageItemOperations, "Fetch",
						wbxml.E(wbxml.PageItemOperations, "Status", wbxml.Text("1")),
						wbxml.E(wbxml.PageItemOperations, "Properties",
							wbxml.E(wbxml.PageAirSyncBase, "ContentType", wbxml.Text("application/pdf")),
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
	res, err := c.FetchAttachment(context.Background(), "ref-1:0:1234", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(res.Data, want) {
		t.Errorf("Data = %q want %q", res.Data, want)
	}
	if res.ContentType != "application/pdf" {
		t.Errorf("ContentType = %q", res.ContentType)
	}
	req, _ := wbxml.Unmarshal(cap.body, wbxml.DefaultRegistry())
	if fr := req.Root.Find("FileReference"); fr == nil || fr.TextContent() != "ref-1:0:1234" {
		t.Errorf("FileReference = %v", fr)
	}
}

func TestFetchAttachment_withRange(t *testing.T) {
	c, cap, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageItemOperations, "ItemOperations",
				wbxml.E(wbxml.PageItemOperations, "Status", wbxml.Text("1")),
				wbxml.E(wbxml.PageItemOperations, "Response",
					wbxml.E(wbxml.PageItemOperations, "Fetch",
						wbxml.E(wbxml.PageItemOperations, "Status", wbxml.Text("1")),
						wbxml.E(wbxml.PageItemOperations, "Range", wbxml.Text("0-1023")),
						wbxml.E(wbxml.PageItemOperations, "Properties",
							wbxml.E(wbxml.PageItemOperations, "Data", wbxml.Opaque([]byte("partial"))),
						),
					),
				),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	res, err := c.FetchAttachment(context.Background(), "ref", 0, 1023)
	if err != nil {
		t.Fatal(err)
	}
	if res.Range != "0-1023" {
		t.Errorf("Range = %q", res.Range)
	}
	req, _ := wbxml.Unmarshal(cap.body, wbxml.DefaultRegistry())
	if rng := req.Root.Find("Range"); rng == nil || rng.TextContent() != "0-1023" {
		t.Errorf("Range request = %v", rng)
	}
}

func TestFetchAttachment_emptyReference(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("server should not be hit")
	})
	if _, err := c.FetchAttachment(context.Background(), "", 0, 0); err == nil {
		t.Error("want error for empty FileReference")
	}
}

func TestFetchAttachment_serverError(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageItemOperations, "ItemOperations",
				wbxml.E(wbxml.PageItemOperations, "Status", wbxml.Text("3")),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	if _, err := c.FetchAttachment(context.Background(), "ref", 0, 0); err == nil {
		t.Error("want error for non-OK status")
	}
}

func TestFetchAttachment_missingData(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageItemOperations, "ItemOperations",
				wbxml.E(wbxml.PageItemOperations, "Status", wbxml.Text("1")),
				wbxml.E(wbxml.PageItemOperations, "Response",
					wbxml.E(wbxml.PageItemOperations, "Fetch",
						wbxml.E(wbxml.PageItemOperations, "Status", wbxml.Text("1")),
						wbxml.E(wbxml.PageItemOperations, "Properties"),
					),
				),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	if _, err := c.FetchAttachment(context.Background(), "ref", 0, 0); err == nil {
		t.Error("want error when Data missing")
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
