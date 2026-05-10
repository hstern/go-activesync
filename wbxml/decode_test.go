// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package wbxml

import (
	"bytes"
	"strings"
	"testing"
)

func TestUnmarshal_FolderSyncResponse(t *testing.T) {
	// Hand-rolled bytes: <FolderSync><Status>1</Status><SyncKey>1</SyncKey>
	//                    <Changes><Count>0</Count></Changes></FolderSync>
	in := []byte{
		0x03, 0x01, 0x6A, 0x00, // header
		0x00, 0x07, // SWITCH_PAGE FolderHierarchy
		0x56,                     // FolderSync (content)
		0x4C, 0x03, '1', 0, 0x01, // Status (content) STR_I "1" END
		0x52, 0x03, '1', 0, 0x01, // SyncKey (content) STR_I "1" END
		0x4E,                     // Changes (content)
		0x57, 0x03, '0', 0, 0x01, // Count (content) STR_I "0" END
		0x01, // END Changes
		0x01, // END FolderSync
	}
	doc, err := Unmarshal(in, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if doc.Root.Name != "FolderSync" {
		t.Errorf("root: %q", doc.Root.Name)
	}
	if got := doc.Root.Find("Status").TextContent(); got != "1" {
		t.Errorf("Status = %q", got)
	}
	if got := doc.Root.Find("SyncKey").TextContent(); got != "1" {
		t.Errorf("SyncKey = %q", got)
	}
	if got := doc.Root.Find("Count").TextContent(); got != "0" {
		t.Errorf("Count = %q", got)
	}
}

func TestUnmarshal_RejectsTruncated(t *testing.T) {
	// Header + SWITCH_PAGE + FolderSync tag + END for status but missing
	// matching END tokens — expect EOF before completing the structure.
	in := []byte{0x03, 0x01, 0x6A, 0x00, 0x00, 0x07, 0x56, 0x4C}
	if _, err := Unmarshal(in, DefaultRegistry()); err == nil {
		t.Fatal("want error for truncated input")
	}
}

func TestUnmarshal_RejectsUnknownTag(t *testing.T) {
	// id 0x11 on page 0 (AirSync) is unallocated in the spec.
	in := []byte{0x03, 0x01, 0x6A, 0x00, 0x11}
	_, err := Unmarshal(in, DefaultRegistry())
	if err == nil || !strings.Contains(err.Error(), "unknown tag") {
		t.Errorf("err = %v", err)
	}
}

func TestUnmarshal_RejectsAttributesFlag(t *testing.T) {
	// Tag byte 0xD6 = 0x80 (attrs) | 0x40 (content) | 0x16 (FolderSync id)
	// after a SWITCH_PAGE to FolderHierarchy. Attributes are unsupported.
	in := []byte{0x03, 0x01, 0x6A, 0x00, 0x00, 0x07, 0xD6}
	_, err := Unmarshal(in, DefaultRegistry())
	if err == nil || !strings.Contains(err.Error(), "attributes") {
		t.Errorf("err = %v", err)
	}
}

func TestUnmarshal_PreservesHeaderFields(t *testing.T) {
	in := []byte{0x03, 0x01, 0x6A, 0x00, 0x00, 0x07, 0x16}
	doc, err := Unmarshal(in, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if doc.Version != 0x03 || doc.PublicID != 1 || doc.Charset != 0x6A {
		t.Errorf("header: v=%d pid=%d charset=%d", doc.Version, doc.PublicID, doc.Charset)
	}
}

func TestUnmarshal_HandlesStringTable(t *testing.T) {
	// String table with 3 bytes — we should consume them and not blow up.
	in := []byte{
		0x03, 0x01, 0x6A,
		0x03, 'a', 'b', 'c',
		0x00, 0x07, 0x16, // SWITCH_PAGE then bare FolderSync (no content)
	}
	doc, err := Unmarshal(in, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if doc.Root.Name != "FolderSync" {
		t.Errorf("root: %q", doc.Root.Name)
	}
}

func TestRoundTrip_Random(t *testing.T) {
	// A more complex synthetic document covering all node types and a
	// codepage switch. Round-trip must be byte-equal.
	doc := &Document{
		Root: E(PageAirSync, "Sync",
			E(PageAirSync, "Collections",
				E(PageAirSync, "Collection",
					E(PageAirSync, "SyncKey", Text("0")),
					E(PageAirSync, "CollectionId", Text("inbox-id")),
					E(PageAirSync, "Options",
						E(PageAirSyncBase, "BodyPreference",
							E(PageAirSyncBase, "Type", Text("2")),
							E(PageAirSyncBase, "TruncationSize", Text("32768")),
						),
					),
					E(PageAirSync, "Commands",
						E(PageAirSync, "Add",
							E(PageAirSync, "ServerId", Text("1:42")),
							E(PageAirSync, "ApplicationData",
								E(PageAirSyncBase, "Body",
									E(PageAirSyncBase, "Data", Opaque([]byte("MIME bytes\x00with NUL"))),
								),
							),
						),
					),
				),
			),
		),
	}
	b1, err := Marshal(doc, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	doc2, err := Unmarshal(b1, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	b2, err := Marshal(doc2, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Errorf("round-trip mismatch\nfirst =% X\nsecond=% X", b1, b2)
	}
}

func TestUnmarshal_NilRegistry(t *testing.T) {
	if _, err := Unmarshal([]byte{0x03, 0x01, 0x6A, 0x00}, nil); err == nil {
		t.Fatal("want error")
	}
}
