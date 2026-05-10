// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/hstern/go-activesync/wbxml"
)

// pimSyncResponse builds a Sync response for any class with the given
// adds (each Add already in WBXML form).
func pimSyncResponse(folderID, syncKey string, adds ...*wbxml.Element) []byte {
	collection := wbxml.E(wbxml.PageAirSync, "Collection",
		wbxml.E(wbxml.PageAirSync, "SyncKey", wbxml.Text(syncKey)),
		wbxml.E(wbxml.PageAirSync, "CollectionId", wbxml.Text(folderID)),
		wbxml.E(wbxml.PageAirSync, "Status", wbxml.Text("1")),
	)
	if len(adds) > 0 {
		commands := wbxml.E(wbxml.PageAirSync, "Commands")
		for _, a := range adds {
			commands.Children = append(commands.Children, a)
		}
		collection.Children = append(collection.Children, commands)
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

func TestSyncContacts_parsesItem(t *testing.T) {
	add := wbxml.E(wbxml.PageAirSync, "Add",
		wbxml.E(wbxml.PageAirSync, "ServerId", wbxml.Text("c-1")),
		wbxml.E(wbxml.PageAirSync, "ApplicationData",
			wbxml.E(wbxml.PageContacts, "FirstName", wbxml.Text("Alice")),
			wbxml.E(wbxml.PageContacts, "LastName", wbxml.Text("Engineer")),
			wbxml.E(wbxml.PageContacts, "Email1Address", wbxml.Text("alice@example.com")),
			wbxml.E(wbxml.PageContacts, "MobilePhoneNumber", wbxml.Text("+1-555-0100")),
		),
	)
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(pimSyncResponse("contacts", "C1", add))
	})
	res, err := c.SyncContacts(context.Background(), "contacts")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 1 {
		t.Fatalf("len = %d", len(res.Added))
	}
	if res.Added[0].FirstName != "Alice" || res.Added[0].Email1Address != "alice@example.com" {
		t.Errorf("got %+v", res.Added[0])
	}
}

func TestSyncTasks_parsesItem(t *testing.T) {
	add := wbxml.E(wbxml.PageAirSync, "Add",
		wbxml.E(wbxml.PageAirSync, "ServerId", wbxml.Text("t-1")),
		wbxml.E(wbxml.PageAirSync, "ApplicationData",
			wbxml.E(wbxml.PageTasks, "Subject", wbxml.Text("Write tests")),
			wbxml.E(wbxml.PageTasks, "Importance", wbxml.Text("2")),
			wbxml.E(wbxml.PageTasks, "Complete", wbxml.Text("0")),
			wbxml.E(wbxml.PageTasks, "DueDate", wbxml.Text("2026-05-15T17:00:00.000Z")),
		),
	)
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(pimSyncResponse("tasks", "T1", add))
	})
	res, err := c.SyncTasks(context.Background(), "tasks")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 1 {
		t.Fatalf("len = %d", len(res.Added))
	}
	tk := res.Added[0]
	if tk.Subject != "Write tests" || tk.Importance != 2 || tk.Complete {
		t.Errorf("got %+v", tk)
	}
	want := time.Date(2026, 5, 15, 17, 0, 0, 0, time.UTC)
	if !tk.DueDate.Equal(want) {
		t.Errorf("DueDate = %v want %v", tk.DueDate, want)
	}
}

func TestSyncNotes_parsesItem(t *testing.T) {
	add := wbxml.E(wbxml.PageAirSync, "Add",
		wbxml.E(wbxml.PageAirSync, "ServerId", wbxml.Text("n-1")),
		wbxml.E(wbxml.PageAirSync, "ApplicationData",
			wbxml.E(wbxml.PageNotes, "Subject", wbxml.Text("Quick note")),
			wbxml.E(wbxml.PageAirSyncBase, "Body",
				wbxml.E(wbxml.PageAirSyncBase, "Type", wbxml.Text("1")),
				wbxml.E(wbxml.PageAirSyncBase, "Data", wbxml.Text("body text")),
			),
			wbxml.E(wbxml.PageNotes, "Categories",
				wbxml.E(wbxml.PageNotes, "Category", wbxml.Text("work")),
				wbxml.E(wbxml.PageNotes, "Category", wbxml.Text("urgent")),
			),
		),
	)
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(pimSyncResponse("notes", "N1", add))
	})
	res, err := c.SyncNotes(context.Background(), "notes")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 1 {
		t.Fatalf("len = %d", len(res.Added))
	}
	n := res.Added[0]
	if n.Subject != "Quick note" || n.Body != "body text" {
		t.Errorf("note = %+v", n)
	}
	if len(n.Categories) != 2 || n.Categories[0] != "work" {
		t.Errorf("categories = %v", n.Categories)
	}
}

func TestGALSearch_parsesEntries(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageSearch, "Search",
				wbxml.E(wbxml.PageSearch, "Status", wbxml.Text("1")),
				wbxml.E(wbxml.PageSearch, "Response",
					wbxml.E(wbxml.PageSearch, "Store",
						wbxml.E(wbxml.PageSearch, "Status", wbxml.Text("1")),
						wbxml.E(wbxml.PageSearch, "Result",
							wbxml.E(wbxml.PageSearch, "Properties",
								wbxml.E(wbxml.PageGAL, "DisplayName", wbxml.Text("Alice Engineer")),
								wbxml.E(wbxml.PageGAL, "EmailAddress", wbxml.Text("alice@corp.com")),
								wbxml.E(wbxml.PageGAL, "Title", wbxml.Text("Staff SWE")),
							),
						),
						wbxml.E(wbxml.PageSearch, "Range", wbxml.Text("0-0")),
						wbxml.E(wbxml.PageSearch, "Total", wbxml.Text("1")),
					),
				),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	res, err := c.GALSearch(context.Background(), "alice", 10)
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || len(res.Entries) != 1 {
		t.Errorf("res = %+v", res)
	}
	if res.Entries[0].EmailAddress != "alice@corp.com" {
		t.Errorf("entry = %+v", res.Entries[0])
	}
}

func TestCreateContact_returnsServerID(t *testing.T) {
	var (
		calls int
	)
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := readCapped(r.Body, 1<<20)
		req, _ := wbxml.Unmarshal(body, wbxml.DefaultRegistry())
		calls++
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		switch calls {
		case 1:
			// bootstrap
			w.Write(pimSyncResponse("contacts", "BOOT"))
		case 2:
			cid := req.Root.Find("ClientId").TextContent()
			doc := &wbxml.Document{
				Root: wbxml.E(wbxml.PageAirSync, "Sync",
					wbxml.E(wbxml.PageAirSync, "Collections",
						wbxml.E(wbxml.PageAirSync, "Collection",
							wbxml.E(wbxml.PageAirSync, "SyncKey", wbxml.Text("DONE")),
							wbxml.E(wbxml.PageAirSync, "CollectionId", wbxml.Text("contacts")),
							wbxml.E(wbxml.PageAirSync, "Status", wbxml.Text("1")),
							wbxml.E(wbxml.PageAirSync, "Responses",
								wbxml.E(wbxml.PageAirSync, "Add",
									wbxml.E(wbxml.PageAirSync, "ClientId", wbxml.Text(cid)),
									wbxml.E(wbxml.PageAirSync, "ServerId", wbxml.Text("contacts:NEW")),
									wbxml.E(wbxml.PageAirSync, "Status", wbxml.Text("1")),
								),
							),
						),
					),
				),
			}
			b, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
			w.Write(b)
		}
	})
	id, err := c.CreateContact(context.Background(), "contacts", ContactDraft{
		FirstName: "Alice", LastName: "Engineer",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "contacts:NEW" {
		t.Errorf("id = %q", id)
	}
	if calls != 2 {
		t.Errorf("calls = %d", calls)
	}
}

func TestCompleteTask_setsCompleteFlag(t *testing.T) {
	var lastBody []byte
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := readCapped(r.Body, 1<<20)
		lastBody = body
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(pimSyncResponse("tasks", "DONE"))
	})
	_ = c.cfg.State.SetSyncKey(context.Background(), "tasks", "PRIOR")
	if err := c.CompleteTask(context.Background(), "tasks", "t-1"); err != nil {
		t.Fatal(err)
	}
	req, _ := wbxml.Unmarshal(lastBody, wbxml.DefaultRegistry())
	complete := req.Root.Find("Complete")
	if complete == nil || complete.TextContent() != "1" {
		t.Errorf("Complete = %v", complete)
	}
	if req.Root.Find("DateCompleted") == nil {
		t.Error("DateCompleted missing")
	}
}
