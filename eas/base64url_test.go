// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
)

func TestBuildBase64Query_layout(t *testing.T) {
	encoded, err := buildBase64Query("14.1", "FolderSync", "DEV1234", "DEV", "POL")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if raw[0] != 141 {
		t.Errorf("version byte = %d", raw[0])
	}
	if raw[1] != commandTokens["FolderSync"] {
		t.Errorf("cmd byte = %d", raw[1])
	}
	if raw[2] != 0 || raw[3] != 0 {
		t.Errorf("locale = %d,%d", raw[2], raw[3])
	}
	// device id length + value
	idLen := raw[4]
	if int(idLen) != len("DEV1234") {
		t.Errorf("idLen = %d", idLen)
	}
	if string(raw[5:5+idLen]) != "DEV1234" {
		t.Errorf("device id = %q", raw[5:5+idLen])
	}
	// device type length + value
	off := 5 + int(idLen)
	dtLen := raw[off]
	if int(dtLen) != len("DEV") {
		t.Errorf("dtLen = %d", dtLen)
	}
	if string(raw[off+1:off+1+int(dtLen)]) != "DEV" {
		t.Errorf("device type = %q", raw[off+1:off+1+int(dtLen)])
	}
	off += 1 + int(dtLen)
	pkLen := raw[off]
	if int(pkLen) != len("POL") {
		t.Errorf("pkLen = %d", pkLen)
	}
	if string(raw[off+1:off+1+int(pkLen)]) != "POL" {
		t.Errorf("policy key = %q", raw[off+1:off+1+int(pkLen)])
	}
}

func TestBuildBase64Query_unknownCommand(t *testing.T) {
	if _, err := buildBase64Query("14.1", "Frobnicate", "x", "M", ""); err == nil {
		t.Error("want error for unknown command")
	}
}

func TestBuildBase64Query_oversizedField(t *testing.T) {
	big := strings.Repeat("X", 256)
	if _, err := buildBase64Query("14.1", "FolderSync", big, "M", ""); err == nil {
		t.Error("want error for oversized field")
	}
}

func TestEncodeVersion(t *testing.T) {
	cases := map[string]byte{
		"14.0": 140, "14.1": 141, "16.0": 160, "12.1": 121,
		"":   141, // fallback
		"99": 99,
	}
	for in, want := range cases {
		if got := encodeVersion(in); got != want {
			t.Errorf("encodeVersion(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestCommandURL_base64Mode(t *testing.T) {
	c, cap, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	})
	c.cfg.Base64URL = true

	if _, err := c.postRaw(context.Background(), "FolderSync", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if cap.url == nil {
		t.Fatal("captured URL is nil")
	}
	q := cap.url.RawQuery
	if q == "" {
		t.Fatal("query empty")
	}
	if strings.Contains(q, "&") {
		t.Errorf("base64 mode should produce a single query parameter, got %q", q)
	}
	// Decode and confirm command byte.
	raw, err := base64.RawURLEncoding.DecodeString(q)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if raw[1] != commandTokens["FolderSync"] {
		t.Errorf("cmd byte = %d", raw[1])
	}
}
