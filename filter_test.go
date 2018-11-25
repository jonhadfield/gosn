package gosn

import (
	"github.com/stretchr/testify/assert"
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
	assert.True(t, res, "failed to match note by title")
}

func TestFilterNoteTitleContains(t *testing.T) {
	gnuNote := createNote("GNU", "Is not Unix")
	filter := Filter{
		Type:       "Note",
		Key:        "Title",
		Comparison: "contains",
		Value:      "N",
	}
	itemFilters := ItemFilters{
		Filters:  []Filter{filter},
		MatchAny: true,
	}
	res := applyNoteFilters(*gnuNote, itemFilters, nil)
	assert.True(t, res, "failed to match note by title contains")
}

func TestFilterNoteText(t *testing.T) {
	gnuNote := createNote("GNU", "Is not Unix")
	filter := Filter{
		Type:       "Note",
		Key:        "Text",
		Comparison: "==",
		Value:      "Is not Unix",
	}
	itemFilters := ItemFilters{
		Filters:  []Filter{filter},
		MatchAny: true,
	}
	res := applyNoteFilters(*gnuNote, itemFilters, nil)
	assert.True(t, res, "failed to match note by text")
}

func TestFilterNoteTextContains(t *testing.T) {
	gnuNote := createNote("GNU", "Is not Unix")
	filter := Filter{
		Type:       "Note",
		Key:        "Text",
		Comparison: "contains",
		Value:      "Unix",
	}
	itemFilters := ItemFilters{
		Filters:  []Filter{filter},
		MatchAny: true,
	}
	res := applyNoteFilters(*gnuNote, itemFilters, nil)
	assert.True(t, res, "failed to match note by title contains")
}

func TestFilterNoteTitleNotEqualTo(t *testing.T) {
	gnuNote := createNote("GNU", "Is not Unix")
	filter := Filter{
		Type:       "Note",
		Key:        "Title",
		Comparison: "!=",
		Value:      "Potato",
	}
	itemFilters := ItemFilters{
		Filters:  []Filter{filter},
		MatchAny: true,
	}
	res := applyNoteFilters(*gnuNote, itemFilters, nil)
	assert.True(t, res, "failed to match note by negative title match")
}

func TestFilterNoteTextNotEqualTo(t *testing.T) {
	gnuNote := createNote("GNU", "Is not Unix")
	filter := Filter{
		Type:       "Note",
		Key:        "Text",
		Comparison: "!=",
		Value:      "Potato",
	}
	itemFilters := ItemFilters{
		Filters:  []Filter{filter},
		MatchAny: true,
	}
	res := applyNoteFilters(*gnuNote, itemFilters, nil)
	assert.True(t, res, "failed to match note by negative text match")
}

func TestFilterNoteTextByRegex(t *testing.T) {
	gnuNote := createNote("GNU", "Is not Unix")
	filter := Filter{
		Type:       "Note",
		Key:        "Text",
		Comparison: "~",
		Value:      "^.*Unix",
	}
	itemFilters := ItemFilters{
		Filters:  []Filter{filter},
		MatchAny: true,
	}
	res := applyNoteFilters(*gnuNote, itemFilters, nil)
	assert.True(t, res, "failed to match note by text regex")
}

func TestFilterNoteTitleByRegex(t *testing.T) {
	gnuNote := createNote("GNU", "Is not Unix")
	filter := Filter{
		Type:       "Tag",
		Key:        "Title",
		Comparison: "~",
		Value:      "^.N.$",
	}
	itemFilters := ItemFilters{
		Filters:  []Filter{filter},
		MatchAny: true,
	}
	res := applyTagFilters(*gnuNote, itemFilters)
	assert.True(t, res, "failed to match note by title text regex")
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
	assert.True(t, res, "failed to match tag by title")
}

func TestFilterTagTitleByRegex(t *testing.T) {
	gnuTag := createTag("GNU")
	filter := Filter{
		Type:       "Tag",
		Key:        "Title",
		Comparison: "~",
		Value:      "^.*U$",
	}
	itemFilters := ItemFilters{
		Filters:  []Filter{filter},
		MatchAny: true,
	}
	res := applyTagFilters(*gnuTag, itemFilters)
	assert.True(t, res, "failed to match tag by title regex")
}

func TestFilterTagTitleByNotEqualTo(t *testing.T) {
	gnuTag := createTag("GNU")
	filter := Filter{
		Type:       "Tag",
		Key:        "Title",
		Comparison: "!=",
		Value:      "potato",
	}
	itemFilters := ItemFilters{
		Filters:  []Filter{filter},
		MatchAny: true,
	}
	res := applyTagFilters(*gnuTag, itemFilters)
	assert.True(t, res, "failed to match tag by title negative title match")
}
