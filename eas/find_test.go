// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"context"
	"net/http"
	"testing"

	"github.com/hstern/go-activesync/wbxml"
)

func TestFindEmail_basic(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageFind, "Find",
				wbxml.E(wbxml.PageFind, "Status", wbxml.Text("1")),
				wbxml.E(wbxml.PageFind, "Range", wbxml.Text("0-1")),
				wbxml.E(wbxml.PageFind, "Total", wbxml.Text("2")),
				wbxml.E(wbxml.PageFind, "Response",
					wbxml.E(wbxml.PageFind, "Result",
						wbxml.E(wbxml.PageAirSync, "ServerId", wbxml.Text("hit-1")),
						wbxml.E(wbxml.PageFind, "Properties",
							wbxml.E(wbxml.PageEmail, "Subject", wbxml.Text("Test")),
							wbxml.E(wbxml.PageEmail, "From", wbxml.Text("alice@x")),
						),
						wbxml.E(wbxml.PageFind, "Preview", wbxml.Text("This is a preview")),
						wbxml.E(wbxml.PageFind, "HasAttachments", wbxml.Text("1")),
					),
				),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	res, err := c.FindEmail(context.Background(), "test", FindOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 2 || len(res.Hits) != 1 {
		t.Errorf("res = %+v", res)
	}
	if res.Hits[0].Item.Subject != "Test" || res.Hits[0].Preview != "This is a preview" {
		t.Errorf("hit = %+v", res.Hits[0])
	}
	if !res.Hits[0].HasAttachments {
		t.Errorf("HasAttachments = false")
	}
}

func TestFindEmail_emptyQuery(t *testing.T) {
	c, _, _ := newTestClient(t, nil)
	if _, err := c.FindEmail(context.Background(), "", FindOptions{}); err == nil {
		t.Error("want error")
	}
}
