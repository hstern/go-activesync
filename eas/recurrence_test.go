// Copyright (C) 2026 Henry Stern
// SPDX-License-Identifier: MIT

package eas

import (
	"testing"
	"time"
)

func TestRecurrence_roundTrip(t *testing.T) {
	r := &Recurrence{
		Type:        RecurrenceWeekly,
		Interval:    1,
		DayOfWeek:   DowMonday | DowWednesday | DowFriday,
		Until:       time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
		Occurrences: 0,
	}
	el := recurrenceToWBXML(r)
	if el == nil {
		t.Fatal("encoded recurrence is nil")
	}
	got := recurrenceFromWBXML(el)
	if got.Type != r.Type {
		t.Errorf("Type: %v vs %v", got.Type, r.Type)
	}
	if got.Interval != r.Interval {
		t.Errorf("Interval: %d vs %d", got.Interval, r.Interval)
	}
	if got.DayOfWeek != r.DayOfWeek {
		t.Errorf("DayOfWeek: %d vs %d", got.DayOfWeek, r.DayOfWeek)
	}
	if !got.Until.Equal(r.Until) {
		t.Errorf("Until: %v vs %v", got.Until, r.Until)
	}
}

func TestException_deletedRoundTrip(t *testing.T) {
	x := Exception{
		ExceptionStartTime: time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
		Deleted:            true,
	}
	el := exceptionToWBXML(x)
	got := parseException(el)
	if !got.Deleted || !got.ExceptionStartTime.Equal(x.ExceptionStartTime) {
		t.Errorf("got %+v", got)
	}
}

func TestException_modifiedRoundTrip(t *testing.T) {
	x := Exception{
		ExceptionStartTime: time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
		Subject:            "Moved!",
		StartTime:          time.Date(2026, 5, 15, 11, 0, 0, 0, time.UTC),
		EndTime:            time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC),
		BusyStatus:         2,
	}
	got := parseException(exceptionToWBXML(x))
	if got.Subject != "Moved!" || got.BusyStatus != 2 {
		t.Errorf("got %+v", got)
	}
	if !got.StartTime.Equal(x.StartTime) || !got.EndTime.Equal(x.EndTime) {
		t.Errorf("times: %v vs %v / %v vs %v", got.StartTime, x.StartTime, got.EndTime, x.EndTime)
	}
}

func TestTimeZone_utcRoundTrip(t *testing.T) {
	enc := EncodeTimeZone(time.UTC)
	tz, err := DecodeTimeZone(enc)
	if err != nil {
		t.Fatal(err)
	}
	if tz.BiasMinutes != 0 || tz.StandardName != "" {
		t.Errorf("UTC tz = %+v", tz)
	}
}

func TestTimeZone_namedZoneRoundTrip(t *testing.T) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skipf("zoneinfo unavailable: %v", err)
	}
	enc := EncodeTimeZone(loc)
	tz, err := DecodeTimeZone(enc)
	if err != nil {
		t.Fatal(err)
	}
	if tz.StandardName != loc.String() {
		t.Errorf("name = %q, want %q", tz.StandardName, loc.String())
	}
	// Bias should be ±420 (PDT/PST). Don't pin which without checking
	// "now" in that zone — Microsoft sign convention is negated minutes.
	if tz.BiasMinutes != 420 && tz.BiasMinutes != 480 {
		t.Errorf("BiasMinutes = %d, want 420 or 480", tz.BiasMinutes)
	}
}

func TestTimeZone_explicitBlobRoundTrip(t *testing.T) {
	in := EASTimeZone{
		BiasMinutes:  300, // -5h offset (EST)
		StandardName: "Eastern Standard Time",
		StandardDate: SystemTime{Year: 0, Month: 11, DayOfWeek: 0, Day: 1, Hour: 2},
		DaylightName: "Eastern Daylight Time",
		DaylightDate: SystemTime{Year: 0, Month: 3, DayOfWeek: 0, Day: 2, Hour: 2},
		DaylightBias: -60,
	}
	out, err := DecodeTimeZone(in.Encode())
	if err != nil {
		t.Fatal(err)
	}
	if out.BiasMinutes != in.BiasMinutes {
		t.Errorf("bias: %d vs %d", out.BiasMinutes, in.BiasMinutes)
	}
	if out.StandardName != in.StandardName {
		t.Errorf("std name: %q vs %q", out.StandardName, in.StandardName)
	}
	if out.DaylightName != in.DaylightName {
		t.Errorf("dst name: %q vs %q", out.DaylightName, in.DaylightName)
	}
	if out.DaylightBias != in.DaylightBias {
		t.Errorf("dst bias: %d vs %d", out.DaylightBias, in.DaylightBias)
	}
	if out.StandardDate != in.StandardDate || out.DaylightDate != in.DaylightDate {
		t.Errorf("system time round-trip mismatch")
	}
}

func TestUTF16RoundTrip(t *testing.T) {
	in := "héllo 世界 🌍"
	enc := utf16Encode(in)
	got := utf16Decode(enc)
	if got != in {
		t.Errorf("utf16 round-trip: %q vs %q", got, in)
	}
}
