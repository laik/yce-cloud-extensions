package controller

import (
	"strings"
	"testing"
)

func TestReCheckName(t *testing.T) {
	var testStr = "aaaaaa-app-vendor-yamei-apollo-plugin-configs-apollo-env-properties"
	testStr = reCheckName(testStr)
	if len(testStr) > 62 {
		t.Fatal("length too long")
	}
	if strings.HasPrefix(testStr, "-"){
		t.Fatal("strings start with -")
	}

}
