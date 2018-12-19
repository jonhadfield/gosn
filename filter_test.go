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
	gnuNoteUUID := GenUUID()
	animalTagUUID := GenUUID()
	cheeseNoteUUID := GenUUID()
	foodTagUUID := GenUUID()
	sportNoteUUID := GenUUID()

	animalTag := createTag("Animal", animalTagUUID)
	gnuNote := createNote("GNU", "Is not Unix", gnuNoteUUID)
	sportNote := createNote("Sport", "Is dull", sportNoteUUID)

	foodTag := createTag("Food", foodTagUUID)
	cheeseNote := createNote("Cheese", "Is not a vegetable", cheeseNoteUUID)

	gnuRef := ItemReference{
		UUID:        gnuNoteUUID,
		ContentType: "Note",
	}
	animalTag.Content.UpsertReferences([]ItemReference{gnuRef})
	cheeseRef := ItemReference{
		UUID:        cheeseNoteUUID,
		ContentType: "Note",
	}
	foodTag.Content.UpsertReferences([]ItemReference{cheeseRef})

	animalTagUUIDFilter := Filter{
		Type:       "Note",
		Key:        "TagUUID",
		Comparison: "==",
		Value:      animalTagUUID,
	}

	foodTagUUIDFilter := Filter{
		Type:       "Note",
		Key:        "TagUUID",
		Comparison: "==",
		Value:      foodTagUUID,
	}

	animalTagUUIDFilterNegative := Filter{
		Type:       "Note",
		Key:        "TagUUID",
		Comparison: "!=",
		Value:      animalTagUUID,
	}

	animalItemFiltersNegativeMatchAny := ItemFilters{
		Filters:  []Filter{animalTagUUIDFilterNegative},
		MatchAny: true,
	}

	animalItemFiltersNegativeMatchAll := ItemFilters{
		Filters:  []Filter{animalTagUUIDFilterNegative},
		MatchAny: false,
	}

	animalItemFilters := ItemFilters{
		Filters:  []Filter{animalTagUUIDFilter},
		MatchAny: true,
	}
	animalAndFoodItemFiltersAnyTrue := ItemFilters{
		Filters:  []Filter{foodTagUUIDFilter, animalTagUUIDFilter},
		MatchAny: true,
	}
	animalAndFoodItemFiltersAnyFalse := ItemFilters{
		Filters:  []Filter{foodTagUUIDFilter, animalTagUUIDFilter},
		MatchAny: false,
	}
	// try match single animal (success)
	res := applyNoteFilters(*gnuNote, animalItemFilters, []Item{*animalTag})
	assert.True(t, res, "failed to match any note by tag uuid")

	// try match animal note against food tag (failure)
	res = applyNoteFilters(*gnuNote, animalItemFilters, []Item{*foodTag})
	assert.False(t, res, "incorrectly matched note by tag uuid")

	// try against any of multiple filters - match any (success)
	res = applyNoteFilters(*cheeseNote, animalAndFoodItemFiltersAnyTrue, []Item{*animalTag, *foodTag})
	assert.True(t, res, "failed to match cheese note against any of animal or food tag")

	// try against any of multiple filters - match all (failure)
	res = applyNoteFilters(*cheeseNote, animalAndFoodItemFiltersAnyFalse, []Item{*animalTag, *foodTag})
	assert.False(t, res, "incorrectly matched cheese note against both animal and food tag")

	// try against any of multiple filters - match any (failure)
	res = applyNoteFilters(*sportNote, animalAndFoodItemFiltersAnyFalse, []Item{*animalTag, *foodTag})
	assert.False(t, res, "incorrectly matched sport note against animal and food tags")

	// try against any of multiple filters - match any (success)
	res = applyNoteFilters(*gnuNote, animalItemFiltersNegativeMatchAny, []Item{*foodTag})
	assert.True(t, res, "expected true as gnu note should be negative match for food tag")

	// try against any of multiple filters - match all (failure)
	res = applyNoteFilters(*gnuNote, animalItemFiltersNegativeMatchAll, []Item{*foodTag, *animalTag})
	assert.False(t, res, "expected false as gnu note should be negative match for food tag only")

	// try against any of multiple filters - match any (failure)
	res = applyNoteFilters(*gnuNote, animalItemFiltersNegativeMatchAny, []Item{*animalTag})
	assert.False(t, res, "expected gnu note not to match negative animal tag")

	// try against any of multiple filters - don't want note to match any of the food nor animal tags (success)
	res = applyNoteFilters(*gnuNote, animalItemFiltersNegativeMatchAny, []Item{*foodTag, *animalTag})
	assert.False(t, res, "wanted negative match against animal tag")

	// try against any of multiple filters - match all (failure)
	res = applyNoteFilters(*gnuNote, animalItemFiltersNegativeMatchAll, []Item{*animalTag, *foodTag})
	assert.False(t, res, "expected gnu note not to match negative animal tag")

	// try against any of multiple filters - match all (success)
	res = applyNoteFilters(*gnuNote, animalItemFiltersNegativeMatchAll, []Item{*foodTag})
	assert.True(t, res, "expected gnu note to negative match food tag")
}

func TestFilterNoteByTagTitle(t *testing.T) {
	gnuNoteUUID := GenUUID()
	animalTagUUID := GenUUID()
	cheeseNoteUUID := GenUUID()
	foodTagUUID := GenUUID()
	sportNoteUUID := GenUUID()

	animalTag := createTag("Animal", animalTagUUID)
	gnuNote := createNote("GNU", "Is not Unix", gnuNoteUUID)
	sportNote := createNote("Sport", "Is dull", sportNoteUUID)

	foodTag := createTag("Food", foodTagUUID)
	cheeseNote := createNote("Cheese", "Is not a vegetable", cheeseNoteUUID)

	gnuRef := ItemReference{
		UUID:        gnuNoteUUID,
		ContentType: "Note",
	}
	animalTag.Content.UpsertReferences([]ItemReference{gnuRef})
	cheeseRef := ItemReference{
		UUID:        cheeseNoteUUID,
		ContentType: "Note",
	}
	foodTag.Content.UpsertReferences([]ItemReference{cheeseRef})

	animalTagUUIDFilter := Filter{
		Type:       "Note",
		Key:        "TagTitle",
		Comparison: "==",
		Value:      "Animal",
	}

	foodTagUUIDFilter := Filter{
		Type:       "Note",
		Key:        "TagTitle",
		Comparison: "==",
		Value:      "Food",
	}

	animalTagUUIDFilterNegative := Filter{
		Type:       "Note",
		Key:        "TagUUID",
		Comparison: "!=",
		Value:      animalTagUUID,
	}

	animalItemFiltersNegativeMatchAny := ItemFilters{
		Filters:  []Filter{animalTagUUIDFilterNegative},
		MatchAny: true,
	}

	animalItemFiltersNegativeMatchAll := ItemFilters{
		Filters:  []Filter{animalTagUUIDFilterNegative},
		MatchAny: false,
	}

	animalItemFilters := ItemFilters{
		Filters:  []Filter{animalTagUUIDFilter},
		MatchAny: true,
	}
	animalAndFoodItemFiltersAnyTrue := ItemFilters{
		Filters:  []Filter{foodTagUUIDFilter, animalTagUUIDFilter},
		MatchAny: true,
	}
	animalAndFoodItemFiltersAnyFalse := ItemFilters{
		Filters:  []Filter{foodTagUUIDFilter, animalTagUUIDFilter},
		MatchAny: false,
	}
	// try match single animal (success)
	res := applyNoteFilters(*gnuNote, animalItemFilters, []Item{*animalTag})
	assert.True(t, res, "failed to match any note by tag title")

	// try match animal note against food tag (failure)
	res = applyNoteFilters(*gnuNote, animalItemFilters, []Item{*foodTag})
	assert.False(t, res, "incorrectly matched note by tag title")

	// try against any of multiple filters - match any (success)
	res = applyNoteFilters(*cheeseNote, animalAndFoodItemFiltersAnyTrue, []Item{*animalTag, *foodTag})
	assert.True(t, res, "failed to match cheese note against any of animal or food tag")

	// try against any of multiple filters - match all (failure)
	res = applyNoteFilters(*cheeseNote, animalAndFoodItemFiltersAnyFalse, []Item{*animalTag, *foodTag})
	assert.False(t, res, "incorrectly matched cheese note against both animal and food tag")

	// try against any of multiple filters - match any (failure)
	res = applyNoteFilters(*sportNote, animalAndFoodItemFiltersAnyFalse, []Item{*animalTag, *foodTag})
	assert.False(t, res, "incorrectly matched sport note against animal and food tags")

	// try against any of multiple filters - match any (success)
	res = applyNoteFilters(*gnuNote, animalItemFiltersNegativeMatchAny, []Item{*foodTag})
	assert.True(t, res, "expected true as gnu note should be negative match for food tag")

	// try against any of multiple filters - match all (failure)
	res = applyNoteFilters(*gnuNote, animalItemFiltersNegativeMatchAll, []Item{*foodTag, *animalTag})
	assert.False(t, res, "expected false as gnu note should be negative match for food tag only")

	// try against any of multiple filters - match any (failure)
	res = applyNoteFilters(*gnuNote, animalItemFiltersNegativeMatchAny, []Item{*animalTag})
	assert.False(t, res, "expected gnu note not to match negative animal tag")

	// try against any of multiple filters - don't want note to match any of the food nor animal tags (success)
	res = applyNoteFilters(*gnuNote, animalItemFiltersNegativeMatchAny, []Item{*foodTag, *animalTag})
	assert.False(t, res, "wanted negative match against animal tag")

	// try against any of multiple filters - match all (failure)
	res = applyNoteFilters(*gnuNote, animalItemFiltersNegativeMatchAll, []Item{*animalTag, *foodTag})
	assert.False(t, res, "expected gnu note not to match negative animal tag")

	// try against any of multiple filters - match all (success)
	res = applyNoteFilters(*gnuNote, animalItemFiltersNegativeMatchAll, []Item{*foodTag})
	assert.True(t, res, "expected gnu note to negative match food tag")
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
