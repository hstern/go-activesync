// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hstern/go-activesync/wbxml"
)

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

func TestCreateContact_returnsServerID(t *testing.T) {
	c, bodyP := twoCallSyncServer(t, "contacts", "BOOT", "contacts:NEW")
	id, err := c.CreateContact(context.Background(), "contacts", ContactDraft{
		FirstName: "Alice", LastName: "Engineer",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "contacts:NEW" {
		t.Errorf("id = %q", id)
	}
	if !strings.Contains(string(**bodyP), "Engineer") {
		t.Error("body missing last name")
	}
}

func TestUpdateContact_emitsChange(t *testing.T) {
	c, lastBody := singleCallSyncServer(t, "contacts")
	if err := c.UpdateContact(context.Background(), "contacts", "c-1", ContactDraft{
		FirstName: "Alice", LastName: "Updated",
	}); err != nil {
		t.Fatal(err)
	}
	req, _ := wbxml.Unmarshal(*lastBody, wbxml.DefaultRegistry())
	if req.Root.Find("Change") == nil {
		t.Error("request missing <Change>")
	}
	if !strings.Contains(string(*lastBody), "Updated") {
		t.Error("body missing new last name")
	}
}

func TestDeleteContact_emitsDelete(t *testing.T) {
	c, lastBody := singleCallSyncServer(t, "contacts")
	if err := c.DeleteContact(context.Background(), "contacts", "c-1"); err != nil {
		t.Fatal(err)
	}
	req, _ := wbxml.Unmarshal(*lastBody, wbxml.DefaultRegistry())
	if req.Root.Find("Delete") == nil {
		t.Error("request missing <Delete>")
	}
}

func TestParseContactItem_allFields(t *testing.T) {
	// Picture: build a base64 payload the parser will decode.
	pic := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46} // JPEG-ish prefix
	picB64 := "/9j/4AAQSkY=" // base64 of pic
	app := wbxml.E(wbxml.PageAirSync, "ApplicationData",
		wbxml.E(wbxml.PageContacts, "FirstName", wbxml.Text("Alice")),
		wbxml.E(wbxml.PageContacts, "LastName", wbxml.Text("Engineer")),
		wbxml.E(wbxml.PageContacts, "MiddleName", wbxml.Text("Q")),
		wbxml.E(wbxml.PageContacts, "Title", wbxml.Text("Dr.")),
		wbxml.E(wbxml.PageContacts, "Suffix", wbxml.Text("PhD")),
		wbxml.E(wbxml.PageContacts, "FileAs", wbxml.Text("Engineer, Alice")),
		wbxml.E(wbxml.PageContacts, "CompanyName", wbxml.Text("Acme")),
		wbxml.E(wbxml.PageContacts, "Department", wbxml.Text("R&D")),
		wbxml.E(wbxml.PageContacts, "JobTitle", wbxml.Text("Staff SWE")),
		wbxml.E(wbxml.PageContacts, "OfficeLocation", wbxml.Text("HQ-3")),
		wbxml.E(wbxml.PageContacts, "Email1Address", wbxml.Text("alice@x")),
		wbxml.E(wbxml.PageContacts, "Email2Address", wbxml.Text("a.engineer@personal")),
		wbxml.E(wbxml.PageContacts, "Email3Address", wbxml.Text("a@old")),
		wbxml.E(wbxml.PageContacts, "HomePhoneNumber", wbxml.Text("+1-555-0100")),
		wbxml.E(wbxml.PageContacts, "BusinessPhoneNumber", wbxml.Text("+1-555-0200")),
		wbxml.E(wbxml.PageContacts, "MobilePhoneNumber", wbxml.Text("+1-555-0300")),
		wbxml.E(wbxml.PageContacts, "HomeAddressStreet", wbxml.Text("1 Home St")),
		wbxml.E(wbxml.PageContacts, "HomeAddressCity", wbxml.Text("Hometown")),
		wbxml.E(wbxml.PageContacts, "HomeAddressState", wbxml.Text("HS")),
		wbxml.E(wbxml.PageContacts, "HomeAddressPostalCode", wbxml.Text("H0H 0H0")),
		wbxml.E(wbxml.PageContacts, "HomeAddressCountry", wbxml.Text("Canada")),
		wbxml.E(wbxml.PageContacts, "BusinessAddressStreet", wbxml.Text("9 Office Blvd")),
		wbxml.E(wbxml.PageContacts, "BusinessAddressCity", wbxml.Text("Worktown")),
		wbxml.E(wbxml.PageContacts, "BusinessAddressState", wbxml.Text("WS")),
		wbxml.E(wbxml.PageContacts, "BusinessAddressPostalCode", wbxml.Text("90210")),
		wbxml.E(wbxml.PageContacts, "BusinessAddressCountry", wbxml.Text("USA")),
		wbxml.E(wbxml.PageContacts, "Birthday", wbxml.Text("1990-04-15T00:00:00.000Z")),
		wbxml.E(wbxml.PageContacts, "Anniversary", wbxml.Text("2018-06-12T00:00:00.000Z")),
		wbxml.E(wbxml.PageContacts, "WebPage", wbxml.Text("https://alice.example")),
		wbxml.E(wbxml.PageContacts, "Picture", wbxml.Text(picB64)),
		wbxml.E(wbxml.PageContacts, "Categories",
			wbxml.E(wbxml.PageContacts, "Category", wbxml.Text("personal")),
			wbxml.E(wbxml.PageContacts, "Category", wbxml.Text("vip")),
		),
		// Wrong codepage should be silently skipped.
		wbxml.E(wbxml.PageEmail, "Subject", wbxml.Text("ignored")),
	)
	got := parseContactItem("c-99", app)
	wantNames := map[string]string{
		"ServerID":       got.ServerID,
		"FirstName":      got.FirstName,
		"LastName":       got.LastName,
		"MiddleName":     got.MiddleName,
		"Title":          got.Title,
		"Suffix":         got.Suffix,
		"FileAs":         got.FileAs,
		"CompanyName":    got.CompanyName,
		"Department":     got.Department,
		"JobTitle":       got.JobTitle,
		"OfficeLocation": got.OfficeLocation,
		"Email1Address":  got.Email1Address,
		"Email2Address":  got.Email2Address,
		"Email3Address":  got.Email3Address,
		"HomePhone":      got.HomePhone,
		"BusinessPhone":  got.BusinessPhone,
		"MobilePhone":    got.MobilePhone,
		"WebPage":        got.WebPage,
	}
	expect := map[string]string{
		"ServerID":       "c-99",
		"FirstName":      "Alice",
		"LastName":       "Engineer",
		"MiddleName":     "Q",
		"Title":          "Dr.",
		"Suffix":         "PhD",
		"FileAs":         "Engineer, Alice",
		"CompanyName":    "Acme",
		"Department":     "R&D",
		"JobTitle":       "Staff SWE",
		"OfficeLocation": "HQ-3",
		"Email1Address":  "alice@x",
		"Email2Address":  "a.engineer@personal",
		"Email3Address":  "a@old",
		"HomePhone":      "+1-555-0100",
		"BusinessPhone":  "+1-555-0200",
		"MobilePhone":    "+1-555-0300",
		"WebPage":        "https://alice.example",
	}
	for k, v := range expect {
		if wantNames[k] != v {
			t.Errorf("%s = %q, want %q", k, wantNames[k], v)
		}
	}
	if got.HomeAddress.Street != "1 Home St" || got.HomeAddress.Country != "Canada" {
		t.Errorf("HomeAddress = %+v", got.HomeAddress)
	}
	if got.BusinessAddress.City != "Worktown" || got.BusinessAddress.PostalCode != "90210" {
		t.Errorf("BusinessAddress = %+v", got.BusinessAddress)
	}
	if got.Birthday.IsZero() || got.Anniversary.IsZero() {
		t.Errorf("dates = %+v", got)
	}
	if !bytes.Equal(got.Picture, pic) {
		t.Errorf("Picture decoded = %v want %v", got.Picture, pic)
	}
	if len(got.Categories) != 2 || got.Categories[0] != "personal" || got.Categories[1] != "vip" {
		t.Errorf("Categories = %v", got.Categories)
	}
}

func TestParseContactItem_nilSafe(t *testing.T) {
	got := parseContactItem("only-id", nil)
	if got.ServerID != "only-id" {
		t.Errorf("ServerID = %q", got.ServerID)
	}
}

func TestParseContactItem_invalidPictureB64IsIgnored(t *testing.T) {
	app := wbxml.E(wbxml.PageAirSync, "ApplicationData",
		wbxml.E(wbxml.PageContacts, "Picture", wbxml.Text("!!!not-base64!!!")),
	)
	got := parseContactItem("c-1", app)
	if got.Picture != nil {
		t.Errorf("Picture = %v, want nil for invalid base64", got.Picture)
	}
}

// The shared sync helpers (addItemViaSync, changeItemViaSync,
// deleteItemViaSync) live in contacts.go because that's the original
// home of the per-item Sync wrapper logic. Their argument-validation
// branches deserve a co-located test.
func TestSyncHelpers_rejectMissingArgs(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("server should not be hit on validation failure")
	})
	cases := []struct {
		name string
		fn   func() error
	}{
		{"add empty folder", func() error {
			_, err := c.addItemViaSync(context.Background(), "", wbxml.E(wbxml.PageAirSync, "ApplicationData"))
			return err
		}},
		{"change empty folder", func() error {
			return c.changeItemViaSync(context.Background(), "", "id", wbxml.E(wbxml.PageAirSync, "ApplicationData"))
		}},
		{"change empty server id", func() error {
			return c.changeItemViaSync(context.Background(), "f", "", wbxml.E(wbxml.PageAirSync, "ApplicationData"))
		}},
		{"delete empty folder", func() error {
			return c.deleteItemViaSync(context.Background(), "", "id")
		}},
		{"delete empty server id", func() error {
			return c.deleteItemViaSync(context.Background(), "f", "")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err == nil {
				t.Error("want error")
			}
		})
	}
}
