// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"bytes"
	"testing"
	"time"

	"github.com/hstern/go-activesync/wbxml"
)

func TestParseEASTime(t *testing.T) {
	cases := []struct {
		in     string
		want   string // RFC3339 representation, or "" for zero
		isZero bool
	}{
		{"", "", true},
		{"   ", "", true},
		{"2024-01-15T12:34:56.000Z", "2024-01-15T12:34:56Z", false},
		{"2024-01-15T12:34:56Z", "2024-01-15T12:34:56Z", false},
		// iCalendar basic forms (Z-Push BackendCalDAV/CardDAV pass these
		// through unchanged from the underlying VEVENT/VTODO).
		{"20240115T123456Z", "2024-01-15T12:34:56Z", false},
		{"20240115T123456", "2024-01-15T12:34:56Z", false},
		{"20240115", "2024-01-15T00:00:00Z", false},
		{"junk", "", true},
	}
	for _, c := range cases {
		got, ok := parseEASTime(c.in)
		if c.isZero {
			if ok || !got.IsZero() {
				t.Errorf("parseEASTime(%q) = (%v, %v), want (zero, false)", c.in, got, ok)
			}
			continue
		}
		if !ok {
			t.Errorf("parseEASTime(%q) returned ok=false, want true", c.in)
			continue
		}
		want, _ := time.Parse(time.RFC3339, c.want)
		if !got.Equal(want) {
			t.Errorf("parseEASTime(%q) = %v, want %v", c.in, got, want)
		}
	}
}

func TestParseEmailItem_minimal(t *testing.T) {
	app := wbxml.E(wbxml.PageAirSync, "ApplicationData",
		wbxml.E(wbxml.PageEmail, "Subject", wbxml.Text("Hello")),
		wbxml.E(wbxml.PageEmail, "From", wbxml.Text("alice@example.com")),
		wbxml.E(wbxml.PageEmail, "To", wbxml.Text("bob@example.com")),
		wbxml.E(wbxml.PageEmail, "Read", wbxml.Text("1")),
		wbxml.E(wbxml.PageEmail, "DateReceived", wbxml.Text("2024-01-15T12:34:56.000Z")),
	)
	got := parseEmailItem("123", app)
	if got.ServerID != "123" {
		t.Errorf("ServerID = %q", got.ServerID)
	}
	if got.Subject != "Hello" {
		t.Errorf("Subject = %q", got.Subject)
	}
	if got.From != "alice@example.com" {
		t.Errorf("From = %q", got.From)
	}
	if got.To != "bob@example.com" {
		t.Errorf("To = %q", got.To)
	}
	if !got.Read {
		t.Errorf("Read = false")
	}
	if got.DateReceived.IsZero() {
		t.Errorf("DateReceived not parsed")
	}
}

func TestParseEmailItem_bodyPlainText(t *testing.T) {
	app := wbxml.E(wbxml.PageAirSync, "ApplicationData",
		wbxml.E(wbxml.PageAirSyncBase, "Body",
			wbxml.E(wbxml.PageAirSyncBase, "Type", wbxml.Text("1")),
			wbxml.E(wbxml.PageAirSyncBase, "EstimatedDataSize", wbxml.Text("12")),
			wbxml.E(wbxml.PageAirSyncBase, "Truncated", wbxml.Text("0")),
			wbxml.E(wbxml.PageAirSyncBase, "Data", wbxml.Text("hello world\n")),
		),
	)
	got := parseEmailItem("x", app)
	if got.BodyType != BodyTypePlain {
		t.Errorf("BodyType = %v", got.BodyType)
	}
	if got.BodyEstimatedSize != 12 {
		t.Errorf("BodyEstimatedSize = %d", got.BodyEstimatedSize)
	}
	if got.BodyTruncated {
		t.Errorf("BodyTruncated = true")
	}
	if got.Body != "hello world\n" {
		t.Errorf("Body = %q", got.Body)
	}
	if got.BodyMIME != nil {
		t.Errorf("BodyMIME unexpectedly populated for plain")
	}
}

func TestParseEmailItem_bodyMIME(t *testing.T) {
	mime := []byte("From: a\r\nSubject: x\r\n\r\nbody")
	app := wbxml.E(wbxml.PageAirSync, "ApplicationData",
		wbxml.E(wbxml.PageAirSyncBase, "Body",
			wbxml.E(wbxml.PageAirSyncBase, "Type", wbxml.Text("4")),
			wbxml.E(wbxml.PageAirSyncBase, "Data", wbxml.Opaque(mime)),
		),
	)
	got := parseEmailItem("x", app)
	if got.BodyType != BodyTypeMIME {
		t.Errorf("BodyType = %v", got.BodyType)
	}
	if !bytes.Equal(got.BodyMIME, mime) {
		t.Errorf("BodyMIME mismatch:\n got=%q\nwant=%q", got.BodyMIME, mime)
	}
	if got.Body != "" {
		t.Errorf("Body unexpectedly populated for MIME: %q", got.Body)
	}
}

func TestParseEmailItem_flagAndImportance(t *testing.T) {
	app := wbxml.E(wbxml.PageAirSync, "ApplicationData",
		wbxml.E(wbxml.PageEmail, "Importance", wbxml.Text("2")),
		wbxml.E(wbxml.PageEmail, "Flag",
			wbxml.E(wbxml.PageEmail, "FlagStatus", wbxml.Text("2")),
		),
	)
	got := parseEmailItem("x", app)
	if got.Importance != 2 {
		t.Errorf("Importance = %d", got.Importance)
	}
	if got.FlagStatus != 2 {
		t.Errorf("FlagStatus = %d", got.FlagStatus)
	}
	if !got.Flagged() {
		t.Errorf("Flagged() = false")
	}
}

func TestParseEmailItem_attachmentsAndConversation(t *testing.T) {
	app := wbxml.E(wbxml.PageAirSync, "ApplicationData",
		wbxml.E(wbxml.PageAirSyncBase, "Attachments",
			wbxml.E(wbxml.PageAirSyncBase, "Attachment"),
		),
		wbxml.E(wbxml.PageEmail2, "ConversationId",
			wbxml.Opaque([]byte{0xDE, 0xAD, 0xBE, 0xEF}),
		),
	)
	got := parseEmailItem("x", app)
	if !got.HasAttachments {
		t.Errorf("HasAttachments = false")
	}
	if !bytes.Equal(got.ConversationID, []byte{0xDE, 0xAD, 0xBE, 0xEF}) {
		t.Errorf("ConversationID = % X", got.ConversationID)
	}
}

func TestParseEmailItem_conversationIDHexText(t *testing.T) {
	app := wbxml.E(wbxml.PageAirSync, "ApplicationData",
		wbxml.E(wbxml.PageEmail2, "ConversationId", wbxml.Text("DEADBEEF")),
	)
	got := parseEmailItem("x", app)
	if !bytes.Equal(got.ConversationID, []byte{0xDE, 0xAD, 0xBE, 0xEF}) {
		t.Errorf("ConversationID from hex text = % X", got.ConversationID)
	}
}

func TestParseEmailItem_nilApp(t *testing.T) {
	got := parseEmailItem("x", nil)
	if got.ServerID != "x" {
		t.Errorf("ServerID = %q", got.ServerID)
	}
	if got.Subject != "" {
		t.Errorf("Subject populated from nil app: %q", got.Subject)
	}
}
