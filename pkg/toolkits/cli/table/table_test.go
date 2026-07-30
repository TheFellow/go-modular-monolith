package table

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

type withStringer string

func (w withStringer) String() string { return string(w) }

func TestPrintTable(t *testing.T) {
	t.Parallel()

	type row struct {
		ID    string `table:"ID" json:"id"`
		Name  string `table:"NAME" json:"name"`
		Hide  string `table:"-" json:"hide"`
		Count int    `table:"COUNT" json:"count"`
	}

	var output bytes.Buffer
	err := printTable(&output, []row{{ID: "ing-1", Name: "Vodka", Count: 2}})
	testutil.Ok(t, err)

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	testutil.Equals(t, len(lines), 2)
	testutil.Equals(t, strings.Join(strings.Fields(lines[0]), ","), "ID,NAME,COUNT")
	testutil.Equals(t, strings.Join(strings.Fields(lines[1]), ","), "ing-1,Vodka,2")
}

func TestPrintTable_Empty(t *testing.T) {
	t.Parallel()

	type row struct {
		ID   string `table:"ID" json:"id"`
		Name string `table:"NAME" json:"name"`
	}

	var output bytes.Buffer
	var rows []row
	err := printTable(&output, rows)
	testutil.Ok(t, err)
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	testutil.Equals(t, len(lines), 1)
	testutil.Equals(t, strings.Join(strings.Fields(lines[0]), ","), "ID,NAME")
}

func TestPrintDetail(t *testing.T) {
	t.Parallel()

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

	var output bytes.Buffer
	err := printDetail(&output, item)
	testutil.Ok(t, err)

	out := output.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	testutil.Equals(t, len(lines), 4)
	parseLine := func(line string) (string, string) {
		parts := strings.SplitN(line, ":", 2)
		testutil.Equals(t, len(parts), 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}

	label, value := parseLine(lines[0])
	testutil.Equals(t, label, "ID")
	testutil.Equals(t, value, "ord-1")
	label, value = parseLine(lines[1])
	testutil.Equals(t, label, "Menu ID")
	testutil.Equals(t, value, "mnu-1")
	label, value = parseLine(lines[2])
	testutil.Equals(t, label, "Created At")
	testutil.Equals(t, value, "2025-02-03T04:05:06Z")
	label, value = parseLine(lines[3])
	testutil.Equals(t, label, "Status")
	testutil.Equals(t, value, "pending")
	testutil.IsFalse(t, strings.Contains(out, "Notes:"))
}
