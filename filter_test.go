package gosn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterNoteTitle(t *testing.T) {
	gnuNote := createNote("GNU", "Is not Unix", "")
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

func TestFilterNoteUUID(t *testing.T) {
	uuid := GenUUID()
	gnuNote := createNote("GNU", "Is not Unix", uuid)
	filter := Filter{
		Type:       "Note",
		Key:        "UUID",
		Comparison: "==",
		Value:      uuid,
	}
	itemFilters := ItemFilters{
		Filters:  []Filter{filter},
		MatchAny: true,
	}
	res := applyNoteFilters(*gnuNote, itemFilters, nil)
	assert.True(t, res, "failed to match note by uuid")
}

func TestFilterNoteByTagUUID(t *testing.T) {
	nUUID := GenUUID()
	tUUID := GenUUID()
	animalTag := createTag("Animal", tUUID)
	gnuNote := createNote("GNU", "Is not Unix", nUUID)
	ref := ItemReference{
		UUID:        nUUID,
		ContentType: "Note",
	}
	animalTag.Content.UpsertReferences([]ItemReference{ref})

	filter := Filter{
		Type:       "Note",
		Key:        "TagUUID",
		Comparison: "==",
		Value:      tUUID,
	}
	itemFilters := ItemFilters{
		Filters:  []Filter{filter},
		MatchAny: true,
	}
	res := applyNoteFilters(*gnuNote, itemFilters, []Item{*animalTag})
	assert.True(t, res, "failed to match note by tag uuid")
}

func TestFilterNoteTitleContains(t *testing.T) {
	gnuNote := createNote("GNU", "Is not Unix", "")
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
	gnuNote := createNote("GNU", "Is not Unix", "")
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
	gnuNote := createNote("GNU", "Is not Unix", "")
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
	gnuNote := createNote("GNU", "Is not Unix", "")
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
	gnuNote := createNote("GNU", "Is not Unix", "")
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
	gnuNote := createNote("GNU", "Is not Unix", "")
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
	gnuNote := createNote("GNU", "Is not Unix", "")
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
	gnuTag := createTag("GNU", "")
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

func TestFilterTagUUID(t *testing.T) {
	uuid := GenUUID()
	gnuTag := createTag("GNU", uuid)
	filter := Filter{
		Type:       "Tag",
		Key:        "UUID",
		Comparison: "==",
		Value:      uuid,
	}
	itemFilters := ItemFilters{
		Filters:  []Filter{filter},
		MatchAny: true,
	}
	res := applyTagFilters(*gnuTag, itemFilters)
	assert.True(t, res, "failed to match tag by uuid")
}

func TestFilterTagTitleByRegex(t *testing.T) {
	gnuTag := createTag("GNU", "")
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
	gnuTag := createTag("GNU", "")
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
