package gosn

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

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

func getResponseBody(resp *http.Response, debug bool) (body []byte, err error) {
	start := time.Now()
	defer func() {
		debugPrint(debug, fmt.Sprintf("getResponseBody | duration: %+v", time.Since(start)))
	}()

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

	readTimeStart := time.Now()
	body, err = ioutil.ReadAll(output)
	debugPrint(debug, fmt.Sprintf("getResponseBody | read time: %+v", time.Since(readTimeStart)))

	return
}
