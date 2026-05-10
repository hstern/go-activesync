// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hstern/go-activesync/wbxml"
)

func calendarSyncResponse(syncKey string, events ...*wbxml.Element) []byte {
	collection := wbxml.E(wbxml.PageAirSync, "Collection",
		wbxml.E(wbxml.PageAirSync, "Class", wbxml.Text("Calendar")),
		wbxml.E(wbxml.PageAirSync, "SyncKey", wbxml.Text(syncKey)),
		wbxml.E(wbxml.PageAirSync, "CollectionId", wbxml.Text("cal")),
		wbxml.E(wbxml.PageAirSync, "Status", wbxml.Text("1")),
	)
	if len(events) > 0 {
		commands := wbxml.E(wbxml.PageAirSync, "Commands")
		for _, e := range events {
			commands.Children = append(commands.Children, e)
		}
		collection.Children = append(collection.Children, commands)
	}
	doc := &wbxml.Document{
		Root: wbxml.E(wbxml.PageAirSync, "Sync",
			wbxml.E(wbxml.PageAirSync, "Collections", collection),
		),
	}
	out, err := wbxml.Marshal(doc, wbxml.DefaultRegistry())
	if err != nil {
		panic(err)
	}
	return out
}

func eventAdd(serverID, subject string, start, end time.Time) *wbxml.Element {
	return wbxml.E(wbxml.PageAirSync, "Add",
		wbxml.E(wbxml.PageAirSync, "ServerId", wbxml.Text(serverID)),
		wbxml.E(wbxml.PageAirSync, "ApplicationData",
			wbxml.E(wbxml.PageCalendar, "Subject", wbxml.Text(subject)),
			wbxml.E(wbxml.PageCalendar, "StartTime", wbxml.Text(formatEASTime(start))),
			wbxml.E(wbxml.PageCalendar, "EndTime", wbxml.Text(formatEASTime(end))),
			wbxml.E(wbxml.PageCalendar, "AllDayEvent", wbxml.Text("0")),
			wbxml.E(wbxml.PageCalendar, "BusyStatus", wbxml.Text("2")),
			wbxml.E(wbxml.PageCalendar, "Attendees",
				wbxml.E(wbxml.PageCalendar, "Attendee",
					wbxml.E(wbxml.PageCalendar, "Name", wbxml.Text("Bob")),
					wbxml.E(wbxml.PageCalendar, "Email", wbxml.Text("bob@x")),
					wbxml.E(wbxml.PageCalendar, "AttendeeStatus", wbxml.Text("3")),
				),
			),
		),
	)
}

func TestSyncCalendar_parsesEvent(t *testing.T) {
	start := time.Date(2026, 5, 9, 14, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(calendarSyncResponse("CK1", eventAdd("ev1", "Team standup", start, end)))
	})
	_ = c.cfg.State.SetSyncKey(context.Background(), "cal", "CK0")
	res, err := c.SyncCalendar(context.Background(), "cal", CalendarSyncOptions{NoBootstrap: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 1 {
		t.Fatalf("len(Added)=%d", len(res.Added))
	}
	ev := res.Added[0]
	if ev.Subject != "Team standup" || !ev.StartTime.Equal(start) || !ev.EndTime.Equal(end) {
		t.Errorf("event = %+v", ev)
	}
	if ev.BusyStatus != 2 {
		t.Errorf("BusyStatus = %d", ev.BusyStatus)
	}
	if len(ev.Attendees) != 1 || ev.Attendees[0].Email != "bob@x" {
		t.Errorf("attendees = %+v", ev.Attendees)
	}
}

func TestCreateEvent_returnsServerID(t *testing.T) {
	var (
		mu     sync.Mutex
		calls  int
		bodies [][]byte
	)
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := readCapped(r.Body, 1<<20)
		mu.Lock()
		calls++
		thisCall := calls
		bodies = append(bodies, body)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		switch thisCall {
		case 1:
			// Bootstrap
			w.Write(calendarSyncResponse("CK-BOOT"))
		case 2:
			// Echo back the ClientId with a Server-assigned id.
			req, _ := wbxml.Unmarshal(body, wbxml.DefaultRegistry())
			cid := req.Root.Find("ClientId").TextContent()
			doc := &wbxml.Document{
				Root: wbxml.E(wbxml.PageAirSync, "Sync",
					wbxml.E(wbxml.PageAirSync, "Collections",
						wbxml.E(wbxml.PageAirSync, "Collection",
							wbxml.E(wbxml.PageAirSync, "SyncKey", wbxml.Text("CK-DONE")),
							wbxml.E(wbxml.PageAirSync, "CollectionId", wbxml.Text("cal")),
							wbxml.E(wbxml.PageAirSync, "Status", wbxml.Text("1")),
							wbxml.E(wbxml.PageAirSync, "Responses",
								wbxml.E(wbxml.PageAirSync, "Add",
									wbxml.E(wbxml.PageAirSync, "ClientId", wbxml.Text(cid)),
									wbxml.E(wbxml.PageAirSync, "ServerId", wbxml.Text("cal:NEW")),
									wbxml.E(wbxml.PageAirSync, "Status", wbxml.Text("1")),
								),
							),
						),
					),
				),
			}
			b, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
			w.Write(b)
		}
	})

	id, err := c.CreateEvent(context.Background(), "cal", EventDraft{
		Subject:   "Test",
		StartTime: time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "cal:NEW" {
		t.Errorf("id = %q", id)
	}
	if calls != 2 {
		t.Errorf("calls = %d (bootstrap + change expected)", calls)
	}
	// Body of second call should contain Subject "Test".
	if !strings.Contains(string(bodies[1]), "Test") {
		t.Errorf("body 2 missing subject")
	}
}

func TestUpdateEvent_andDeleteEvent(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(calendarSyncResponse("CK"))
	})
	_ = c.cfg.State.SetSyncKey(context.Background(), "cal", "CK-PRIOR")

	if err := c.UpdateEvent(context.Background(), "cal", "ev1", EventDraft{
		Subject:   "Renamed",
		StartTime: time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteEvent(context.Background(), "cal", "ev1"); err != nil {
		t.Fatal(err)
	}
}

func TestRespondInvite_basic(t *testing.T) {
	c, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		doc := &wbxml.Document{
			Root: wbxml.E(wbxml.PageMeetingResponse, "MeetingResponse",
				wbxml.E(wbxml.PageMeetingResponse, "Result",
					wbxml.E(wbxml.PageMeetingResponse, "Status", wbxml.Text("1")),
					wbxml.E(wbxml.PageMeetingResponse, "CalendarId", wbxml.Text("cal:42")),
				),
			),
		}
		body, _ := wbxml.Marshal(doc, wbxml.DefaultRegistry())
		w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
		w.Write(body)
	})
	res, err := c.RespondInvite(context.Background(), "inbox", "invite-1", MeetingAccept)
	if err != nil {
		t.Fatal(err)
	}
	if res.CalendarID != "cal:42" || res.Status != 1 {
		t.Errorf("res = %+v", res)
	}
}

func TestParseEventBody_textAndOpaque(t *testing.T) {
	textBody := wbxml.E(wbxml.PageAirSyncBase, "Body",
		wbxml.E(wbxml.PageAirSyncBase, "Type", wbxml.Text("1")),
		wbxml.E(wbxml.PageAirSyncBase, "Data", wbxml.Text("plain body")),
	)
	out := EventItem{}
	parseEventBody(&out, textBody)
	if out.BodyType != BodyTypePlain || out.Body != "plain body" {
		t.Errorf("text body: got %+v", out)
	}

	opaqueBody := wbxml.E(wbxml.PageAirSyncBase, "Body",
		wbxml.E(wbxml.PageAirSyncBase, "Type", wbxml.Text("4")),
		wbxml.E(wbxml.PageAirSyncBase, "Data", wbxml.Opaque([]byte("MIME bytes"))),
	)
	out2 := EventItem{}
	parseEventBody(&out2, opaqueBody)
	if out2.BodyType != BodyTypeMIME || out2.Body != "MIME bytes" {
		t.Errorf("opaque body: got %+v", out2)
	}
}

func TestParseEventBody_skipsOtherCodepages(t *testing.T) {
	body := wbxml.E(wbxml.PageAirSyncBase, "Body",
		// Wrong codepage — must be ignored.
		wbxml.E(wbxml.PageCalendar, "Type", wbxml.Text("99")),
	)
	out := EventItem{}
	parseEventBody(&out, body)
	if out.BodyType != 0 {
		t.Errorf("expected zero BodyType, got %v", out.BodyType)
	}
}

func TestFormatEASTime(t *testing.T) {
	t1 := time.Date(2024, 1, 15, 12, 34, 56, 0, time.UTC)
	got := formatEASTime(t1)
	want := "2024-01-15T12:34:56.000Z"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
