package gosn

import (
	"testing"
)

func TestFilterNoteTitle(t *testing.T) {
	gnuNote := createNote("GNU", "Is not Unix")
	filter := Filter{
		Type:       "Note",
		Key:        "Title",
		Comparison: "==",
		Value:      "GNU",
	}
	itemFilters := ItemFilters{
		Filters:  []Filter{filter},
		MatchAny: true,
	}
	res := applyNoteFilters(*gnuNote, itemFilters, nil)
	if !res {
		t.Error("failed to match note by title")
	}
}

func TestFilterTagTitle(t *testing.T) {
	gnuTag := createTag("GNU")
	filter := Filter{
		Type:       "Tag",
		Key:        "Title",
		Comparison: "==",
		Value:      "GNU",
	}
	itemFilters := ItemFilters{
		Filters:  []Filter{filter},
		MatchAny: true,
	}
	res := applyTagFilters(*gnuTag, itemFilters)
	if !res {
		t.Error("failed to match tag by title")
	}
}

func TestFilterNoteText(t *testing.T) {
	gnuNote := createNote("GNU", "Is not Unix")
	filter := Filter{
		Type:       "Tag",
		Key:        "Text",
		Comparison: "==",
		Value:      "Is not Unix",
	}
	itemFilters := ItemFilters{
		Filters:  []Filter{filter},
		MatchAny: true,
	}
	res := applyTagFilters(*gnuNote, itemFilters)
	if !res {
		t.Error("failed to match note by text")
	}
}

func TestFilterNoteTextByRegex(t *testing.T) {
	gnuNote := createNote("GNU", "Is not Unix")
	filter := Filter{
		Type:       "Tag",
		Key:        "Text",
		Comparison: "~",
		Value:      "^.*Unix",
	}
	itemFilters := ItemFilters{
		Filters:  []Filter{filter},
		MatchAny: true,
	}
	res := applyTagFilters(*gnuNote, itemFilters)
	if !res {
		t.Error("failed to match note by text regex")
	}
}
