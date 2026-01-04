package optional_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func TestSome(t *testing.T) {
	v := optional.NewSome("hi")
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
	var v optional.Value[string] = optional.NewNone[string]()
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
	var a optional.Value[int] = optional.NewSome(2)
	mapped := optional.Map(a, func(x int) string { return "n" })
	if _, ok := mapped.(optional.Some[string]); !ok {
		t.Fatalf("expected some")
	}

	var b optional.Value[int] = optional.NewNone[int]()
	mapped2 := optional.Map(b, func(x int) string { return "n" })
	if _, ok := mapped2.(optional.None[string]); !ok {
		t.Fatalf("expected none")
	}
}

func TestFlatMap(t *testing.T) {
	var a optional.Value[int] = optional.NewSome(2)
	out := optional.FlatMap(a, func(x int) optional.Value[string] {
		if x == 2 {
			return optional.NewSome("ok")
		}
		return optional.NewNone[string]()
	})
	if s, ok := out.(optional.Some[string]); !ok || s.Val != "ok" {
		t.Fatalf("unexpected out: %#v", out)
	}

	var b optional.Value[int] = optional.NewNone[int]()
	out2 := optional.FlatMap(b, func(x int) optional.Value[string] { return optional.NewSome("nope") })
	if _, ok := out2.(optional.None[string]); !ok {
		t.Fatalf("expected none")
	}
}
