package gosn

import (
	"regexp"
	"strconv"
	"strings"
)

type ItemFilters struct {
	MatchAny bool
	Filters  []Filter
}

type Filter struct {
	Type       string
	Key        string
	Comparison string
	Value      string
}

func filterItems(items []Item, itemFilters ItemFilters) []Item {
	var filtered []Item
	var tags []Item
	for _, i := range items {
		if i.ContentType == "Tag" {
			tags = append(tags, i)
		}
	}
	for _, item := range items {
		switch item.ContentType {
		case "Note":
			if found := applyNoteFilters(item, itemFilters, tags); found {
				filtered = append(filtered, item)
			}
		case "Tag":
			if found := applyTagFilters(item, itemFilters); found {
				filtered = append(filtered, item)
			}
		}
	}
	return filtered
}

func applyNoteFilters(item Item, itemFilters ItemFilters, tags []Item) bool {
	var matchedAll bool
	for i, filter := range itemFilters.Filters {
		if filter.Type != "Note" {
			continue
		}
		switch strings.ToLower(filter.Key) {
		case "title": // GetTitle
			if item.Content == nil {
				matchedAll = false
			} else {
				switch filter.Comparison {
				case "~":
					r := regexp.MustCompile(filter.Value)
					if r.MatchString(item.Content.GetTitle()) {
						if itemFilters.MatchAny {
							return true
						}
						matchedAll = true
					} else {
						matchedAll = false
					}
				case "==":
					if item.Content.GetTitle() == filter.Value {
						if itemFilters.MatchAny {
							return true
						}
						matchedAll = true
					} else {
						matchedAll = false

					}
				case "!=":
					if item.Content.GetTitle() != filter.Value {
						if itemFilters.MatchAny {
							return true
						}
						matchedAll = true
					} else {
						matchedAll = false
					}
				case "contains":
					if item.Content != nil && strings.Contains(item.Content.GetTitle(), filter.Value) {
						if itemFilters.MatchAny {
							return true
						}
						matchedAll = true

					} else {
						matchedAll = false
					}
				}
			}

		case "text": // Text
			if item.Content == nil {
				matchedAll = false
			} else {
				switch filter.Comparison {
				case "~":
					// TODO: Don't compile every time
					r := regexp.MustCompile(filter.Value)
					text := item.Content.GetText()
					if r.MatchString(text) {
						if itemFilters.MatchAny {
							return true
						}
						matchedAll = true
					} else {
						matchedAll = false
					}
				case "==":
					if item.Content.GetText() == filter.Value {
						if itemFilters.MatchAny {
							return true
						}
						matchedAll = true
					} else {
						matchedAll = false
					}
				case "!=":
					if item.Content.GetText() != filter.Value {
						if itemFilters.MatchAny {
							return true
						}
						matchedAll = true
					} else {
						matchedAll = false
					}
				case "contains":
					if strings.Contains(item.Content.GetText(), filter.Value) {
						if itemFilters.MatchAny {
							return true
						}
						matchedAll = true
					} else {
						matchedAll = false
					}
				}
			}
		case "tagtitle": // Tag Title
			var matchesTag bool
			for _, tag := range tags {
				if tag.Content != nil && tag.Content.GetTitle() == filter.Value {
					for _, ref := range tag.Content.References() {
						if item.UUID == ref.UUID {
							matchesTag = true
						}
					}
				}

			}
			if matchesTag {
				if itemFilters.MatchAny {
					return true
				}
				matchedAll = true
			} else {
				matchedAll = false
			}
		case "taguuid": // Tag UUID
			var matchesTag bool

			for _, tag := range tags {
				for _, ref := range tag.Content.References() {
					if item.UUID == ref.UUID {
						matchesTag = true
					}
				}
			}
			if matchesTag {
				if itemFilters.MatchAny {
					return true
				}
				matchedAll = true
			} else {
				matchedAll = false
			}
		case "uuid": // UUID
			if item.UUID == filter.Value {
				if itemFilters.MatchAny {
					return true
				}
				matchedAll = true
			} else {
				matchedAll = false
			}
		case "deleted":
			isDel, _ := strconv.ParseBool(filter.Value)
			if item.Deleted == isDel {
				if itemFilters.MatchAny {
					return true
				}
				matchedAll = true
			} else {
				matchedAll = false
			}
		default:
			matchedAll = true // if no criteria specified then filter applies to type only
		}
		// if last filter and matchedAll is true, then return true
		if i == len(itemFilters.Filters)-1 {
			if matchedAll {
				return true
			}
		}
	}
	return matchedAll
}

func applyTagFilters(item Item, itemFilters ItemFilters) bool {
	var matchedAll bool
	for i, filter := range itemFilters.Filters {
		if filter.Type != "Tag" {
			continue
		}
		switch strings.ToLower(filter.Key) {
		case "title":
			if item.Content == nil {
				matchedAll = false
			} else {
				switch filter.Comparison {
				case "~":
					r := regexp.MustCompile(filter.Value)
					if r.MatchString(item.Content.GetTitle()) {
						if itemFilters.MatchAny {
							return true
						}
						matchedAll = true
					} else {
						matchedAll = false
					}
				case "==":
					if item.Content.GetTitle() == filter.Value {
						if itemFilters.MatchAny {
							return true
						}
						matchedAll = true
					} else {
						matchedAll = false
					}
				case "!=":
					if item.Content.GetTitle() != filter.Value {
						if itemFilters.MatchAny {
							return true
						}
						matchedAll = true

					} else {
						matchedAll = false
					}
				case "contains":
					if strings.Contains(item.Content.GetTitle(), filter.Value) {
						if itemFilters.MatchAny {
							return true
						}
						matchedAll = true

					} else {
						matchedAll = false
					}
				}
			}
		case "uuid":
			if item.UUID == filter.Value {
				if itemFilters.MatchAny {
					return true
				}
				matchedAll = true

			} else {
				matchedAll = false
			}
		default:
			matchedAll = true // if no criteria specified then filter applies to type only, so true
		}
		// if last filter and matchedAll is true, then return true
		if i == len(itemFilters.Filters)-1 {
			if matchedAll {
				return true
			}
		}
	}

	return false
}
