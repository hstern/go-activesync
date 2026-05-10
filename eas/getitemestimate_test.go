// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"context"
	"net/http"
	"testing"

	"github.com/hstern/go-activesync/wbxml"
)

func TestGetItemEstimate_basic(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageGetItemEstimate, "GetItemEstimate",
				wbxml.E(wbxml.PageGetItemEstimate, "Response",
					wbxml.E(wbxml.PageGetItemEstimate, "Status", wbxml.Text("1")),
					wbxml.E(wbxml.PageGetItemEstimate, "Collection",
						wbxml.E(wbxml.PageGetItemEstimate, "CollectionId", wbxml.Text("inbox")),
						wbxml.E(wbxml.PageGetItemEstimate, "Class", wbxml.Text("Email")),
						wbxml.E(wbxml.PageGetItemEstimate, "Estimate", wbxml.Text("42")),
					),
				),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	_ = c.cfg.State.SetSyncKey(context.Background(), "inbox", "S1")
	res, err := c.GetItemEstimate(context.Background(), []string{"inbox"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].Estimate != 42 || res[0].CollectionID != "inbox" {
		t.Errorf("res = %+v", res)
	}
}

func TestGetItemEstimate_emptyArgs(t *testing.T) {
	c, _, _ := newTestClient(t, nil)
	if _, err := c.GetItemEstimate(context.Background(), nil); err == nil {
		t.Error("want error")
	}
}
