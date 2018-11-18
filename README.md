# gosn
[![Build Status](https://www.travis-ci.org/jonhadfield/gosn.svg?branch=master)](https://www.travis-ci.org/jonhadfield/gosn) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/jonhadfield/gosn/) [![Go Report Card](https://goreportcard.com/badge/github.com/jonhadfield/gosn)](https://goreportcard.com/report/github.com/jonhadfield/gosn) [![Coverage Status](https://coveralls.io/repos/github/jonhadfield/gosn/badge.svg?branch=master)](https://coveralls.io/github/jonhadfield/gosn?branch=master) 


# about
<a href="https://standardnotes.org/" target="_blank">Standard Notes</a> is a service and application for the secure management and storage of notes.
gosn is a library to help develop your own application to manage notes on the official, or your self-hosted, Standard Notes server.

# status

A work in progress. Please create backup before using this to manage notes.

# installation

Using go get: ``` go get github.com/jonhadfield/gosn```

# basic usage
## authenticating

To interact with Standard Notes you first need to sign in:

```golang
    sIn := gosn.SignInInput{
        Email:     "someone@example.com",
        Password:  "mysecret,
    }
    sOut, _ := gosn.SignIn(sIn)
```

This will return a session containing the necessary secrets and information to make requests to get or put data.

## getting items

```golang
    input := GetItemsInput{
        Session: sOut.Session,
    }
    
    output, _ = GetItems(input)
```

## creating a note

```golang
    # define note content
    content := NoteContent{
        Title:          "Note Title",
        Text:           "Note Text",
    }
    # define note
    note := NewNote()
    note.Content = content
    
    # create note
    pii := PutItemsInput{
    		Session: sOut.Session,
    		Items:   []gosn.Notes{note},
    }
    _, _ = PutItems(pii)
```

