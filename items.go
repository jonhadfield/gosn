package gosn

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gopkg.in/matryer/try.v1"
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

// NewSetting returns an Item of type Setting without content
func NewSetting(settingType string) *Item {
	item := newItem()
	item.ContentType = settingType
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
	References() ItemReferences
	// update or insert item references
	UpsertReferences(input ItemReferences)
	// set references
	SetReferences(input ItemReferences)
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
	Items       EncryptedItems `json:"retrieved_items"`
	SavedItems  EncryptedItems `json:"saved_items"`
	Unsaved     EncryptedItems `json:"unsaved"`
	SyncToken   string         `json:"sync_token"`
	CursorToken string         `json:"cursor_token"`
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
	OutType     string
	BatchSize   int // number of items to retrieve
	PageSize    int // override default number of items to request with each sync call
}

// GetItemsOutput defines the output from retrieving items
// It contains slices of items based on their state
// see: https://standardfile.org/ for state details
type GetItemsOutput struct {
	Items      EncryptedItems // items new or modified since last sync
	SavedItems EncryptedItems // dirty items needing resolution
	Unsaved    EncryptedItems // items not saved during sync
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

type EncryptedItems []EncryptedItem

func (ei EncryptedItems) Decrypt(Mk, Ak string) (o DecryptedItems, err error) {
	funcName := funcNameOutputStart + "Decrypt" + funcNameOutputEnd
	debug(funcName, fmt.Errorf("items: %d", len(ei)))

	for _, eItem := range ei {
		var item DecryptedItem
		if eItem.EncItemKey != "" {
			var decryptedEncItemKey string
			decryptedEncItemKey, err = decryptString(eItem.EncItemKey, Mk, Ak, eItem.UUID)
			if err != nil {
				return
			}
			itemEncryptionKey := decryptedEncItemKey[:len(decryptedEncItemKey)/2]
			itemAuthKey := decryptedEncItemKey[len(decryptedEncItemKey)/2:]

			var decryptedContent string
			decryptedContent, err = decryptString(eItem.Content, itemEncryptionKey, itemAuthKey, eItem.UUID)

			if err != nil {
				return
			}
			item.Content = decryptedContent
		}
		item.UUID = eItem.UUID
		item.Deleted = eItem.Deleted
		item.ContentType = eItem.ContentType
		item.UpdatedAt = eItem.UpdatedAt
		item.CreatedAt = eItem.CreatedAt
		o = append(o, item)
	}
	return
}

func (ei EncryptedItems) DecryptAndParse(Mk, Ak string) (o Items, err error) {
	funcName := funcNameOutputStart + "DecryptAndParse" + funcNameOutputEnd
	debug(funcName, fmt.Errorf("items: %d", len(ei)))

	var di DecryptedItems
	di, err = ei.Decrypt(Mk, Ak)
	if err != nil {
		return
	}
	o, err = di.Parse()
	return
}

// GetItems retrieves items from the API using optional filters
func GetItems(input GetItemsInput) (output GetItemsOutput, err error) {
	funcName := funcNameOutputStart + "GetItems" + funcNameOutputEnd

	var sResp syncResponse

	// retry logic is to handle responses that are too large
	// so we can reduce number we retrieve with each sync request
	rErr := try.Do(func(attempt int) (bool, error) {
		var rErr error
		sResp, rErr = getItemsViaAPI(input)
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
	output.Items = sResp.Items
	output.Items.DeDupe()
	output.Unsaved = sResp.Unsaved
	output.Unsaved.DeDupe()
	output.SavedItems = sResp.SavedItems
	output.SavedItems.DeDupe()
	output.Cursor = sResp.CursorToken
	output.SyncToken = sResp.SyncToken
	// strip any duplicates (https://github.com/standardfile/rails-engine/issues/5)
	debug(funcName, fmt.Errorf("sync token: %+v", stripLineBreak(output.SyncToken)))
	return
}

// PutItemsInput defines the input used to put items
type PutItemsInput struct {
	Items     EncryptedItems
	SyncToken string
	Session   Session
}

// PutItemsOutput defines the output from putting items
type PutItemsOutput struct {
	ResponseBody syncResponse
}

func (i *Items) Validate() error {
	var updatedTime time.Time
	var err error
	// TODO finish item validation
	for _, item := range *i {
		// validate content if being added
		if !item.Deleted {
			if stringInSlice(item.ContentType, []string{"Tag", "Note"}, true) {
				updatedTime, err = item.Content.GetUpdateTime()
				switch {
				case item.Content.GetTitle() == "":
					err = fmt.Errorf("failed to create \"%s\" due to missing title: \"%s\"",
						item.ContentType, item.UUID)
				case updatedTime.IsZero():
					err = fmt.Errorf("failed to create \"%s\" due to missing content updated time: \"%s\"",
						item.ContentType, item.Content.GetTitle())
				case item.CreatedAt == "":
					err = fmt.Errorf("failed to create \"%s\" due to missing created at date: \"%s\"",
						item.ContentType, item.Content.GetTitle())
				}
				if err != nil {
					return err
				}
			}
		}
	}
	return err
}

func (i *Items) Encrypt(Mk, Ak string) (e EncryptedItems, err error) {
	e, err = encryptItems(i, Mk, Ak)
	return
}

// PutItems validates and then syncs items via API
func PutItems(i PutItemsInput) (output PutItemsOutput, err error) {
	funcName := funcNameOutputStart + "PutItems" + funcNameOutputEnd
	debug(funcName, fmt.Sprintf("putting %d items", len(i.Items)))

	// for each page size, send to push and get response
	syncToken := stripLineBreak(i.SyncToken)
	var savedItems []EncryptedItem

	// put items in big chunks, default being page size
	for x := 0; x < len(i.Items); x += PageSize {
		var finalChunk bool
		var lastItemInChunkIndex int
		// if current big chunk > num encrypted items then it's the last
		if x+PageSize >= len(i.Items) {
			lastItemInChunkIndex = len(i.Items) - 1
			finalChunk = true
		} else {
			lastItemInChunkIndex = x + PageSize
		}
		debug(funcName, fmt.Sprintf("putting items: %d to %d", x+1, lastItemInChunkIndex+1))

		bigChunkSize := (lastItemInChunkIndex - x) + 1
		fullChunk := i.Items[x : lastItemInChunkIndex+1]
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
				itemsToPut := i.Items[subChunkStart : subChunkEnd+1]
				encItemJSON, _ = json.Marshal(itemsToPut)
				var s []EncryptedItem
				s, syncToken, rErr = putChunk(i.Session, encItemJSON)
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

func putChunk(session Session, encItemJSON []byte) (savedItems []EncryptedItem, syncToken string, err error) {
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

type EncryptedItem struct {
	UUID        string `json:"uuid"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	EncItemKey  string `json:"enc_item_key"`
	Deleted     bool   `json:"deleted"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type DecryptedItem struct {
	UUID        string `json:"uuid"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	Deleted     bool   `json:"deleted"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type DecryptedItems []DecryptedItem

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

//func (item EncryptedItem) Export() (string, error) {
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
	Items Items // Tags
	ToRef Items // Items To Reference
}

type UpdateItemRefsOutput struct {
	Items Items // Tags
}

func UpdateItemRefs(i UpdateItemRefsInput) UpdateItemRefsOutput {
	var updated Items // updated tags
	for _, item := range i.Items {
		var refs ItemReferences
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

func (settingContent *SettingContent) References() ItemReferences {
	return nil
}

func (settingContent *SettingContent) SetReferences(newRefs ItemReferences) {
	return
}

func (settingContent *SettingContent) UpsertReferences(newRefs ItemReferences) {
	return
}

func (settingContent *SettingContent) SetUpdateTime(uTime time.Time) {
	settingContent.AppData.OrgStandardNotesSN.ClientUpdatedAt = uTime.Format(timeLayout)
}

func (settingContent *SettingContent) GetTitle() string {
	return ""
}

func (settingContent *SettingContent) SetTitle(title string) {

}

func (settingContent *SettingContent) GetText() string {
	return ""
}

func (settingContent *SettingContent) SetText(text string) {

}

func (settingContent *SettingContent) GetAppData() AppDataContent {
	return settingContent.AppData
}

func (settingContent *SettingContent) SetAppData(data AppDataContent) {
	settingContent.AppData = data
}

func (settingContent *SettingContent) GetUpdateTime() (time.Time, error) {
	if settingContent.AppData.OrgStandardNotesSN.ClientUpdatedAt == "" {
		return time.Time{}, fmt.Errorf("notset")
	}
	return time.Parse(timeLayout, settingContent.AppData.OrgStandardNotesSN.ClientUpdatedAt)
}

func (noteContent *NoteContent) SetReferences(newRefs ItemReferences) {
	noteContent.ItemReferences = newRefs
}
func (tagContent *TagContent) SetReferences(newRefs ItemReferences) {
	tagContent.ItemReferences = newRefs
}

func (tagContent *TagContent) UpsertReferences(newRefs ItemReferences) {
	for _, newRef := range newRefs {
		var found bool
		for _, existingRef := range tagContent.ItemReferences {
			if existingRef.UUID == newRef.UUID {
				found = true
			}
		}
		if !found {
			tagContent.ItemReferences = append(tagContent.ItemReferences, newRef)
		}
	}
}

func (noteContent *NoteContent) UpsertReferences(newRefs ItemReferences) {
	for _, newRef := range newRefs {
		var found bool
		for _, existingRef := range noteContent.ItemReferences {
			if existingRef.UUID == newRef.UUID {
				found = true
			}
		}
		if !found {
			noteContent.ItemReferences = append(noteContent.ItemReferences, newRef)
		}
	}
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

func getItemsViaAPI(input GetItemsInput) (out syncResponse, err error) {
	funcName := funcNameOutputStart + "getItemsViaAPI" + funcNameOutputEnd
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
	out.Items = bodyContent.Items
	out.SavedItems = bodyContent.SavedItems
	out.Unsaved = bodyContent.Unsaved
	out.SyncToken = bodyContent.SyncToken
	out.CursorToken = bodyContent.CursorToken
	if input.BatchSize > 0 {
		return
	}

	if bodyContent.CursorToken != "" && bodyContent.CursorToken != "null" {
		var newOutput syncResponse
		input.SyncToken = out.SyncToken
		input.CursorToken = out.CursorToken
		input.PageSize = limit
		newOutput, err = getItemsViaAPI(input)
		out.Items = append(out.Items, newOutput.Items...)
		out.SavedItems = append(out.Items, newOutput.SavedItems...)
		out.Unsaved = append(out.Items, newOutput.Unsaved...)
	} else {
		return out, err
	}
	out.CursorToken = ""
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
	Title          string         `json:"title"`
	Text           string         `json:"text"`
	ItemReferences ItemReferences `json:"references"`
	AppData        AppDataContent `json:"appData"`
}

type SettingContent struct {
	Identifier string         `json:"identifier"`
	Name       string         `json:"name"`
	HostedURL  string         `json:"hosted_url"`
	AppData    AppDataContent `json:"appData"`
}

func (noteContent *NoteContent) GetUpdateTime() (time.Time, error) {
	if noteContent.AppData.OrgStandardNotesSN.ClientUpdatedAt == "" {
		return time.Time{}, fmt.Errorf("notset")
	}
	return time.Parse(timeLayout, noteContent.AppData.OrgStandardNotesSN.ClientUpdatedAt)
}

func (tagContent *TagContent) GetUpdateTime() (time.Time, error) {
	if tagContent.AppData.OrgStandardNotesSN.ClientUpdatedAt == "" {
		return time.Time{}, fmt.Errorf("notset")
	}
	return time.Parse(timeLayout, tagContent.AppData.OrgStandardNotesSN.ClientUpdatedAt)

}

func (noteContent *NoteContent) SetUpdateTime(uTime time.Time) {
	noteContent.AppData.OrgStandardNotesSN.ClientUpdatedAt = uTime.Format(timeLayout)
}

func (tagContent *TagContent) SetUpdateTime(uTime time.Time) {
	tagContent.AppData.OrgStandardNotesSN.ClientUpdatedAt = uTime.Format(timeLayout)
}

func (noteContent *NoteContent) GetTitle() string {
	return noteContent.Title
}

func (noteContent *NoteContent) SetTitle(title string) {
	noteContent.Title = title
}

func (tagContent *TagContent) SetTitle(title string) {
	tagContent.Title = title
}

func (noteContent *NoteContent) GetText() string {
	return noteContent.Text
}

func (noteContent *NoteContent) SetText(text string) {
	noteContent.Text = text
}

func (tagContent *TagContent) GetText() string {
	// Tags only have titles, so empty string
	return ""
}

func (tagContent *TagContent) SetText(text string) {

}

func (tagContent *TagContent) TextContains(findString string, matchCase bool) bool {
	// Tags only have titles, so always false
	return false
}

func (tagContent *TagContent) GetTitle() string {
	return tagContent.Title
}

func (tagContent *TagContent) References() ItemReferences {
	var output ItemReferences
	return append(output, tagContent.ItemReferences...)
}

func (tagContent *TagContent) GetAppData() AppDataContent {
	return tagContent.AppData
}

func (noteContent *NoteContent) GetAppData() AppDataContent {
	return noteContent.AppData
}

func (noteContent *NoteContent) SetAppData(data AppDataContent) {
	noteContent.AppData = data
}

func (tagContent *TagContent) SetAppData(data AppDataContent) {
	tagContent.AppData = data
}

func (noteContent *NoteContent) References() ItemReferences {
	var output ItemReferences
	return append(output, noteContent.ItemReferences...)
}

type TagContent struct {
	Title          string         `json:"title"`
	ItemReferences ItemReferences `json:"references"`
	AppData        AppDataContent `json:"appData"`
}

type ItemReferences []ItemReference

type Items []Item

func (di *DecryptedItems) Parse() (p Items, err error) {
	for _, i := range *di {
		var processedItem Item
		processedItem.ContentType = i.ContentType
		processedItem.UUID = i.UUID
		if !i.Deleted {
			processedItem.Content, err = processContentModel(i.ContentType, i.Content)
			if processedItem.Content == nil && len(i.Content) > 0 {
				log.Fatal("processContentModel failed")
			}
			if err != nil {
				return
			}
		}
		var cAt, uAt time.Time
		cAt, err = time.Parse(timeLayout, i.CreatedAt)
		if err != nil {
			return
		}
		processedItem.CreatedAt = cAt.Format(timeLayout)
		uAt, err = time.Parse(timeLayout, i.UpdatedAt)
		if err != nil {
			return
		}
		processedItem.UpdatedAt = uAt.Format(timeLayout)
		processedItem.Deleted = i.Deleted
		processedItem.UUID = i.UUID
		if processedItem.Content != nil {
			if processedItem.Content.GetTitle() != "" {
				processedItem.ContentSize += len(processedItem.Content.GetTitle())
			}
			if processedItem.Content.GetText() != "" {
				processedItem.ContentSize += len(processedItem.Content.GetText())
			}
		}
		p = append(p, processedItem)
	}
	return
}



func processContentModel(contentType, input string) (output ClientStructure, err error) {
	// identify content model
	// try and unmarshall Item
	var itemContent NoteContent
	switch {
	case contentType == "Note":
		err = json.Unmarshal([]byte(input), &itemContent)
		return &itemContent, err
	case contentType == "Tag":
		var tagContent TagContent
		err = json.Unmarshal([]byte(input), &tagContent)
		return &tagContent, err
	case strings.HasPrefix(contentType, "SN|"):
		var settingContent SettingContent
		err = json.Unmarshal([]byte(input), &settingContent)
		return &settingContent, err
	default:
		log.Fatal("unrecognised type:", contentType)
	}
	return
}

func (ei *EncryptedItems) DeDupe() {
	var encountered []string
	var deDuped EncryptedItems
	for _, i := range *ei {
		if !stringInSlice(i.UUID, encountered, true) {
			deDuped = append(deDuped, i)
		}
		encountered = append(encountered, i.UUID)
	}
	*ei = deDuped
}

func (i *Items) DeDupe() {
	var encountered []string
	var deDuped Items
	for _, j := range *i {
		if !stringInSlice(j.UUID, encountered, true) {
			deDuped = append(deDuped, j)
		}
		encountered = append(encountered, j.UUID)
	}
	*i = deDuped
}

func (tagContent TagContent) Equals(e TagContent) bool {
	if tagContent.Title != e.Title {
		return false
	}
	// TODO: compare references
	return true
}

func (item Item) Equals(e Item) bool {
	if item.UUID != e.UUID {
		return false
	}
	if item.ContentType != e.ContentType {
		return false
	}
	if item.Deleted != e.Deleted {
		return false
	}
	if item.Content.GetTitle() != e.Content.GetTitle() {
		return false
	}
	if item.Content.GetText() != e.Content.GetText() {
		return false
	}
	return true
}

func (noteContent NoteContent) Copy() *NoteContent {
	res := new(NoteContent)
	res.Title = noteContent.Title
	res.Text = noteContent.Text
	res.AppData = noteContent.AppData
	res.ItemReferences = noteContent.ItemReferences
	return res
}
func (tagContent TagContent) Copy() *TagContent {
	res := new(TagContent)
	res.Title = tagContent.Title
	res.AppData = tagContent.AppData
	res.ItemReferences = tagContent.ItemReferences
	return res
}

func (item Item) Copy() *Item {
	res := new(Item)
	switch item.Content.(type) {
	case *NoteContent:
		tContent := item.Content.(*NoteContent)
		res.Content = tContent.Copy()

	case *TagContent:
		tContent := item.Content.(*TagContent)
		res.Content = tContent.Copy()
	default:
		fmt.Printf("unable to copy items with content of type: %s", reflect.TypeOf(item.Content))
	}
	res.UpdatedAt = item.UpdatedAt
	res.CreatedAt = item.CreatedAt
	res.ContentSize = item.ContentSize
	res.ContentType = item.ContentType
	res.UUID = item.UUID
	return res
}
