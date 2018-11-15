package gosn

import (
	"net/http"
	"time"
)

const (
	// API
	apiServer        = "https://sync.standardnotes.org"
	authParamsPath   = "/auth/params"  // remote path for getting auth parameters
	authRegisterPath = "/auth"         // remote path for registering user
	signInPath       = "/auth/sign_in" // remote path for authenticating
	syncPath         = "/items/sync"   // remote path for making sync calls
	// PageSize is the maximum number of items to return with each call
	PageSize            = 200
	timeLayout          = "2006-01-02T15:04:05.000Z"
	defaultSNVersion    = "003"
	defaultPasswordCost = 110000

	// LOGGING
	libName             = "gosn" // name of library used in logging
	maxDebugChars       = 120    // number of characters to display when logging API response body
	funcNameOutputStart = "["    // prefix for outputting function name in log messages
	funcNameOutputEnd   = "]"    // suffix for outputting function name in log messages

	// HTTP
	MaxIdleConnections = 5  // HTTP transport limit
	MaxConnsPerHost    = 5  // HTTP transport limit
	RequestTimeout     = 60 // HTTP transport limit
)

var (
	httpClient *http.Client
)

func init() {
	httpClient = createHTTPClient()
}

// createHTTPClient for connection re-use
func createHTTPClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: MaxIdleConnections,
			MaxConnsPerHost:     MaxConnsPerHost,
			DisableKeepAlives:   false,
		},
		Timeout: time.Duration(RequestTimeout) * time.Second,
	}

	return client
}

func debug(funcName string, msg interface{}) {
	if debugLog != nil {
		debugLog(libName, funcName, msg)
	}
}
