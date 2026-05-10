// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"context"
	"net/http"
	"testing"

	"github.com/hstern/go-activesync/wbxml"
)

func TestValidateCert_perCertStatus(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageValidateCert, "ValidateCert",
				wbxml.E(wbxml.PageValidateCert, "Status", wbxml.Text("1")),
				wbxml.E(wbxml.PageValidateCert, "Certificate",
					wbxml.E(wbxml.PageValidateCert, "Status", wbxml.Text("1")),
				),
				wbxml.E(wbxml.PageValidateCert, "Certificate",
					wbxml.E(wbxml.PageValidateCert, "Status", wbxml.Text("17")), // unspecified
				),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	results, err := c.ValidateCert(context.Background(), [][]byte{{0x30, 0x01}, {0x30, 0x02}}, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].Status != 1 || results[1].Status != 17 {
		t.Errorf("results = %+v", results)
	}
}

func TestValidateCert_validation(t *testing.T) {
	c, _, _ := newTestClient(t, nil)
	if _, err := c.ValidateCert(context.Background(), nil, nil, false); err == nil {
		t.Error("want error")
	}
}
