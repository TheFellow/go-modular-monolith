package models

import "testing"

func TestMenuValidateAcceptsWhitespacePaddedNameWithoutNormalizing(t *testing.T) {
	t.Parallel()

	menu := Menu{
		Name:   "  Dinner Service  ",
		Status: MenuStatusDraft,
	}

	if err := menu.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if menu.Name != "  Dinner Service  " {
		t.Fatalf("Validate() normalized name to %q", menu.Name)
	}
}
