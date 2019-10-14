package conf

import (
	"io"
	"os"
)

type FileSystem interface {
	Open(name string) (ReadSeekCloser, error)
	Stat(path string) (os.FileInfo, error)
}

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type OsFS struct{}

func (OsFS) Open(name string) (ReadSeekCloser, error) { return os.Open(name) }
func (OsFS) Stat(name string) (os.FileInfo, error)    { return os.Stat(name) }
