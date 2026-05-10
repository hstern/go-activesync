// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

// Package eas implements an Exchange ActiveSync (EAS) 14.1 client.
//
// EAS is Microsoft's mobile-mail sync protocol, originally for Outlook
// Mobile but supported by open-source servers including Z-Push and SOGo.
// The wire format is HTTP(S) POST with WBXML bodies (see the wbxml
// package). All commands except Ping and OPTIONS use WBXML.
//
// # Usage
//
// A Client wraps the connection state for a single account. Construct one
// with NewClient, providing pre-resolved credentials and a StateStore for
// SyncKey / PolicyKey persistence:
//
//	c, err := eas.NewClient(eas.Config{
//	    ServerURL: "https://mail.example.com/Microsoft-Server-ActiveSync",
//	    Username:  "henry",
//	    Password:  pw,                  // already retrieved from keyring
//	    DeviceID:  "32hexcharsofidhere",
//	    State:     eas.NewMemoryState(),
//	})
//	if err != nil { return err }
//
//	if err := c.Provision(ctx); err != nil { return err }
//	folders, err := c.FolderSync(ctx)
//
// # Stateful commands
//
// Commands that change server-side state (Provision, Sync, FolderSync) read
// and write keys via the StateStore. Persist state across process restarts
// by providing a durable implementation; an in-memory store is supplied for
// tests and one-shot CLIs.
//
// # More
//
// See README.md in this directory for an API tour with diagrams,
// per-class command tables, and worked examples (S/MIME, recurrence,
// structured search). Or browse the full godoc.
package eas
