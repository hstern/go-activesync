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

func TestResolveRecipients_basic(t *testing.T) {
	c, cap, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageResolveRecipients, "ResolveRecipients",
				wbxml.E(wbxml.PageResolveRecipients, "Status", wbxml.Text("1")),
				wbxml.E(wbxml.PageResolveRecipients, "Response",
					wbxml.E(wbxml.PageResolveRecipients, "To", wbxml.Text("alice")),
					wbxml.E(wbxml.PageResolveRecipients, "Status", wbxml.Text("1")),
					wbxml.E(wbxml.PageResolveRecipients, "RecipientCount", wbxml.Text("1")),
					wbxml.E(wbxml.PageResolveRecipients, "Recipient",
						wbxml.E(wbxml.PageResolveRecipients, "Type", wbxml.Text("1")),
						wbxml.E(wbxml.PageResolveRecipients, "DisplayName", wbxml.Text("Alice Engineer")),
						wbxml.E(wbxml.PageResolveRecipients, "EmailAddress", wbxml.Text("alice@x")),
						wbxml.E(wbxml.PageResolveRecipients, "Availability",
							wbxml.E(wbxml.PageResolveRecipients, "Status", wbxml.Text("1")),
							wbxml.E(wbxml.PageResolveRecipients, "MergedFreeBusy", wbxml.Text("000022000000")),
						),
					),
				),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	now := time.Date(2026, 5, 9, 9, 0, 0, 0, time.UTC)
	res, err := c.ResolveRecipients(context.Background(), []string{"alice"}, ResolveOptions{
		AvailabilityStart: now,
		AvailabilityEnd:   now.Add(8 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || len(res[0].Recipients) != 1 {
		t.Fatalf("res = %+v", res)
	}
	r := res[0].Recipients[0]
	if r.EmailAddress != "alice@x" {
		t.Errorf("email = %q", r.EmailAddress)
	}
	if r.MergedFreeBusy != "000022000000" {
		t.Errorf("free/busy = %q", r.MergedFreeBusy)
	}
	// Request shape: To + Availability options.
	req, _ := wbxml.Unmarshal(cap.body, wbxml.DefaultRegistry())
	if to := req.Root.Find("To"); to == nil || to.TextContent() != "alice" {
		t.Errorf("To = %v", to)
	}
	if av := req.Root.Find("Availability"); av == nil {
		t.Error("Availability not in request")
	}
}

func TestResolveRecipients_validation(t *testing.T) {
	c, _, _ := newTestClient(t, nil)
	if _, err := c.ResolveRecipients(context.Background(), nil, ResolveOptions{}); err == nil {
		t.Error("want error")
	}
}
