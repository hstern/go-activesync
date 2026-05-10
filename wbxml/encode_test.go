// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package wbxml

import (
	"bytes"
	"testing"
)

func TestWriteMBUInt32(t *testing.T) {
	cases := []struct {
		v    uint32
		want []byte
	}{
		{0, []byte{0x00}},
		{0x6A, []byte{0x6A}},
		{0x7F, []byte{0x7F}},
		{0x80, []byte{0x81, 0x00}},
		{0x81, []byte{0x81, 0x01}},
		{0x4000, []byte{0x81, 0x80, 0x00}},
		{0xFFFFFFFF, []byte{0x8F, 0xFF, 0xFF, 0xFF, 0x7F}},
	}
	for _, c := range cases {
		var buf bytes.Buffer
		writeMBUInt32(&buf, c.v)
		if !bytes.Equal(buf.Bytes(), c.want) {
			t.Errorf("v=%#x: got %v, want %v", c.v, buf.Bytes(), c.want)
		}
	}
}

// TestMarshal_FolderSyncRequest builds a minimal FolderSync request and
// asserts the exact byte layout. This is the simplest non-trivial EAS
// request: <FolderSync xmlns="FolderHierarchy"><SyncKey>0</SyncKey></FolderSync>.
func TestMarshal_FolderSyncRequest(t *testing.T) {
	doc := &Document{
		Root: E(PageFolderHierarchy, "FolderSync",
			E(PageFolderHierarchy, "SyncKey", Text("0")),
		),
	}
	got, err := Marshal(doc, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{
		// header
		0x03,       // version 1.3
		0x01,       // public id 1
		0x6A,       // charset 106 (UTF-8)
		0x00,       // empty string table length
		0x00, 0x07, // SWITCH_PAGE to FolderHierarchy (page 7)
		0x56, // FolderSync (0x16) | content (0x40) = 0x56
		0x52, // SyncKey (0x12) | content (0x40) = 0x52
		0x03, // STR_I
		'0',  // text
		0x00, // NUL terminator
		0x01, // END (close SyncKey)
		0x01, // END (close FolderSync)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("FolderSync bytes mismatch\n got=% X\nwant=% X", got, want)
	}
}

// TestMarshal_SwitchPageMidStream verifies a multi-codepage doc emits
// SWITCH_PAGE at the right point and the boundary book-keeping is correct.
func TestMarshal_SwitchPageMidStream(t *testing.T) {
	// <Sync xmlns="AirSync"><Collections><Collection>
	//   <SyncKey>0</SyncKey>
	//   <CollectionId>1</CollectionId>
	//   <Options><BodyPreference xmlns="AirSyncBase">
	//     <Type>1</Type></BodyPreference></Options>
	// </Collection></Collections></Sync>
	doc := &Document{
		Root: E(PageAirSync, "Sync",
			E(PageAirSync, "Collections",
				E(PageAirSync, "Collection",
					E(PageAirSync, "SyncKey", Text("0")),
					E(PageAirSync, "CollectionId", Text("1")),
					E(PageAirSync, "Options",
						E(PageAirSyncBase, "BodyPreference",
							E(PageAirSyncBase, "Type", Text("1")),
						),
					),
				),
			),
		),
	}
	b, err := Marshal(doc, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	// Locate the SWITCH_PAGE bytes (0x00 page).
	switches := 0
	for i := 0; i < len(b)-1; i++ {
		if b[i] == 0x00 && b[i+1] == PageAirSyncBase {
			switches++
		}
	}
	if switches != 1 {
		t.Errorf("expected exactly one SWITCH_PAGE to AirSyncBase, got %d (bytes=% X)", switches, b)
	}
	// Round-trip should yield byte-equal output.
	doc2, err := Unmarshal(b, DefaultRegistry())
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	b2, err := Marshal(doc2, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, b2) {
		t.Errorf("round-trip not byte-equal\nfirst =% X\nsecond=% X", b, b2)
	}
}

func TestMarshal_OpaqueContent(t *testing.T) {
	mime := []byte("From: a@b\r\n\r\nhi")
	doc := &Document{
		Root: E(PageAirSyncBase, "Body",
			E(PageAirSyncBase, "Data", Opaque(mime)),
		),
	}
	b, err := Marshal(doc, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	doc2, err := Unmarshal(b, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	data := doc2.Root.Find("Data")
	if data == nil || len(data.Children) != 1 {
		t.Fatal("Data element missing or wrong child count")
	}
	op, ok := data.Children[0].(Opaque)
	if !ok {
		t.Fatalf("data child = %T, want Opaque", data.Children[0])
	}
	if !bytes.Equal(op, mime) {
		t.Errorf("opaque body mismatch:\n got=%q\nwant=%q", op, mime)
	}
}

func TestMarshal_RejectsUnknownTag(t *testing.T) {
	doc := &Document{Root: E(PageAirSync, "NonExistent")}
	_, err := Marshal(doc, DefaultRegistry())
	if err == nil {
		t.Fatal("want error for unknown tag")
	}
}

func TestMarshal_RejectsTextWithNUL(t *testing.T) {
	doc := &Document{Root: E(PageAirSync, "SyncKey", Text("a\x00b"))}
	_, err := Marshal(doc, DefaultRegistry())
	if err == nil {
		t.Fatal("want error for NUL in text")
	}
}

func TestMarshal_RejectsNilDocument(t *testing.T) {
	if _, err := Marshal(nil, DefaultRegistry()); err == nil {
		t.Fatal("want error for nil doc")
	}
}

func TestMarshal_RejectsNilRoot(t *testing.T) {
	if _, err := Marshal(&Document{}, DefaultRegistry()); err == nil {
		t.Fatal("want error for nil root")
	}
}

func TestMarshal_RejectsNilRegistry(t *testing.T) {
	if _, err := Marshal(&Document{Root: E(0, "x")}, nil); err == nil {
		t.Fatal("want error for nil registry")
	}
}
