package gosn

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	uuid "github.com/satori/go.uuid"
)

func stripLineBreak(input string) string {
	if strings.HasSuffix(input, "\n") {
		return input[:len(input)-1]
	}

	return input
}

// GenUUID generates a unique identifier required when creating a new item
func GenUUID() string {
	newUUID := uuid.NewV4()
	return newUUID.String()
}

func stringInSlice(inStr string, inSlice []string, matchCase bool) bool {
	for i := range inSlice {
		if matchCase && inStr == inSlice[i] {
			return true
		} else if strings.EqualFold(inStr, inSlice[i]) {
			return true
		}
	}

	return false
}

func getResponseBody(resp *http.Response) (body []byte, err error) {
	var output io.ReadCloser

	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		output, err = gzip.NewReader(resp.Body)
		if err != nil {
			return
		}
	default:
		output = resp.Body
	}

	buf := new(bytes.Buffer)

	_, err = buf.ReadFrom(output)
	if err != nil {
		return
	}

	body = buf.Bytes()

	return
}
