// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

func TestOptions_parsesHeaders(t *testing.T) {
	c, cap, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("MS-ASProtocolVersions", "2.5,12.0,12.1,14.0,14.1")
		w.Header().Set("MS-ASProtocolCommands", "Sync,FolderSync,Provision,SendMail")
		w.WriteHeader(http.StatusOK)
	})
	got, err := c.Options(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cap.method != http.MethodOptions {
		t.Errorf("method = %q, want OPTIONS", cap.method)
	}
	wantVersions := []string{"2.5", "12.0", "12.1", "14.0", "14.1"}
	if !reflect.DeepEqual(got.ProtocolVersions, wantVersions) {
		t.Errorf("versions = %v", got.ProtocolVersions)
	}
	wantCmds := []string{"Sync", "FolderSync", "Provision", "SendMail"}
	if !reflect.DeepEqual(got.Commands, wantCmds) {
		t.Errorf("commands = %v", got.Commands)
	}
	if !got.Supports("14.1") {
		t.Error("Supports(14.1) = false")
	}
	if got.Supports("99.9") {
		t.Error("Supports(99.9) = true")
	}
	if !got.HasCommand("FolderSync") {
		t.Error("HasCommand(FolderSync) = false")
	}
	if got.HasCommand("Wat") {
		t.Error("HasCommand(Wat) = true")
	}
}

func TestOptions_HTTPError(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no", http.StatusForbidden)
	})
	_, err := c.Options(context.Background())
	if !IsHTTPStatus(err, 403) {
		t.Errorf("err = %v", err)
	}
}

func TestSupportsAndHasCommand_nilSafe(t *testing.T) {
	var o *OptionsResult
	if o.Supports("14.1") {
		t.Error("nil should not support anything")
	}
	if o.HasCommand("X") {
		t.Error("nil should not have any command")
	}
}

func TestNegotiateVersion_picksHighest(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("MS-ASProtocolVersions", "12.1,14.0,14.1,16.0")
		w.WriteHeader(200)
	})
	v, err := c.NegotiateVersion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if v != "14.1" {
		t.Errorf("got %q, want 14.1 (preferred)", v)
	}
	if c.cfg.ASVersion != "14.1" {
		t.Errorf("cfg.ASVersion = %q", c.cfg.ASVersion)
	}
}

func TestNegotiateVersion_fallsBackTo12(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("MS-ASProtocolVersions", "12.0,12.1")
		w.WriteHeader(200)
	})
	v, err := c.NegotiateVersion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if v != "12.1" {
		t.Errorf("got %q, want 12.1", v)
	}
}

func TestNegotiateVersion_optionsErrorReturnsCurrent(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no", 500)
	})
	v, err := c.NegotiateVersion(context.Background())
	if err == nil {
		t.Error("want error from broken OPTIONS")
	}
	if v != "14.1" {
		t.Errorf("got %q, want unchanged 14.1 default", v)
	}
}

func TestSplitCSV(t *testing.T) {
	if got := splitCSV(""); got != nil {
		t.Errorf("empty: %v", got)
	}
	if got := splitCSV("a, b ,c"); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("got %v", got)
	}
}
