package serve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

type WaitGroup struct {
	count  int
	ctx    context.Context
	cancel func()
	errors chan error
}

func sortMiddlewares(in []Middleware) ([]Middleware, error) {
	tmp, out := map[string]Middleware{}, []Middleware{}
	for _, m := range in {
		name := strings.Split(fmt.Sprintf("%T", m.Handler), ".")[1]
		tmp[name] = m
	}
	for _, h := range sortedHandlers {
		name := strings.Split(fmt.Sprintf("%T", h), ".")[1]
		if m, ok := tmp[name]; ok {
			out = append(out, m)
		}
	}
	if len(tmp) != len(out) {
		return nil, errors.New("duplicate middlewares are not allowed")
	}
	return out, nil
}

func readJSON(path string, v interface{}) error {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, v)
}

func NewWaitGroup(ctx context.Context) *WaitGroup {
	ctx, cancel := context.WithCancel(ctx)
	return &WaitGroup{ctx: ctx, cancel: cancel, errors: make(chan error)}
}

func (wg *WaitGroup) Start(fn func(context.Context) error) {
	wg.count++
	go func() { wg.errors <- fn(wg.ctx) }()
}

func (wg *WaitGroup) Cancel() { wg.cancel() }

func (wg *WaitGroup) Wait() error {
	err := <-wg.errors
	wg.cancel()
	for i := 1; i < wg.count; i++ {
		<-wg.errors
	}
	return err
}

func ReloadSignalContext(signals ...os.Signal) context.Context {
	c := make(chan os.Signal)
	signal.Notify(c, signals...)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-c
		cancel()
	}()
	return ctx
}

// https://www.freedesktop.org/software/systemd/man/systemd-socket-activate.html
// https://www.freedesktop.org/software/systemd/man/sd_listen_fds.html
func systemdSocket() (net.Listener, error) {
	pid, err := strconv.Atoi(os.Getenv("LISTEN_PID"))
	if err != nil || pid != os.Getpid() {
		return nil, errors.New("not called from systemd")
	}
	n, err := strconv.Atoi(os.Getenv("LISTEN_FDS"))
	if err != nil || n != 1 {
		return nil, fmt.Errorf("wrong number of file descriptors passed: %d", n)
	}
	f := os.NewFile(3, "systemd socket") // 1 stdout, 2 stderr
	if f == nil {
		return nil, errors.New("file descriptor 3 does not exist")
	}
	return net.FileListener(f)
}
