package serve

import (
	"testing"
)

func TestStartStop(t *testing.T) {
}

/*
things i want to test

- ensure reload kills all goroutines - https://golang.org/pkg/runtime/#NumGoroutine
  restart server a few times and check #routines doesn't increase
- log rotates files as expected by time and is able to write while rotating
  advance mocked time (midnight)
  actually, extract awaitRotateCondition and turn it into a func() chan bool
  -> can be time based or size based or whatever
  -> and in tests it can be controlled by me
- proxy proxies correctly
  spin up another server at some port and let it do static and then proxy that
  run the static tests against that?
- static serves static correctly (and not files)
- auth protects paths correctly

- systemd integration stuff
- how to replace
*/
