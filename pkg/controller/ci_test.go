package controller

import "testing"

func TestExtractProject(t *testing.T) {
	var git0 = "http://git.com/wocao/go-HyperLPR.git"
	var git1 = "root@git.com:wocao/go-HyperLPR.git"
	g0, err := extractProject(git0)
	if err != nil {
		t.Fatal(err)
	}
	g1, err := extractProject(git1)
	if err != nil {
		t.Fatal(err)
	}
	if g0 != g1 && g0 != "go-HyperLPR" {
		t.Fatal("unexpected error")
	}
}
