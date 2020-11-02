package file

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

// IConvert exports the interface as the named
type IConvert interface {
	Convert() ([]byte, error)
}

type fileImpl struct{ path string }

// NewIConvert new the path converted to bytes slice
func NewIConvert(path string) IConvert {
	return &fileImpl{path: path}
}

func (f *fileImpl) Convert() ([]byte, error) {
	r, err := openFile(f.path)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}

func openFile(path string) (io.Reader, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("path (%s) does not exist", path)
	}
	r, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return r, nil
}
