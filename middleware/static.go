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

	fileServer http.Handler
}

type FS struct{ http.FileSystem }
type File struct{ http.File }

func (s *Static) Wrap(http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.fileServer.ServeHTTP(w, r)
	})
}

func (s *Static) Start(ctx context.Context) error {
	if f, err := os.Stat(s.Root); err != nil || !f.IsDir() {
		return fmt.Errorf("root must be a directory: %s", err)
	}
	fs := http.FileSystem(http.Dir(s.Root))
	if !s.ListDirectories {
		fs = FS{fs}
	}
	s.fileServer = http.FileServer(fs)
	<-ctx.Done()
	return nil
}

func (fs FS) Open(name string) (http.File, error) {
	f, err := fs.FileSystem.Open(name)
	return File{f}, err
}

func (f File) Readdir(int) ([]os.FileInfo, error) { return nil, nil }
