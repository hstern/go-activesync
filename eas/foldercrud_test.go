// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"context"
	"net/http"
	"testing"

	"github.com/hstern/go-activesync/wbxml"
)

func TestFolderCreate_returnsServerID(t *testing.T) {
	c, cap, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageFolderHierarchy, "FolderCreate",
				wbxml.E(wbxml.PageFolderHierarchy, "Status", wbxml.Text("1")),
				wbxml.E(wbxml.PageFolderHierarchy, "SyncKey", wbxml.Text("FS-2")),
				wbxml.E(wbxml.PageFolderHierarchy, "ServerId", wbxml.Text("new-folder")),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	_ = c.cfg.State.SetSyncKey(context.Background(), FolderRootID, "FS-1")

	res, err := c.FolderCreate(context.Background(), "0", "Projects", FolderTypeUserGeneric)
	if err != nil {
		t.Fatal(err)
	}
	if res.ServerID != "new-folder" || res.SyncKey != "FS-2" {
		t.Errorf("res = %+v", res)
	}
	// Persisted.
	stored, _ := c.cfg.State.SyncKey(context.Background(), FolderRootID)
	if stored != "FS-2" {
		t.Errorf("sync key not persisted: %q", stored)
	}
	// Request includes DisplayName + Type.
	req, _ := wbxml.Unmarshal(cap.body, wbxml.DefaultRegistry())
	if dn := req.Root.Find("DisplayName"); dn == nil || dn.TextContent() != "Projects" {
		t.Errorf("DisplayName: %v", dn)
	}
}

func TestFolderUpdate_persistsKey(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageFolderHierarchy, "FolderUpdate",
				wbxml.E(wbxml.PageFolderHierarchy, "Status", wbxml.Text("1")),
				wbxml.E(wbxml.PageFolderHierarchy, "SyncKey", wbxml.Text("FS-3")),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	_ = c.cfg.State.SetSyncKey(context.Background(), FolderRootID, "FS-2")
	if err := c.FolderUpdate(context.Background(), "f1", "0", "Renamed"); err != nil {
		t.Fatal(err)
	}
	stored, _ := c.cfg.State.SyncKey(context.Background(), FolderRootID)
	if stored != "FS-3" {
		t.Errorf("stored = %q", stored)
	}
}

func TestFolderDelete_returnsErrorOnBadStatus(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageFolderHierarchy, "FolderDelete",
				wbxml.E(wbxml.PageFolderHierarchy, "Status", wbxml.Text("3")),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	_ = c.cfg.State.SetSyncKey(context.Background(), FolderRootID, "x")
	err := c.FolderDelete(context.Background(), "f1")
	if !IsStatusCode(err, 3) {
		t.Errorf("err = %v", err)
	}
}
