package gosn

import (
	"testing"
)

func TestStringInSliceCaseSensitive(t *testing.T) {
	if !stringInSlice("Marmite", []string{"Cheese", "Marmite", "Toast"}, true) {
		t.Error("failed.")
	}
}

func TestStringInSliceCaseInsensitive(t *testing.T) {
	if !stringInSlice("marmite", []string{"Cheese", "Marmite", "Toast"}, false) {
		t.Error("failed.")
	}
}
