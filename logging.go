package gosn

type Logger func(...interface{})

var debugLog Logger
