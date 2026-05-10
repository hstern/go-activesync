// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/hstern/go-activesync/wbxml"
)

// fakeProvisionServer responds to two consecutive Provision requests with
// (TempKey="TKEY") then (FinalKey="FKEY"). It records the X-MS-PolicyKey
// header sent on each call so the test can verify the client uses the
// temp key on the second request.
type fakeProvisionServer struct {
	mu         sync.Mutex
	calls      int
	policyKeys []string
	tempKey    string
	finalKey   string
}

func (f *fakeProvisionServer) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.calls++
	f.policyKeys = append(f.policyKeys, r.Header.Get("X-MS-PolicyKey"))
	call := f.calls
	f.mu.Unlock()

	var key string
	switch call {
	case 1:
		key = f.tempKey
	case 2:
		key = f.finalKey
	}
	resp := &wbxml.Document{
		Root: wbxml.E(wbxml.PageProvision, "Provision",
			wbxml.E(wbxml.PageProvision, "Status", wbxml.Text("1")),
			wbxml.E(wbxml.PageProvision, "Policies",
				wbxml.E(wbxml.PageProvision, "Policy",
					wbxml.E(wbxml.PageProvision, "PolicyType", wbxml.Text("MS-EAS-Provisioning-WBXML")),
					wbxml.E(wbxml.PageProvision, "Status", wbxml.Text("1")),
					wbxml.E(wbxml.PageProvision, "PolicyKey", wbxml.Text(key)),
				),
			),
		),
	}
	body, err := wbxml.Marshal(resp, wbxml.DefaultRegistry())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
	w.Write(body)
}

func TestProvision_twoPhaseHandshake(t *testing.T) {
	f := &fakeProvisionServer{tempKey: "TKEY", finalKey: "FKEY"}
	c, _, _ := newTestClient(t, f.handle)
	if err := c.Provision(context.Background()); err != nil {
		t.Fatal(err)
	}
	if f.calls != 2 {
		t.Errorf("calls = %d, want 2", f.calls)
	}
	if f.policyKeys[0] != "" {
		t.Errorf("call 1 X-MS-PolicyKey: %q, want empty", f.policyKeys[0])
	}
	if f.policyKeys[1] != "TKEY" {
		t.Errorf("call 2 X-MS-PolicyKey: %q, want TKEY", f.policyKeys[1])
	}
	pk, _ := c.cfg.State.PolicyKey(context.Background())
	if pk != "FKEY" {
		t.Errorf("persisted policy key: %q, want FKEY", pk)
	}
}

func TestProvision_topLevelStatusError(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		resp := &wbxml.Document{
			Root: wbxml.E(wbxml.PageProvision, "Provision",
				wbxml.E(wbxml.PageProvision, "Status", wbxml.Text("142")), // PolicyRefresh
			),
		}
		body, _ := wbxml.Marshal(resp, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	err := c.Provision(context.Background())
	if !IsStatusCode(err, 142) {
		t.Errorf("err = %v", err)
	}
}

func TestRemoteWipeRequested(t *testing.T) {
	if RemoteWipeRequested(nil) {
		t.Error("nil doc should be false")
	}
	if RemoteWipeRequested(&wbxml.Document{}) {
		t.Error("nil root should be false")
	}
	noWipe := &wbxml.Document{Root: wbxml.E(wbxml.PageProvision, "Provision",
		wbxml.E(wbxml.PageProvision, "Status", wbxml.Text("1")),
	)}
	if RemoteWipeRequested(noWipe) {
		t.Error("absent <RemoteWipe> should be false")
	}
	withWipe := &wbxml.Document{Root: wbxml.E(wbxml.PageProvision, "Provision",
		wbxml.E(wbxml.PageProvision, "Status", wbxml.Text("1")),
		wbxml.E(wbxml.PageProvision, "RemoteWipe"),
	)}
	if !RemoteWipeRequested(withWipe) {
		t.Error("present <RemoteWipe> should be true")
	}
}

func TestAcknowledgeRemoteWipe_emitsStatus(t *testing.T) {
	c, cap, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{Root: wbxml.E(wbxml.PageProvision, "Provision",
			wbxml.E(wbxml.PageProvision, "Status", wbxml.Text("1")),
		)}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	if err := c.AcknowledgeRemoteWipe(context.Background(), 0); err != nil { // 0 → defaults to 1
		t.Fatal(err)
	}
	req, _ := wbxml.Unmarshal(cap.body, wbxml.DefaultRegistry())
	wipe := req.Root.Find("RemoteWipe")
	if wipe == nil {
		t.Fatal("missing <RemoteWipe>")
	}
	if st := wipe.Find("Status"); st == nil || st.TextContent() != "1" {
		t.Errorf("Status = %v, want 1 (default)", st)
	}
}

func TestAcknowledgeRemoteWipe_serverError(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{Root: wbxml.E(wbxml.PageProvision, "Provision",
			wbxml.E(wbxml.PageProvision, "Status", wbxml.Text("3")),
		)}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	if err := c.AcknowledgeRemoteWipe(context.Background(), 2); err == nil {
		t.Error("want error for non-OK status")
	}
}

func TestProvision_missingPolicyKey(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		resp := &wbxml.Document{
			Root: wbxml.E(wbxml.PageProvision, "Provision",
				wbxml.E(wbxml.PageProvision, "Status", wbxml.Text("1")),
				wbxml.E(wbxml.PageProvision, "Policies",
					wbxml.E(wbxml.PageProvision, "Policy",
						wbxml.E(wbxml.PageProvision, "PolicyType", wbxml.Text("X")),
						wbxml.E(wbxml.PageProvision, "Status", wbxml.Text("1")),
						// No PolicyKey
					),
				),
			),
		}
		body, _ := wbxml.Marshal(resp, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	err := c.Provision(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no temporary policy key") {
		t.Errorf("err = %v", err)
	}
}
