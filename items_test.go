package gosn

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Pallinder/go-randomdata"
)

var (
	sInput = SignInInput{
		Email:     os.Getenv("SN_EMAIL"),
		Password:  os.Getenv("SN_PASSWORD"),
		APIServer: os.Getenv("SN_SERVER"),
	}
)

const (
	testParagraph = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec et porttitor metus. " +
		"Suspendisse vulputate lacinia quam in vulputate. Duis vulputate, magna quis efficitur egestas, " +
		"ante libero euismod purus, vel dignissim enim leo et enim. Vivamus egestas magna dolor, id interdum " +
		"nibh volutpat non. Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridiculus " +
		"mus. Quisque vel odio ex. Nunc sollicitudin urna ut lectus semper bibendum."
)

func _createNotes(session Session, input map[string]string) (output PutItemsOutput, err error) {
	for k, v := range input {
		newNote := NewItem()
		newNote.ContentType = "Note"
		createTime := time.Now().Format(timeLayout)
		newNote.CreatedAt = createTime
		newNote.UpdatedAt = createTime
		newNoteContent := &NoteContent{
			Title: k,
			Text:  v,
		}
		newNoteContent.SetUpdateTime(time.Now())
		newNote.Content = newNoteContent

		putItemsInput := PutItemsInput{
			Session: session,
			Items:   []Item{*newNote},
		}
		output, err = PutItems(putItemsInput)
		if err != nil {
			err = fmt.Errorf("PutItems Failed: %v", err)
			return
		}
	}
	return
}

func _createTags(session Session, input []string) (output PutItemsOutput, err error) {
	for _, tt := range input {
		newTag := NewItem()
		newTag.ContentType = "Tag"
		createTime := time.Now().Format(timeLayout)
		newTag.CreatedAt = createTime
		newTag.UpdatedAt = createTime
		newTagContent := &NoteContent{
			Title: tt,
		}
		newTagContent.SetUpdateTime(time.Now())
		newTag.Content = newTagContent

		putItemsInput := PutItemsInput{
			Session: session,
			Items:   []Item{*newTag},
		}
		output, err = PutItems(putItemsInput)
		if err != nil {
			err = fmt.Errorf("PutItems Failed: %v", err)
			return
		}
	}
	return
}

func _deleteAllTagsAndNotes(session Session) (err error) {
	gnf := Filter{
		Type: "Note",
	}
	gtf := Filter{
		Type: "Tag",
	}
	f := ItemFilters{
		Filters:  []Filter{gnf, gtf},
		MatchAny: true,
	}
	gii := GetItemsInput{
		Session: session,
		Filters: f,
	}
	i, err := GetItems(gii)
	if err != nil {
		return
	}
	var toDel []Item
	for x := range i.Items {
		md := i.Items[x]
		md.Deleted = true
		toDel = append(toDel, md)
	}

	putItemsInput := PutItemsInput{
		Session: session,
		Items:   toDel,
	}
	_, err = PutItems(putItemsInput)
	if err != nil {
		err = fmt.Errorf("PutItems Failed: %v", err)
		return
	}
	return
}

func _getItems(session Session, itemFilters ItemFilters) (items []Item, err error) {
	getItemsInput := GetItemsInput{
		Filters: itemFilters,
		Session: session,
	}
	var gio GetItemsOutput
	gio, err = GetItems(getItemsInput)
	if err != nil {
		err = fmt.Errorf("GetItems Failed: %v", err)
		return
	}
	items = gio.Items
	return
}

func createNote(title, text string) *Item {
	note := NewItem()
	content := NewNoteContent()
	content.Title = title
	content.Text = text
	note.ContentType = "Note"
	note.Content = content
	return note
}

func createTag(title string) *Item {
	tag := NewItem()
	content := NewTagContent()
	content.Title = title
	tag.ContentType = "Tag"
	tag.Content = content
	return tag
}

func TestNoteTagging(t *testing.T) {
	// SetDebugLogger(log.Println)

	sOutput, err := SignIn(sInput)
	if err != nil {
		t.Errorf("SignIn Failed - err returned: %v", err)
	}

	// create base notes
	newNotes := genNotes(100, 2)
	if err != nil {
		t.Errorf("SignIn Failed - err returned: %v", err)
	}
	pii := PutItemsInput{
		Session: sOutput.Session,
		Items:   newNotes,
	}
	_, err = PutItems(pii)
	if err != nil {
		t.Errorf(err.Error())
	}

	dogNote := createNote("Dogs", "Can't look up")
	cheeseNote := createNote("Cheese", "Is not a vegetable")
	baconNote := createNote("Bacon", "Goes with everything")
	gnuNote := createNote("GNU", "Is not Unix")
	spiderNote := createNote("Spiders", "Are not welcome")

	animalTag := createTag("Animal Facts")
	foodTag := createTag("Food Facts")

	// tag dog and gnu note with animal tag
	updatedAnimalTagsInput := UpdateItemRefsInput{
		Items: []Item{*animalTag},
		ToRef: []Item{*dogNote, *gnuNote, *spiderNote},
	}
	updatedAnimalTagsOutput := UpdateItemRefs(updatedAnimalTagsInput)
	// confirm new tags both reference dog and gnu notes
	animalNoteUUIDs := []string{
		dogNote.UUID,
		gnuNote.UUID,
		spiderNote.UUID,
	}

	foodNoteUUIDs := []string{
		cheeseNote.UUID,
		baconNote.UUID,
	}

	// tag cheese note with food tag
	updatedFoodTagsInput := UpdateItemRefsInput{
		Items: []Item{*foodTag},
		ToRef: []Item{*cheeseNote, *baconNote},
	}
	updatedFoodTagsOutput := UpdateItemRefs(updatedFoodTagsInput)

	//
	for _, at := range updatedAnimalTagsOutput.Items {
		for _, ref := range at.Content.References() {
			if !stringInSlice(ref.UUID, animalNoteUUIDs, true) {
				t.Error("failed to find an animal note reference")
			}
			if stringInSlice(ref.UUID, foodNoteUUIDs, true) {
				t.Error("found a food note reference")
			}

		}
	}

	for _, ft := range updatedFoodTagsOutput.Items {
		for _, ref := range ft.Content.References() {
			if !stringInSlice(ref.UUID, foodNoteUUIDs, true) {
				t.Error("failed to find an food note reference")
			}
			if stringInSlice(ref.UUID, animalNoteUUIDs, true) {
				t.Error("found an animal note reference")
			}
		}
	}

	// Put Notes and Tags
	var allItems []Item
	allItems = append(allItems, *dogNote, *cheeseNote, *gnuNote)
	allItems = append(allItems, updatedAnimalTagsOutput.Items...)
	allItems = append(allItems, updatedFoodTagsOutput.Items...)

	pii = PutItemsInput{
		Items:   allItems,
		Session: sOutput.Session,
	}
	_, err = PutItems(pii)
	if err != nil {
		t.Errorf("failed to put items: %+v", err)
	}
	getAnimalNotesFilter := Filter{
		Type:       "Note",
		Key:        "TagTitle",
		Comparison: "==",
		Value:      "Animal Facts",
	}
	getAnimalNotesFilters := ItemFilters{
		Filters: []Filter{getAnimalNotesFilter},
	}
	getAnimalNotesInput := GetItemsInput{
		Session: sOutput.Session,
		Filters: getAnimalNotesFilters,
	}
	var getAnimalNotesOutput GetItemsOutput
	getAnimalNotesOutput, err = GetItems(getAnimalNotesInput)
	if err != nil {
		t.Error("failed to retrieve animal notes by tag")
	}
	// check two notes are animal tagged ones
	animalNoteTitles := []string{
		dogNote.Content.GetTitle(),
		gnuNote.Content.GetTitle(),
	}
	if len(getAnimalNotesOutput.Items) != 2 {
		t.Errorf("expected two tags, got: %d", len(getAnimalNotesOutput.Items))
	}
	for _, fn := range getAnimalNotesOutput.Items {
		if !stringInSlice(fn.Content.GetTitle(), animalNoteTitles, true) {
			t.Error("got non animal note based on animal tag")
		}
	}

	// get using regex
	regexFilter := Filter{
		Type:       "Note",
		Comparison: "~",
		Key:        "Text",
		Value:      `not\s(Unix|a vegetable)`,
	}
	regexFilters := ItemFilters{
		Filters: []Filter{regexFilter},
	}
	getNotesInput := GetItemsInput{
		Session: sOutput.Session,
		Filters: regexFilters,
	}
	var getNotesOutput GetItemsOutput
	getNotesOutput, err = GetItems(getNotesInput)
	if err != nil {
		t.Error("failed to retrieve notes using regex")
	}
	// check two notes are animal tagged ones
	expectedNoteTitles := []string{"Cheese", "GNU"}
	if len(getNotesOutput.Items) != len(expectedNoteTitles) {
		t.Errorf("expected two notes, got: %d", len(getNotesOutput.Items))
	}
	for _, fn := range getNotesOutput.Items {
		if !stringInSlice(fn.Content.GetTitle(), expectedNoteTitles, true) {
			t.Errorf("got unexpected result: %s", fn.Content.GetTitle())
		}
	}

	// clean up
	if err := _deleteAllTagsAndNotes(sOutput.Session); err != nil {
		t.Errorf("failed to delete items")
	}

}

func TestSearchNotesByText(t *testing.T) {
	//SetDebugLogger(log.Println)
	sOutput, err := SignIn(sInput)
	if err != nil {
		t.Errorf("SignIn Failed - err returned: %v", err)
	}
	// create two notes
	noteInput := map[string]string{
		"Dog Fact":    "Dogs can't look up",
		"Cheese Fact": "Cheese is not a vegetable",
	}
	if _, err = _createNotes(sOutput.Session, noteInput); err != nil {
		t.Errorf("failed to create notes")
	}
	// find one note by text
	var foundItems []Item
	filterOne := Filter{
		Type:       "Note",
		Key:        "Text",
		Comparison: "contains",
		Value:      "Cheese",
	}
	var itemFilters ItemFilters
	itemFilters.Filters = []Filter{filterOne}
	foundItems, err = _getItems(sOutput.Session, itemFilters)
	if err != nil {
		t.Error(err.Error())
	}
	// check correct items returned
	switch len(foundItems) {
	case 0:
		t.Errorf("no notes returned")
	case 1:
		if foundItems[0].Content.GetTitle() != "Cheese Fact" {
			t.Errorf("incorrect note returned (title mismatch)")
		}
		if !foundItems[0].Content.TextContains("Cheese is not a vegetable", true) {
			t.Errorf("incorrect note returned (text mismatch)")
		}
	default:
		t.Errorf("expected one note but got: %d", len(foundItems))

	}
	// clean up
	if err := _deleteAllTagsAndNotes(sOutput.Session); err != nil {
		t.Errorf("failed to delete items")
	}

}

func TestSearchNotesByRegexTitleFilter(t *testing.T) {
	//SetDebugLogger(log.Println)
	sOutput, err := SignIn(sInput)
	if err != nil {
		t.Errorf("SignIn Failed - err returned: %v", err)
	}
	// create two notes
	noteInput := map[string]string{
		"Dog Fact":    "Dogs can't look up",
		"Cheese Fact": "Cheese is not a vegetable",
	}
	if _, err = _createNotes(sOutput.Session, noteInput); err != nil {
		t.Errorf("failed to create notes")
	}
	// find one note by text
	var foundItems []Item
	filterOne := Filter{
		Type:       "Note",
		Key:        "Title",
		Comparison: "~",
		Value:      "^Do.*",
	}
	var itemFilters ItemFilters
	itemFilters.Filters = []Filter{filterOne}
	foundItems, err = _getItems(sOutput.Session, itemFilters)
	if err != nil {
		t.Error(err.Error())
	}
	// check correct items returned
	switch len(foundItems) {
	case 0:
		t.Errorf("no notes returned")
	case 1:
		if foundItems[0].Content.GetTitle() != "Dog Fact" {
			t.Errorf("incorrect note returned (title mismatch)")
		}
		if !foundItems[0].Content.TextContains("Dogs can't look up", true) {
			t.Errorf("incorrect note returned (text mismatch)")
		}
	default:
		t.Errorf("expected one note but got: %d", len(foundItems))

	}
	// clean up
	if err := _deleteAllTagsAndNotes(sOutput.Session); err != nil {
		t.Errorf("failed to delete items")
	}

}

func TestPutItemsAddSingleNote(t *testing.T) {
	//SetDebugLogger(log.Println)
	sOutput, err := SignIn(sInput)

	if err != nil {
		t.Errorf("SignIn Failed - err returned: %v", err)
	}

	newNoteContent := NoteContent{
		Title:          "TestTitle",
		Text:           testParagraph,
		ItemReferences: nil,
	}
	newNoteContent.SetUpdateTime(time.Now())
	newNote := NewItem()
	newNote.ContentType = "Note"
	createTime := time.Now().Format(timeLayout)
	newNote.CreatedAt = createTime
	newNote.UpdatedAt = createTime
	newNote.Content = &newNoteContent
	putItemsInput := PutItemsInput{
		Items:   []Item{*newNote},
		Session: sOutput.Session,
	}
	var putItemsOutput PutItemsOutput
	putItemsOutput, err = PutItems(putItemsInput)
	if err != nil {
		t.Errorf("PutItems Failed - err returned: %v", err)
	}
	// ### confirm single item saved
	numSaved := len(putItemsOutput.ResponseBody.SavedItems)
	if numSaved != 1 {
		t.Errorf("PutItems Failed - expected 1 item to be created but %d were", numSaved)
	}
	// ### retrieve items and check new item has been persisted
	uuidOfNewItem := putItemsOutput.ResponseBody.SavedItems[0].UUID
	getItemsInput := GetItemsInput{
		Session: sOutput.Session,
	}
	var gio GetItemsOutput
	gio, err = GetItems(getItemsInput)
	if err != nil {
		t.Errorf("failed to get items - err returned: %v", err)
	}
	var foundCreatedItem bool
	for i := range gio.Items {
		if gio.Items[i].UUID == uuidOfNewItem {
			foundCreatedItem = true
			if gio.Items[i].ContentType != "Note" {
				t.Errorf("content type of new item is incorrect - expected: Note got: %s",
					gio.Items[i].ContentType)
			}
			if gio.Items[i].Deleted {
				t.Errorf("deleted status of new item is incorrect - expected: False got: True")
			}
			if gio.Items[i].Content.GetText() != testParagraph {
				t.Errorf("text of new item is incorrect - expected: %s got: %s",
					testParagraph, gio.Items[i].Content.GetText())
			}
		}
	}
	if !foundCreatedItem {
		t.Errorf("failed to get created Item by UUID")
	}

	// clean up
	if err := _deleteAllTagsAndNotes(sOutput.Session); err != nil {
		t.Errorf("failed to delete items")
	}
}

func TestSearchTagsByText(t *testing.T) {
	//SetDebugLogger(log.Println)
	sOutput, signInErr := SignIn(sInput)
	if signInErr != nil {
		t.Errorf("SignIn Failed - err returned: %v", signInErr)
	}
	tagInput := []string{"Rod, Jane", "Zippy, Bungle"}
	var err error
	if _, err = _createTags(sOutput.Session, tagInput); err != nil {
		t.Errorf("failed to create tags")
	}
	// find one note by text
	var foundItems []Item
	filterOne := Filter{
		Type:       "Tag",
		Key:        "Title",
		Comparison: "contains",
		Value:      "Bungle",
	}
	var itemFilters ItemFilters
	itemFilters.Filters = []Filter{filterOne}
	foundItems, err = _getItems(sOutput.Session, itemFilters)
	if err != nil {
		t.Error(err.Error())
	}
	// check correct items returned
	switch len(foundItems) {
	case 0:
		t.Errorf("no tags returned")
	case 1:
		if foundItems[0].Content.GetTitle() != "Zippy, Bungle" {
			t.Errorf("incorrect tag returned (title mismatch)")
		}
	default:
		t.Errorf("expected one tag but got: %d", len(foundItems))

	}
	// clean up
	if err := _deleteAllTagsAndNotes(sOutput.Session); err != nil {
		t.Errorf("failed to delete items")
	}

}

func TestSearchTagsByRegex(t *testing.T) {
	//SetDebugLogger(log.Println)
	sOutput, signInErr := SignIn(sInput)
	if signInErr != nil {
		t.Errorf("SignIn Failed - err returned: %v", signInErr)
	}
	tagInput := []string{"Rod, Jane", "Zippy, Bungle"}
	var err error
	if _, err = _createTags(sOutput.Session, tagInput); err != nil {
		t.Errorf("failed to create tags")
	}
	// find one note by text
	var foundItems []Item
	filterOne := Filter{
		Type:       "Tag",
		Key:        "Title",
		Comparison: "~",
		Value:      "pp",
	}
	var itemFilters ItemFilters
	itemFilters.Filters = []Filter{filterOne}
	foundItems, err = _getItems(sOutput.Session, itemFilters)
	if err != nil {
		t.Error(err.Error())
	}
	// check correct items returned
	switch len(foundItems) {
	case 0:
		t.Errorf("no tags returned")
	case 1:
		if foundItems[0].Content.GetTitle() != "Zippy, Bungle" {
			t.Errorf("incorrect tag returned (title mismatch)")
		}
	default:
		t.Errorf("expected one tag but got: %d", len(foundItems))

	}
	// clean up
	if err := _deleteAllTagsAndNotes(sOutput.Session); err != nil {
		t.Errorf("failed to delete items")
	}

}

func TestCreateAndGet200NotesInBatchesOf50(t *testing.T) {
	newNotes := genNotes(200, 2)
	sOutput, err := SignIn(sInput)
	if err != nil {
		t.Errorf("SignIn Failed - err returned: %v", err)
	}
	pii := PutItemsInput{
		Session: sOutput.Session,
		Items:   newNotes,
	}
	_, err = PutItems(pii)
	if err != nil {
		t.Errorf(err.Error())
	}
	var retrievedNotes []Item
	var cursorToken string
	for {
		giFilter := Filter{
			Type:  "Note",
			Key:   "Deleted",
			Value: "False",
		}
		giFilters := ItemFilters{
			Filters: []Filter{giFilter},
		}
		gii := GetItemsInput{
			Session:     sOutput.Session,
			Filters:     giFilters,
			CursorToken: cursorToken,
			BatchSize:   50,
		}
		var gio GetItemsOutput
		gio, err = GetItems(gii)
		if err != nil {
			t.Error(err)
		}

		retrievedNotes = append(retrievedNotes, gio.Items...)
		if stripLineBreak(gio.Cursor) == "" {
			break
		} else {
			cursorToken = gio.Cursor
		}
	}
	retrievedNotes = DeDupeItems(retrievedNotes)

	if len(retrievedNotes) != 200 {
		t.Errorf("expected 200 items but got %d\n", len(retrievedNotes))
	}

	if err := _deleteAllTagsAndNotes(sOutput.Session); err != nil {
		t.Errorf("failed to delete items")
	}

}

func genRandomText(paragraphs int) string {
	var strBuilder strings.Builder

	for i := 1; i <= paragraphs; i++ {
		strBuilder.WriteString(randomdata.Paragraph())
	}
	return strBuilder.String()
}

func genNotes(num int, textParas int) (notes []Item) {
	for i := 1; i <= num; i++ {
		time.Sleep(3 * time.Millisecond)
		noteContent := &NoteContent{
			Title:          fmt.Sprintf("%d,%s", i, "Title"),
			Text:           fmt.Sprintf("%d,%s", i, genRandomText(textParas)),
			ItemReferences: []ItemReference{},
		}
		noteContent.SetUpdateTime(time.Now())
		newNote := NewItem()
		newNote.ContentType = "Note"
		newNote.Content = noteContent
		notes = append(notes, *newNote)
	}
	return notes
}
