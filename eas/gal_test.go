// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"context"
	"net/http"
	"testing"

	"github.com/hstern/go-activesync/wbxml"
)

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
