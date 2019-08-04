package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

type Static struct {
	Root            string
	ListDirectories bool
}

type FS struct{ http.FileSystem }
type File struct{ http.File }

func (s *Static) Wrap(http.Handler) (http.Handler, func(context.Context) error, error) {
	if f, err := os.Stat(s.Root); err != nil || !f.IsDir() {
		return nil, nil, fmt.Errorf("root must be a directory: %s", err)
	}
	fs := http.FileSystem(http.Dir(s.Root))
	if !s.ListDirectories {
		fs = FS{fs}
	}
	fileServer := http.FileServer(fs)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fileServer.ServeHTTP(w, r)
	}), nil, nil
}

func (fs FS) Open(name string) (http.File, error) {
	f, err := fs.FileSystem.Open(name)
	return File{f}, err
}

func (f File) Readdir(int) ([]os.FileInfo, error) { return nil, nil }
