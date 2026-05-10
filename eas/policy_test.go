// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"context"
	"net/http"
	"testing"

	"github.com/hstern/go-activesync/wbxml"
)

func TestProvision_capturesPolicyFields(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		// Build a Provision response with a real EASProvisionDoc.
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageProvision, "Provision",
				wbxml.E(wbxml.PageProvision, "Status", wbxml.Text("1")),
				wbxml.E(wbxml.PageProvision, "Policies",
					wbxml.E(wbxml.PageProvision, "Policy",
						wbxml.E(wbxml.PageProvision, "PolicyType", wbxml.Text("MS-EAS-Provisioning-WBXML")),
						wbxml.E(wbxml.PageProvision, "Status", wbxml.Text("1")),
						wbxml.E(wbxml.PageProvision, "PolicyKey", wbxml.Text("PK1")),
						wbxml.E(wbxml.PageProvision, "Data",
							wbxml.E(wbxml.PageProvision, "EASProvisionDoc",
								wbxml.E(wbxml.PageProvision, "DevicePasswordEnabled", wbxml.Text("1")),
								wbxml.E(wbxml.PageProvision, "MinDevicePasswordLength", wbxml.Text("8")),
								wbxml.E(wbxml.PageProvision, "RequireDeviceEncryption", wbxml.Text("1")),
								wbxml.E(wbxml.PageProvision, "AllowCamera", wbxml.Text("0")),
								wbxml.E(wbxml.PageProvision, "MaxAttachmentSize", wbxml.Text("10485760")),
								wbxml.E(wbxml.PageProvision, "AllowBluetooth", wbxml.Text("2")),
								wbxml.E(wbxml.PageProvision, "Hash", wbxml.Text("policy-hash-1")),
							),
						),
					),
				),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})

	// Phase 1 alone is enough to populate the policy.
	if _, _, err := c.provisionPhase(context.Background(), "0", false); err != nil {
		t.Fatal(err)
	}
	p := c.LastPolicy()
	if p == nil {
		t.Fatal("policy not captured")
	}
	if !p.DevicePasswordEnabled {
		t.Errorf("DevicePasswordEnabled = false")
	}
	if p.MinDevicePasswordLength != 8 {
		t.Errorf("MinDevicePasswordLength = %d", p.MinDevicePasswordLength)
	}
	if !p.RequireDeviceEncryption {
		t.Errorf("RequireDeviceEncryption = false")
	}
	if p.AllowCamera {
		t.Errorf("AllowCamera should be false")
	}
	if p.MaxAttachmentSizeBytes != 10485760 {
		t.Errorf("MaxAttachmentSize = %d", p.MaxAttachmentSizeBytes)
	}
	if p.AllowBluetooth != 2 {
		t.Errorf("AllowBluetooth = %d", p.AllowBluetooth)
	}
	if p.Hash != "policy-hash-1" {
		t.Errorf("Hash = %q", p.Hash)
	}
}

func TestLastPolicy_nilByDefault(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	})
	if p := c.LastPolicy(); p != nil {
		t.Errorf("LastPolicy = %+v, want nil", p)
	}
}
