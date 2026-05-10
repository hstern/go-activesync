// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

//go:build integration

// Package eas integration tests. Skipped by default; build with the
// "integration" tag and point at a real EAS server via env vars:
//
//	EAS_INTEGRATION_URL    https://host/Microsoft-Server-ActiveSync
//	EAS_INTEGRATION_USER   the account username (often the email address)
//	EAS_INTEGRATION_PASS   the account password
//	EAS_INTEGRATION_DEVICE optional 32-hex device id (default integration00…)
//	EAS_INTEGRATION_VERSION optional EAS version (default: NegotiateVersion result)
//	EAS_INTEGRATION_INSECURE if "1", skip TLS verification (self-signed test cert)
//	EAS_INTEGRATION_VERBOSE  if "1", enable debug logging to stderr
//
// Run from the repo root:
//
//	go test -tags integration -v ./eas
//
// The testenv/ directory contains a Docker Compose stack that provides
// a known-good target; see testenv/README.md.
package eas_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hstern/go-activesync/eas"
)

func mustEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Skipf("set %s to run integration tests", key)
	}
	return v
}

// integrationClient builds a Client wired for the env-supplied server.
func integrationClient(t *testing.T) *eas.Client {
	t.Helper()
	url := mustEnv(t, "EAS_INTEGRATION_URL")
	user := mustEnv(t, "EAS_INTEGRATION_USER")
	pass := mustEnv(t, "EAS_INTEGRATION_PASS")

	deviceID := os.Getenv("EAS_INTEGRATION_DEVICE")
	if deviceID == "" {
		deviceID = "integration00000000000000000000"
	}
	asVersion := os.Getenv("EAS_INTEGRATION_VERSION")
	if asVersion == "" {
		asVersion = "14.1"
	}

	hc := http.DefaultClient
	if os.Getenv("EAS_INTEGRATION_INSECURE") == "1" {
		hc = &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}}
	}

	logLevel := slog.LevelInfo
	if os.Getenv("EAS_INTEGRATION_VERBOSE") == "1" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	c, err := eas.NewClient(eas.Config{
		ServerURL:    url,
		Username:     user,
		Password:     pass,
		DeviceID:     deviceID,
		DeviceType:   "GoActivesyncIntegration",
		ASVersion:    asVersion,
		UserAgent:    "go-activesync-integration-test/0.1",
		HTTPClient:   hc,
		Logger:       logger,
		State:        eas.NewMemoryState(),
		GzipRequests: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func mustOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegration_OptionsAndNegotiate(t *testing.T) {
	c := integrationClient(t)
	ctx := context.Background()

	opts, err := c.Options(ctx)
	mustOK(t, err)
	if len(opts.ProtocolVersions) == 0 {
		t.Fatal("server announced no protocol versions")
	}
	t.Logf("server versions: %v", opts.ProtocolVersions)
	t.Logf("server commands: %v", opts.Commands)

	if !opts.HasCommand("FolderSync") || !opts.HasCommand("Sync") {
		t.Errorf("server missing core commands: %v", opts.Commands)
	}

	v, err := c.NegotiateVersion(ctx)
	mustOK(t, err)
	t.Logf("negotiated: %s", v)
}

func TestIntegration_Provision(t *testing.T) {
	c := integrationClient(t)
	ctx := context.Background()
	mustOK(t, c.Provision(ctx))
	if p := c.LastPolicy(); p != nil {
		t.Logf("server policy: passwordEnabled=%v minLen=%d maxAttachKB=%d",
			p.DevicePasswordEnabled, p.MinDevicePasswordLength,
			p.MaxAttachmentSizeBytes/1024)
	}
}

// provisionedClient returns a freshly-provisioned client ready for
// data commands.
func provisionedClient(t *testing.T) *eas.Client {
	t.Helper()
	c := integrationClient(t)
	ctx := context.Background()
	if _, err := c.NegotiateVersion(ctx); err != nil {
		t.Logf("negotiate (continuing): %v", err)
	}
	if err := c.SettingsDeviceInformation(ctx, eas.DeviceInformation{
		Model: "GoActivesyncIntegration", FriendlyName: "go-activesync test",
	}); err != nil {
		t.Logf("device info (continuing): %v", err)
	}
	mustOK(t, c.Provision(ctx))
	return c
}

func TestIntegration_FolderSync_HasInbox(t *testing.T) {
	c := provisionedClient(t)
	res, err := c.FolderSync(context.Background())
	mustOK(t, err)
	t.Logf("FolderSync: key=%s added=%d updated=%d deleted=%d",
		res.SyncKey, len(res.Added), len(res.Updated), len(res.Deleted))
	for _, f := range res.Added {
		t.Logf("  [%s] %-30s id=%s", f.Type, f.DisplayName, f.ServerID)
	}
	if findInboxID(res.Added) == "" {
		t.Error("no Inbox folder returned")
	}
}

func TestIntegration_SyncEmail_Bootstrap(t *testing.T) {
	c := provisionedClient(t)
	ctx := context.Background()
	folders, err := c.FolderSync(ctx)
	mustOK(t, err)
	inbox := findInboxID(folders.Added)
	if inbox == "" {
		t.Skip("no Inbox folder")
	}
	res, err := c.SyncEmail(ctx, inbox, eas.EmailSyncOptions{WindowSize: 25})
	mustOK(t, err)
	t.Logf("SyncEmail Inbox: key=%s added=%d more=%v",
		res.SyncKey, len(res.Added), res.MoreAvailable)
	for i, e := range res.Added {
		if i >= 3 {
			break
		}
		t.Logf("  %s — %q  from=%s", e.ServerID, e.Subject, e.From)
	}
}

func TestIntegration_SendMail_Loopback(t *testing.T) {
	c := provisionedClient(t)
	ctx := context.Background()
	user := os.Getenv("EAS_INTEGRATION_USER")
	subject := fmt.Sprintf("go-activesync integration loopback %d", time.Now().UnixNano())
	mime := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n"+
			"This message was sent by the go-activesync integration test.\r\n",
		user, user, subject))
	mustOK(t, c.SendMail(ctx, eas.SendMailOptions{MIME: mime, SkipSaveInSent: true}))
	t.Logf("SendMail succeeded; subject=%q", subject)

	// Best-effort verify via a single Sync after a brief settle period.
	// We don't poll aggressively because some servers (z-push 2.7's
	// LoopDetection in particular) treat repeated Syncs from the same
	// device within seconds as a stuck client and refuse with 449.
	folders, err := c.FolderSync(ctx)
	mustOK(t, err)
	inbox := findInboxID(folders.Added)
	if inbox == "" {
		t.Skip("no Inbox; can't verify loopback")
	}

	time.Sleep(5 * time.Second) // give postfix → dovecot LMTP a moment

	// Bootstrap (first call returns no items per protocol) then sync.
	if _, err := c.SyncEmail(ctx, inbox, eas.EmailSyncOptions{
		WindowSize: 1, NoBootstrap: true,
	}); err != nil {
		t.Logf("bootstrap sync (continuing): %v", err)
	}
	res, err := c.SyncEmail(ctx, inbox, eas.EmailSyncOptions{
		WindowSize: 50, BodyType: eas.BodyTypePlain,
	})
	if err != nil {
		t.Logf("post-send sync: %v — SendMail confirmed at protocol level, delivery unverifiable here", err)
		return
	}
	for _, e := range res.Added {
		if strings.Contains(e.Subject, subject) {
			t.Logf("loopback delivered: id=%s", e.ServerID)
			return
		}
	}
	t.Logf("loopback not in this batch (saw %d items); SendMail confirmed at protocol level", len(res.Added))
}

func TestIntegration_Calendar_CRUD(t *testing.T) {
	c := provisionedClient(t)
	ctx := context.Background()
	folders, err := c.FolderSync(ctx)
	mustOK(t, err)
	calID := findFirstByType(folders.Added, eas.FolderTypeCalendar, eas.FolderTypeUserCalendar)
	if calID == "" {
		t.Skip("no Calendar folder")
	}

	start := time.Now().Add(24 * time.Hour).Truncate(time.Hour).UTC()
	end := start.Add(time.Hour)
	subject := fmt.Sprintf("go-activesync integration event %d", time.Now().UnixNano())

	id, err := c.CreateEvent(ctx, calID, eas.EventDraft{
		Subject:    subject,
		Location:   "test",
		StartTime:  start,
		EndTime:    end,
		BusyStatus: 2,
	})
	mustOK(t, err)
	t.Logf("created event id=%s", id)
	t.Cleanup(func() {
		if err := c.DeleteEvent(context.Background(), calID, id); err != nil {
			t.Logf("cleanup DeleteEvent: %v", err)
		}
	})

	// Sync to confirm the event landed.
	res, err := c.SyncCalendar(ctx, calID, eas.CalendarSyncOptions{})
	mustOK(t, err)
	found := false
	for _, ev := range res.Added {
		if ev.Subject == subject {
			found = true
			t.Logf("event visible: id=%s start=%v", ev.ServerID, ev.StartTime)
			break
		}
	}
	if !found {
		t.Logf("event %q not yet visible (sync timing) — checking Changed", subject)
		for _, ev := range res.Changed {
			if ev.Subject == subject {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("created event not visible after Sync")
	}
}

func TestIntegration_GAL_Search(t *testing.T) {
	c := provisionedClient(t)
	ctx := context.Background()
	user := os.Getenv("EAS_INTEGRATION_USER")
	q := user
	if i := strings.IndexByte(q, '@'); i >= 0 {
		q = q[:i]
	}
	res, err := c.GALSearch(ctx, q, 5)
	if err != nil {
		t.Skipf("GAL search unsupported on this server: %v", err)
	}
	t.Logf("GAL: %d entries (total %d)", len(res.Entries), res.Total)
}

func TestIntegration_Settings_OOF_RoundTrip(t *testing.T) {
	c := provisionedClient(t)
	ctx := context.Background()
	cfg, err := c.GetOof(ctx)
	if err != nil {
		t.Skipf("OOF unsupported on this server: %v", err)
	}
	t.Logf("current OOF state: %v", cfg.State)
	// Don't actually set OOF — would surprise the user. Just confirm
	// the read path works.
}

// --- helpers ---------------------------------------------------------------

func findInboxID(folders []eas.Folder) string {
	for _, f := range folders {
		if f.Type == eas.FolderTypeInbox {
			return f.ServerID
		}
	}
	return ""
}

func findFirstByType(folders []eas.Folder, types ...eas.FolderType) string {
	for _, t := range types {
		for _, f := range folders {
			if f.Type == t {
				return f.ServerID
			}
		}
	}
	return ""
}
