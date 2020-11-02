package file

import (
	"fmt"
	"io"
	"os"
)

type IConvert interface {
	Convert(io.Reader) ([]byte, error)
}

type fileImpl struct{ path string }

func NewIConvert(path string) IConvert {
	return &fileImpl{path: path}
}

func (f *fileImpl) Convert(r io.Reader) ([]byte, error) {

	return nil, nil
}

func OpenFile(path string) (io.Reader, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("path (%s) does not exist", path)
	}

	return nil, nil
}

func ReadFile(c IConvert, r io.Reader) ([]byte, error) {
	return c.Convert(r)
}
