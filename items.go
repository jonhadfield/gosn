package gosn

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/matryer/try.v1"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Item describes a decrypted item
type Item struct {
	UUID        string
	Content     ClientStructure
	ContentType string
	Deleted     bool
	CreatedAt   string
	UpdatedAt   string
	ContentSize int
}

// returns a new, typeless item
func newItem() *Item {
	now := time.Now().Format(timeLayout)
	return &Item{
		UUID:      GenUUID(),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewNote returns an Item of type Note without content
func NewNote() *Item {
	item := newItem()
	item.ContentType = "Note"
	return item
}

// NewTag returns an Item of type Tag without content
func NewTag() *Item {
	item := newItem()
	item.ContentType = "Tag"
	return item
}

// NewNoteContent returns an empty Note content instance
func NewNoteContent() *NoteContent {
	c := &NoteContent{}
	c.SetUpdateTime(time.Now())
	return c
}

// NewTagContent returns an empty Tag content instance
func NewTagContent() *TagContent {
	c := &TagContent{}
	c.SetUpdateTime(time.Now())
	return c
}

// ClientStructure defines behaviour of an Item's content entry
type ClientStructure interface {
	References() []ItemReference
	// update or insert item references
	UpsertReferences(input []ItemReference)
	// set references
	SetReferences(input []ItemReference)
	// return title
	GetTitle() string
	// set title
	SetTitle(input string)
	// set text
	SetText(input string)
	// return text
	GetText() string
	// get last update time
	GetUpdateTime() (time.Time, error)
	// set last update time
	SetUpdateTime(time.Time)
	// get appdata
	GetAppData() AppDataContent
	// set appdata
	SetAppData(data AppDataContent)
}

type syncResponse struct {
	Items       []encryptedItem `json:"retrieved_items"`
	SavedItems  []encryptedItem `json:"saved_items"`
	Unsaved     []encryptedItem `json:"unsaved"`
	SyncToken   string          `json:"sync_token"`
	CursorToken string          `json:"cursor_token"`
}

// AppTagConfig defines expected configuration structure for making Tag related operations
type AppTagConfig struct {
	Email    string
	Token    string
	FindText string
	FindTag  string
	NewTags  []string
	Debug    bool
}

// GetItemsInput defines the input for retrieving items
type GetItemsInput struct {
	Session     Session
	SyncToken   string
	CursorToken string
	Filters     ItemFilters
	BatchSize   int // number of items to retrieve
	PageSize    int // override default number of items to request with each sync call
}

// GetItemsOutput defines the output from retrieving items
// It contains slices of items based on their state
// see: https://standardfile.org/ for state details
type GetItemsOutput struct {
	Items      []Item // items new or modified since last sync
	SavedItems []Item // dirty items needing resolution
	Unsaved    []Item // items not saved during sync
	SyncToken  string
	Cursor     string
}

const retryScaleFactor = 0.25

func resizeForRetry(in *GetItemsInput) {
	switch {
	case in.BatchSize != 0:
		in.BatchSize = int(math.Ceil(float64(in.BatchSize) * retryScaleFactor))
	case in.PageSize != 0:
		in.PageSize = int(math.Ceil(float64(in.PageSize) * retryScaleFactor))
	default:
		in.PageSize = int(math.Ceil(float64(PageSize) * retryScaleFactor))
	}
}

// GetItems retrieves items from the API using optional filters
func GetItems(input GetItemsInput) (output GetItemsOutput, err error) {
	funcName := funcNameOutputStart + "GetItems" + funcNameOutputEnd

	// retry logic is to handle responses that are too large
	// so we can reduce number we retrieve with each sync request
	rErr := try.Do(func(attempt int) (bool, error) {
		var rErr error
		output, rErr = getItems(input)
		if rErr != nil && strings.Contains(strings.ToLower(rErr.Error()), "too large") {
			initialSize := input.PageSize
			resizeForRetry(&input)
			debug(funcName, fmt.Sprintf("failed to retrieve %d items "+
				"at a time so reducing to %d", initialSize, input.PageSize))
		}
		return attempt < 3, rErr
	})
	if rErr != nil {
		log.Fatalln("error:", err)
	}

	// strip any duplicates (https://github.com/standardfile/rails-engine/issues/5)
	output.DeDupe()
	// filter results if provided
	if len(input.Filters.Filters) > 0 {
		output.Items = filterItems(output.Items, input.Filters)
	}
	debug(funcName, fmt.Errorf("sync token: %+v", stripLineBreak(output.SyncToken)))
	return
}

// PutItemsInput defines the input used to put items
type PutItemsInput struct {
	Items     []Item
	SyncToken string
	Session   Session
}

// PutItemsOutput defines the output from putting items
type PutItemsOutput struct {
	ResponseBody syncResponse
}

func validateInput(input PutItemsInput) error {
	var updatedTime time.Time
	var err error
	// TODO finish item validation
	for _, inputItem := range input.Items {
		// validate content if being added
		if !inputItem.Deleted {
			if stringInSlice(inputItem.ContentType, []string{"Tag", "Note"}, true) {
				updatedTime, err = inputItem.Content.GetUpdateTime()
				switch {
				case inputItem.Content.GetTitle() == "":
					err = fmt.Errorf("failed to create \"%s\" due to missing title: \"%s\"",
						inputItem.ContentType, inputItem.UUID)
				case updatedTime.IsZero():
					err = fmt.Errorf("failed to create \"%s\" due to missing content updated time: \"%s\"",
						inputItem.ContentType, inputItem.Content.GetTitle())
				case inputItem.CreatedAt == "":
					err = fmt.Errorf("failed to create \"%s\" due to missing created at date: \"%s\"",
						inputItem.ContentType, inputItem.Content.GetTitle())
				}
				if err != nil {
					return err
				}
			}
		}
	}
	return err
}

// PutItems validates and then syncs items via API
func PutItems(input PutItemsInput) (output PutItemsOutput, err error) {
	funcName := funcNameOutputStart + "PutItems" + funcNameOutputEnd
	debug(funcName, fmt.Sprintf("putting %d items", len(input.Items)))

	err = validateInput(input)
	if err != nil {
		return
	}

	var encryptedItems []encryptedItem
	encryptedItems, err = encryptItems(input.Items, input.Session.Mk, input.Session.Ak)
	if err != nil {
		return
	}

	// for each page size, send to push and get response
	syncToken := stripLineBreak(input.SyncToken)
	var savedItems []encryptedItem

	// put items in big chunks, default being page size
	for x := 0; x < len(encryptedItems); x += PageSize {
		var finalChunk bool
		var lastItemInChunkIndex int
		// if current big chunk > num encrypted items then it's the last
		if x+PageSize >= len(encryptedItems) {
			lastItemInChunkIndex = len(encryptedItems) - 1
			finalChunk = true
		} else {
			lastItemInChunkIndex = x + PageSize
		}
		debug(funcName, fmt.Sprintf("putting items: %d to %d", x+1, lastItemInChunkIndex+1))

		bigChunkSize := (lastItemInChunkIndex - x) + 1
		fullChunk := encryptedItems[x : lastItemInChunkIndex+1]
		var subChunkStart, subChunkEnd int
		subChunkStart = x
		subChunkEnd = lastItemInChunkIndex
		// initialise running total
		totalPut := 0
		// keep trying to push chunk of encrypted items in reducing subChunk sizes until it succeeds
		maxAttempts := 20
		try.MaxRetries = 20
		for {
			rErr := try.Do(func(attempt int) (bool, error) {
				var rErr error
				// if chunk is too big to put then try with smaller chunk
				var encItemJSON []byte
				itemsToPut := encryptedItems[subChunkStart : subChunkEnd+1]
				encItemJSON, _ = json.Marshal(itemsToPut)
				var s []encryptedItem
				s, syncToken, rErr = putChunk(input.Session, encItemJSON)
				if rErr != nil && strings.Contains(strings.ToLower(rErr.Error()), "too large") {
					subChunkEnd = resizePutForRetry(subChunkStart, subChunkEnd, len(encItemJSON))
				}
				if rErr == nil {
					savedItems = append(savedItems, s...)
					totalPut += len(itemsToPut)
				}
				debug(funcName, fmt.Sprintf("attempt: %d of %d", attempt, maxAttempts))
				return attempt < maxAttempts, rErr
			})
			if rErr != nil {
				err = errors.New("failed to put all items")
				return
			}

			// if it's not the last of the chunk then continue with next subChunk
			if totalPut < bigChunkSize {
				subChunkStart = subChunkEnd + 1
				subChunkEnd = lastItemInChunkIndex
				continue
			}

			// if it's last of the full chunk, then break
			if len(fullChunk) == lastItemInChunkIndex {
				break
			}

			if totalPut == len(fullChunk) {
				break
			}

		} // end infinite for loop for subset
		if finalChunk {
			break
		}
	} // end looping through encrypted items

	output.ResponseBody.SyncToken = syncToken
	output.ResponseBody.SavedItems = savedItems

	return
}

func resizePutForRetry(start, end, numBytes int) int {
	preShrink := end
	// reduce to 90%
	multiplier := 0.90
	// if size is over 2M then be more aggressive and half
	if numBytes > 2000000 {
		multiplier = 0.50
	}
	end = int(math.Ceil(float64(end) * multiplier))
	if end <= start {
		end = start + 1
	}
	if preShrink == end && preShrink > 1 {
		end--
	}
	return end
}

func putChunk(session Session, encItemJSON []byte) (savedItems []encryptedItem, syncToken string, err error) {
	reqBody := []byte(`{"items":` + string(encItemJSON) +
		`,"sync_token":"` + stripLineBreak(syncToken) + `"}`)
	var syncResp *http.Response
	syncResp, err = makeSyncRequest(session, reqBody)
	if err != nil {
		return
	}
	switch syncResp.StatusCode {
	case 413:
		err = errors.New("payload too large")
		_ = syncResp.Body.Close()
	}
	if syncResp.StatusCode > 400 {
		_ = syncResp.Body.Close()
		return
	}

	// process response body
	var syncRespBodyBytes []byte
	syncRespBodyBytes, err = getResponseBody(syncResp)
	if err != nil {
		return
	}
	err = syncResp.Body.Close()
	if err != nil {
		return
	}
	// get item results from API response
	var bodyContent syncResponse
	bodyContent, err = getBodyContent(syncRespBodyBytes)
	if err != nil {
		return
	}
	// Get new items
	syncToken = stripLineBreak(bodyContent.SyncToken)
	savedItems = bodyContent.SavedItems
	return
}

type encryptedItem struct {
	UUID        string `json:"uuid"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	EncItemKey  string `json:"enc_item_key"`
	Deleted     bool   `json:"deleted"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type decryptedItem struct {
	UUID        string `json:"uuid"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	Deleted     bool   `json:"deleted"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// {
//      "uuid": "3162fe3a-1b5b-4cf5-b88a-afcb9996b23a",
//      "content_type": "Note",
//      "content": {
//        "references": [
//          {
//            "uuid": "901751a0-0b85-4636-93a3-682c4779b634",
//            "content_type": "Tag"
//          }
//        ],
//        "title": "...",
//        "text": "..."
//      },
//      "created_at": "2016-12-16T17:37:50.000Z"
//    },
//
//    {
//      "uuid": "023112fe-9066-481e-8a63-f15f27d3f904",
//      "content_type": "Tag",
//      "content": {
//        "references": [
//          {
//            "uuid": "94cba6b7-6b55-41d6-89a5-e3db8be9fbbf",
//            "content_type": "Note"
//          }
//        ],
//        "title": "essays"
//      },
//      "created_at": "2016-12-16T17:13:20.000Z"
//    }

//func (item encryptedItem) Export() (string, error) {
//	var sb strings.Builder
//	sb.WriteString(fmt.Sprintf("\"uuid\": \"%s\",", item.UUID))
//	sb.WriteString(fmt.Sprintf("\"content_type\": \"%s\",", item.ContentType))
//	content, err := json.Marshal(item.Content)
//	sb.WriteString(fmt.Sprintf("\"content_type\": \"%s\",", json.Marshal(item.Content)))
//	switch item.ContentType {
//	case "Note":
//
//
//	}
//	return input.Content != nil
//}

type UpdateItemRefsInput struct {
	Items []Item // Tags
	ToRef []Item // Items To Reference
}

type UpdateItemRefsOutput struct {
	Items []Item // Tags
}

func UpdateItemRefs(i UpdateItemRefsInput) UpdateItemRefsOutput {
	var updated []Item // updated tags
	for _, item := range i.Items {
		var refs []ItemReference
		for _, tr := range i.ToRef {
			ref := ItemReference{
				UUID:        tr.UUID,
				ContentType: tr.ContentType,
			}
			refs = append(refs, ref)
		}
		item.Content.UpsertReferences(refs)
		updated = append(updated, item)
	}
	return UpdateItemRefsOutput{
		Items: updated,
	}
}
func (input *NoteContent) SetReferences(newRefs []ItemReference) {
	input.ItemReferences = newRefs
}
func (input *TagContent) SetReferences(newRefs []ItemReference) {
	input.ItemReferences = newRefs
}

func (input *TagContent) UpsertReferences(newRefs []ItemReference) {
	for _, newRef := range newRefs {
		var found bool
		for _, existingRef := range input.ItemReferences {
			if existingRef.UUID == newRef.UUID {
				found = true
			}
		}
		if !found {
			input.ItemReferences = append(input.ItemReferences, newRef)
		}
	}
}

func (input *NoteContent) UpsertReferences(newRefs []ItemReference) {
	for _, newRef := range newRefs {
		var found bool
		for _, existingRef := range input.ItemReferences {
			if existingRef.UUID == newRef.UUID {
				found = true
			}
		}
		if !found {
			input.ItemReferences = append(input.ItemReferences, newRef)
		}
	}
}

func (input *GetItemsOutput) DeDupe() {
	input.Items = DeDupeItems(input.Items)
	input.SavedItems = DeDupeItems(input.SavedItems)
	input.Unsaved = DeDupeItems(input.Unsaved)
}

func makeSyncRequest(session Session, reqBody []byte) (response *http.Response, err error) {
	funcName := funcNameOutputStart + "makeSyncRequest" + funcNameOutputEnd

	var request *http.Request
	request, err = http.NewRequest(http.MethodPost, session.Server+syncPath, bytes.NewBuffer(reqBody))
	if err != nil {
		return
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+session.Token)
	response, err = httpClient.Do(request)

	if response.StatusCode >= 400 {
		debug(funcName, fmt.Errorf("sync of %d req bytes failed with: %s", len(reqBody), response.Status))
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		debug(funcName, fmt.Errorf("sync of %d req bytes succeeded with: %s", len(reqBody), response.Status))
	}
	return
}

func getItems(input GetItemsInput) (out GetItemsOutput, err error) {
	funcName := funcNameOutputStart + "getItems" + funcNameOutputEnd
	// determine how many items to retrieve with each call
	var limit int
	switch {
	case input.BatchSize > 0:
		debug(funcName, fmt.Sprintf("input.BatchSize: %d", input.BatchSize))
		// batch size must be lower than or equal to page size
		limit = input.BatchSize
	case input.PageSize > 0:
		debug(funcName, fmt.Sprintf("input.PageSize: %d", input.PageSize))
		limit = input.PageSize
	default:
		debug(funcName, fmt.Sprintf("default - limit: %d", PageSize))
		limit = PageSize
	}
	debug(funcName, fmt.Sprintf("using limit: %d", limit))
	var requestBody []byte
	// generate request body
	switch {
	case input.CursorToken == "":
		debug(funcName, "cursor is empty")
		requestBody = []byte(`{"limit":` + strconv.Itoa(limit) + `}`)
	case input.CursorToken == "null":
		debug(funcName, "\ncursor is null")
		requestBody = []byte(`{"limit":` + strconv.Itoa(limit) +
			`,"items":[],"sync_token":"` + input.SyncToken + `\n","cursor_token":null}`)
	case input.CursorToken != "":
		debug(funcName, fmt.Sprintf("\ncursor is %s", stripLineBreak(input.CursorToken)))
		rawST := input.SyncToken
		input.SyncToken = stripLineBreak(rawST)
		newST := stripLineBreak(input.SyncToken)
		requestBody = []byte(`{"limit":` + strconv.Itoa(limit) +
			`,"items":[],"sync_token":"` + newST + `\n","cursor_token":"` + stripLineBreak(input.CursorToken) + `\n"}`)
	}

	// make the request
	debug(funcName, fmt.Sprintf("making request: %s", stripLineBreak(string(requestBody))))
	syncResp, err := makeSyncRequest(input.Session, requestBody)
	if err != nil {
		return
	}
	// process response body
	var syncRespBodyBytes []byte
	syncRespBodyBytes, err = getResponseBody(syncResp)
	if err != nil {
		return
	}
	if syncResp.StatusCode == 413 {
		fmt.Println("413 err: request entity too large")
		return out, errors.New("413: request entity too large")
	}
	err = syncResp.Body.Close()
	if err != nil {
		return
	}

	// get encrypted items from API response
	var bodyContent syncResponse
	bodyContent, err = getBodyContent(syncRespBodyBytes)
	if err != nil {
		return
	}

	// decrypt retrieved items
	var dItems, dSavedItems, dUnsaved []decryptedItem
	dItems, dSavedItems, dUnsaved, err = decryptItems(bodyContent, input.Session.Mk, input.Session.Ak)
	if err != nil {
		return
	}

	out.SavedItems, err = processDecryptedItems(dSavedItems)
	if err != nil {
		return
	}
	out.Unsaved, err = processDecryptedItems(dUnsaved)
	if err != nil {
		return
	}
	out.Items, err = processDecryptedItems(dItems)
	if err != nil {
		return
	}
	out.SyncToken = bodyContent.SyncToken
	out.Cursor = bodyContent.CursorToken
	if input.BatchSize > 0 {
		return
	}

	if bodyContent.CursorToken != "" && bodyContent.CursorToken != "null" {
		var newOutput GetItemsOutput
		input.SyncToken = out.SyncToken
		input.CursorToken = out.Cursor
		input.PageSize = limit
		newOutput, err = getItems(input)
		out = appendItems(out, newOutput)
	} else {

		return out, err
	}

	return
}

// ItemReference defines a reference from one item to another
type ItemReference struct {
	// unique identifier of the item being referenced
	UUID string `json:"uuid"`
	// type of item being referenced
	ContentType string `json:"content_type"`
}

type OrgStandardNotesSNDetail struct {
	ClientUpdatedAt string `json:"client_updated_at"`
}
type AppDataContent struct {
	OrgStandardNotesSN OrgStandardNotesSNDetail `json:"org.standardnotes.sn"`
}

type NoteContent struct {
	Title          string          `json:"title"`
	Text           string          `json:"text"`
	ItemReferences []ItemReference `json:"references"`
	AppData        AppDataContent  `json:"appData"`
}

func (input *NoteContent) GetUpdateTime() (time.Time, error) {
	if input.AppData.OrgStandardNotesSN.ClientUpdatedAt == "" {
		return time.Time{}, fmt.Errorf("notset")
	}
	return time.Parse(timeLayout, input.AppData.OrgStandardNotesSN.ClientUpdatedAt)
}

func (input *TagContent) GetUpdateTime() (time.Time, error) {
	if input.AppData.OrgStandardNotesSN.ClientUpdatedAt == "" {
		return time.Time{}, fmt.Errorf("notset")
	}
	return time.Parse(timeLayout, input.AppData.OrgStandardNotesSN.ClientUpdatedAt)

}

func (input *NoteContent) SetUpdateTime(uTime time.Time) {
	input.AppData.OrgStandardNotesSN.ClientUpdatedAt = uTime.Format(timeLayout)
}

func (input *TagContent) SetUpdateTime(uTime time.Time) {
	input.AppData.OrgStandardNotesSN.ClientUpdatedAt = uTime.Format(timeLayout)
}

func (input *NoteContent) GetTitle() string {
	return input.Title
}

func (input *NoteContent) SetTitle(title string) {
	input.Title = title
}

func (input *TagContent) SetTitle(title string) {
	input.Title = title
}

func (input *NoteContent) GetText() string {
	return input.Text
}

func (input *NoteContent) SetText(text string) {
	input.Text = text
}

func (input *TagContent) GetText() string {
	// Tags only have titles, so empty string
	return ""
}

func (input *TagContent) SetText(text string) {

}

func (input *TagContent) TextContains(findString string, matchCase bool) bool {
	// Tags only have titles, so always false
	return false
}

func (input *TagContent) GetTitle() string {
	return input.Title
}

func (input *TagContent) References() []ItemReference {
	var output []ItemReference
	return append(output, input.ItemReferences...)
}

func (input *TagContent) GetAppData() AppDataContent {
	return input.AppData
}

func (input *NoteContent) GetAppData() AppDataContent {
	return input.AppData
}

func (input *NoteContent) SetAppData(data AppDataContent) {
	input.AppData = data
}

func (input *TagContent) SetAppData(data AppDataContent) {
	input.AppData = data
}

func (input *NoteContent) References() []ItemReference {
	var output []ItemReference
	return append(output, input.ItemReferences...)
}

type TagContent struct {
	Title          string          `json:"title"`
	ItemReferences []ItemReference `json:"references"`
	AppData        AppDataContent  `json:"appData"`
}

func processDecryptedItems(input []decryptedItem) (output []Item, err error) {
	for i := range input {
		var processedItem Item
		processedItem.ContentType = input[i].ContentType
		if !input[i].Deleted {
			processedItem.Content, err = processContentModel(input[i].ContentType, input[i].Content)
			if err != nil {
				return
			}
		}
		var cAt, uAt time.Time
		cAt, err = time.Parse(timeLayout, input[i].CreatedAt)
		if err != nil {
			return
		}
		processedItem.CreatedAt = cAt.Format(timeLayout)
		uAt, err = time.Parse(timeLayout, input[i].UpdatedAt)
		if err != nil {
			return
		}
		processedItem.UpdatedAt = uAt.Format(timeLayout)
		processedItem.Deleted = input[i].Deleted
		processedItem.UUID = input[i].UUID
		if processedItem.Content != nil {
			if processedItem.Content.GetTitle() != "" {
				processedItem.ContentSize += len(processedItem.Content.GetTitle())
			}
			if processedItem.Content.GetText() != "" {
				processedItem.ContentSize += len(processedItem.Content.GetText())
			}
		}
		output = append(output, processedItem)
	}
	return
}

func appendItems(existing, newItems GetItemsOutput) (output GetItemsOutput) {
	output.Items = append(existing.Items, newItems.Items...)
	output.Unsaved = append(existing.Unsaved, newItems.Unsaved...)
	output.SavedItems = append(existing.SavedItems, newItems.SavedItems...)
	return
}

func processContentModel(contentType, input string) (output ClientStructure, err error) {
	// identify content model
	// try and unmarshall Item
	var itemContent NoteContent
	switch contentType {
	case "Note":
		err = json.Unmarshal([]byte(input), &itemContent)
		return &itemContent, err

	case "Tag":
		var tagContent TagContent
		err = json.Unmarshal([]byte(input), &tagContent)
		return &tagContent, err
	}
	return
}

// DeDupeItems removes any duplicates from a list of items
func DeDupeItems(input []Item) []Item {
	var encountered []string
	var deDuped []Item
	for i := range input {
		if !stringInSlice(input[i].UUID, encountered, true) {
			deDuped = append(deDuped, input[i])
		}
		encountered = append(encountered, input[i].UUID)
	}
	return deDuped
}
