package file

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestConvert(t *testing.T) {
	content := []byte("temporary file's content")
	dir, err := ioutil.TempDir("", "example")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(dir) // clean up

	tmpfn := filepath.Join(dir, "tmpfile")
	if err := ioutil.WriteFile(tmpfn, content, 0666); err != nil {
		t.Fatal(err)
	}

	if bs, err := NewIConvert(tmpfn).Convert(); err != nil || len(bs) < 1 || !bytes.Equal(bs, content) {
		t.Fatal("non expect error")
	}
}
