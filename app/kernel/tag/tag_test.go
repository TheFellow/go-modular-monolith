package tag_test

import (
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  tag.Tag
	}{
		{"key only", "seasonal", tag.Tag{Key: "seasonal"}},
		{"key and value", "region=west", tag.Tag{Key: "region", Value: "west"}},
		{"first separator wins", "source=vendor=preferred", tag.Tag{Key: "source", Value: "vendor=preferred"}},
		{"outer whitespace is trimmed", "  Audience = Internal Users  ", tag.Tag{Key: "Audience", Value: "Internal Users"}},
		{"case is preserved", "Region=West", tag.Tag{Key: "Region", Value: "West"}},
		{"empty value is key only", "featured=", tag.Tag{Key: "featured"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tag.Parse(tt.value)
			testutil.Ok(t, err)
			testutil.Equals(t, got, tt.want)
			testutil.Equals(t, got.String(), tt.want.String())
		})
	}
}

func TestInvalidTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"empty key with value", "=west"},
		{"key too long", strings.Repeat("k", tag.MaxKeyLength+1)},
		{"value too long", "key=" + strings.Repeat("v", tag.MaxValueLength+1)},
		{"invalid UTF-8 key", string([]byte{0xff})},
		{"invalid UTF-8 value", "key=" + string([]byte{0xff})},
		{"control character in key", "line\nbreak=value"},
		{"control character in value", "key=line\nbreak"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := tag.Parse(tt.value)
			testutil.ErrorIsInvalid(t, err)
		})
	}
}

func TestNewRejectsAmbiguousKey(t *testing.T) {
	t.Parallel()

	_, err := tag.New("key=value", "another")
	testutil.ErrorIsInvalid(t, err)
}

func TestTagsUpsertAndRemove(t *testing.T) {
	t.Parallel()

	tags := tag.Tags{
		{Key: "region", Value: "west"},
		{Key: "featured"},
	}
	tags = tags.Upsert(tag.Tag{Key: "region", Value: "east"})
	tags = tags.Upsert(tag.Tag{Key: "audience", Value: "members"})
	testutil.Equals(t, tags.Strings(), []string{"audience=members", "featured", "region=east"})

	tags = tags.Remove(" region ")
	tags = tags.Remove("missing")
	testutil.Equals(t, tags.Strings(), []string{"audience=members", "featured"})
}

func TestTagsMapRoundTrip(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"region":   "west",
		"featured": "",
		"Audience": "Members",
	}
	tags, err := tag.FromMap(values)
	testutil.Ok(t, err)
	testutil.Equals(t, tags.Strings(), []string{"Audience=Members", "featured", "region=west"})
	testutil.Equals(t, tags.Map(), values)

	values["region"] = "changed"
	testutil.Equals(t, tags.Map()["region"], "west")
}

func TestFromMapRejectsKeysThatCollideAfterTrimming(t *testing.T) {
	t.Parallel()

	_, err := tag.FromMap(map[string]string{"region": "west", " region ": "east"})
	testutil.ErrorIsInvalid(t, err)
}

func TestTagsValidate(t *testing.T) {
	t.Parallel()

	testutil.Ok(t, tag.Tags{{Key: "featured"}, {Key: "region", Value: "west"}}.Validate())
	testutil.ErrorIsInvalid(t, tag.Tags{{Key: "region"}, {Key: "region", Value: "west"}}.Validate())
	testutil.ErrorIsInvalid(t, tag.Tags{{Key: " untrimmed"}}.Validate())
}
