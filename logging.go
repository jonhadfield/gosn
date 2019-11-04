package gosn

type Logger func(...interface{})

var debugLog Logger

func SetDebugLogger(logger Logger) {
	debugLog = logger
}

var errorLog Logger

func SetErrorLogger(logger Logger) {
	errorLog = logger
}
