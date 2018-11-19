package gosn

import (
	"testing"
)

//func getTestAnimalNotes() []Item {
//	dogNote := createNote("Dogs", "Can't look up")
//	gnuNote := createNote("GNU", "Is not Unix")
//	spiderNote := createNote("Spiders", "Are not welcome")
//	return []Item{*dogNote, *gnuNote, *spiderNote}
//}
//
//func getTestFoodNotes() []Item {
//	cheeseNote := createNote("Cheese", "Is not a vegetable")
//	baconNote := createNote("Bacon", "Goes with everything")
//	return []Item{*cheeseNote, *baconNote}
//}

func TestFilterNoteTitle(t *testing.T) {
	gnuNote := createNote("GNU", "Is not Unix")
	filter := Filter{
		Type:"Note",
		Key: "Title",
		Comparison:"==",
		Value: "GNU",
	}
	itemFilters := ItemFilters{
		Filters: []Filter{filter},
		MatchAny:true,
	}
	res := applyNoteFilters(*gnuNote, itemFilters, nil)
	if ! res {
		t.Error("failed to match note by title")
	}
}