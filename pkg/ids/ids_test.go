package ids

import (
	"bytes"
	"testing"
	"time"

	"github.com/cedar-policy/cedar-go"
	"github.com/segmentio/ksuid"
)

func TestNew_PrefixAndParse(t *testing.T) {
	t.Parallel()

	uid, err := New(cedar.EntityType("Mixology::Drink"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if got, want := uid.Type, cedar.EntityType("Mixology::Drink"); got != want {
		t.Fatalf("unexpected entity type: got %q want %q", got, want)
	}

	idStr := string(uid.ID)
	if len(idStr) == 0 || idStr[:4] != "drk-" {
		t.Fatalf("expected drk- prefix, got %q", idStr)
	}

	parsed, err := Parse(idStr)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if parsed.IsNil() {
		t.Fatalf("expected non-nil ksuid")
	}

	ts, err := Time(idStr)
	if err != nil {
		t.Fatalf("Time: %v", err)
	}
	if ts.IsZero() {
		t.Fatalf("expected non-zero time")
	}
}

func TestNew_UnknownEntityType_DerivesPrefix(t *testing.T) {
	t.Parallel()

	uid, err := New(cedar.EntityType("Mixology::Widget"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	idStr := string(uid.ID)
	if len(idStr) == 0 || idStr[:4] != "wid-" {
		t.Fatalf("expected wid- prefix, got %q", idStr)
	}
}

func TestParseAndTime_LexicographicSort(t *testing.T) {
	t.Parallel()

	t0 := time.Unix(1700000000, 0).UTC()
	t1 := t0.Add(time.Second).UTC()
	payload := bytes.Repeat([]byte{0x7f}, 16)

	k0, err := ksuid.FromParts(t0, payload)
	if err != nil {
		t.Fatalf("ksuid.FromParts(t0): %v", err)
	}
	k1, err := ksuid.FromParts(t1, payload)
	if err != nil {
		t.Fatalf("ksuid.FromParts(t1): %v", err)
	}

	id0 := "drk-" + k0.String()
	id1 := "drk-" + k1.String()

	if !(id0 < id1) {
		t.Fatalf("expected lexicographic sort by time: %q < %q", id0, id1)
	}

	got0, err := Parse(id0)
	if err != nil {
		t.Fatalf("Parse(id0): %v", err)
	}
	got1, err := Parse(id1)
	if err != nil {
		t.Fatalf("Parse(id1): %v", err)
	}
	if got0 != k0 || got1 != k1 {
		t.Fatalf("unexpected parse results: got0=%s got1=%s", got0, got1)
	}

	ts0, err := Time(id0)
	if err != nil {
		t.Fatalf("Time(id0): %v", err)
	}
	ts1, err := Time(id1)
	if err != nil {
		t.Fatalf("Time(id1): %v", err)
	}
	if !ts0.Equal(t0) || !ts1.Equal(t1) {
		t.Fatalf("unexpected times: ts0=%s ts1=%s", ts0, ts1)
	}
}
