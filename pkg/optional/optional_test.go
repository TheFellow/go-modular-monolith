package optional_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func TestSome(t *testing.T) {
	t.Parallel()
	v := optional.Some("hi")
	if !v.IsSome() || v.IsNone() {
		t.Fatalf("expected some")
	}
	got, ok := v.Unwrap()
	if !ok || got != "hi" {
		t.Fatalf("unwrap got=%q ok=%v", got, ok)
	}
	if v.Must() != "hi" {
		t.Fatalf("must")
	}
	if v.Or("x") != "hi" {
		t.Fatalf("or")
	}
	if v.OrElse(func() string { return "x" }) != "hi" {
		t.Fatalf("orelse")
	}
}

func TestNone(t *testing.T) {
	t.Parallel()
	var v optional.Value[string] = optional.None[string]()
	if v.IsSome() || !v.IsNone() {
		t.Fatalf("expected none")
	}
	got, ok := v.Unwrap()
	if ok || got != "" {
		t.Fatalf("unwrap got=%q ok=%v", got, ok)
	}
	if v.Or("x") != "x" {
		t.Fatalf("or")
	}
	if v.OrElse(func() string { return "x" }) != "x" {
		t.Fatalf("orelse")
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		}
	}()
	_ = v.Must()
}

func TestMap(t *testing.T) {
	t.Parallel()
	var a optional.Value[int] = optional.Some(2)
	mapped := optional.Map(a, func(x int) string { return "n" })
	if !mapped.IsSome() {
		t.Fatalf("expected some")
	}

	var b optional.Value[int] = optional.None[int]()
	mapped2 := optional.Map(b, func(x int) string { return "n" })
	if !mapped2.IsNone() {
		t.Fatalf("expected none")
	}
}

func TestFlatMap(t *testing.T) {
	t.Parallel()
	var a optional.Value[int] = optional.Some(2)
	out := optional.FlatMap(a, func(x int) optional.Value[string] {
		if x == 2 {
			return optional.Some("ok")
		}
		return optional.None[string]()
	})
	got, ok := out.Unwrap()
	if !ok || got != "ok" {
		t.Fatalf("unexpected out: %#v", out)
	}

	var b optional.Value[int] = optional.None[int]()
	out2 := optional.FlatMap(b, func(x int) optional.Value[string] { return optional.Some("nope") })
	if !out2.IsNone() {
		t.Fatalf("expected none")
	}
}
