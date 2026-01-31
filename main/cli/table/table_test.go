package table

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

type withStringer string

func (w withStringer) String() string { return string(w) }

func captureOutput(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = orig
	}()

	callErr := fn()
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String(), callErr
}

func TestPrintTable(t *testing.T) {
	type row struct {
		ID    string `table:"ID" json:"id"`
		Name  string `table:"NAME" json:"name"`
		Hide  string `table:"-" json:"hide"`
		Count int    `table:"COUNT" json:"count"`
	}

	out, err := captureOutput(t, func() error {
		return PrintTable([]row{{ID: "ing-1", Name: "Vodka", Count: 2}})
	})
	if err != nil {
		t.Fatalf("PrintTable: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if got := strings.Fields(lines[0]); strings.Join(got, ",") != "ID,NAME,COUNT" {
		t.Fatalf("unexpected header: %q", lines[0])
	}
	if got := strings.Fields(lines[1]); strings.Join(got, ",") != "ing-1,Vodka,2" {
		t.Fatalf("unexpected row: %q", lines[1])
	}
}

func TestPrintTable_Empty(t *testing.T) {
	type row struct {
		ID   string `table:"ID" json:"id"`
		Name string `table:"NAME" json:"name"`
	}

	out, err := captureOutput(t, func() error {
		var rows []row
		return PrintTable(rows)
	})
	if err != nil {
		t.Fatalf("PrintTable: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if got := strings.Fields(lines[0]); strings.Join(got, ",") != "ID,NAME" {
		t.Fatalf("unexpected header: %q", lines[0])
	}
}

func TestPrintDetail(t *testing.T) {
	type detail struct {
		ID        string       `json:"id"`
		MenuID    string       `json:"menu_id"`
		CreatedAt time.Time    `json:"created_at"`
		Status    withStringer `json:"status"`
		Notes     string       `json:"notes,omitempty"`
	}

	item := detail{
		ID:        "ord-1",
		MenuID:    "mnu-1",
		CreatedAt: time.Date(2025, 2, 3, 4, 5, 6, 0, time.UTC),
		Status:    "pending",
	}

	out, err := captureOutput(t, func() error {
		return PrintDetail(item)
	})
	if err != nil {
		t.Fatalf("PrintDetail: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}
	parseLine := func(line string) (string, string) {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			t.Fatalf("unexpected line format: %q", line)
		}
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}

	label, value := parseLine(lines[0])
	if label != "ID" || value != "ord-1" {
		t.Fatalf("unexpected id line: %q", lines[0])
	}
	label, value = parseLine(lines[1])
	if label != "Menu ID" || value != "mnu-1" {
		t.Fatalf("unexpected menu id line: %q", lines[1])
	}
	label, value = parseLine(lines[2])
	if label != "Created At" || value != "2025-02-03T04:05:06Z" {
		t.Fatalf("unexpected created at line: %q", lines[2])
	}
	label, value = parseLine(lines[3])
	if label != "Status" || value != "pending" {
		t.Fatalf("unexpected status line: %q", lines[3])
	}
	if strings.Contains(out, "Notes:") {
		t.Fatalf("expected omitempty notes to be skipped")
	}
}
