package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestServeDrainsActiveRequest(t *testing.T) {
	listener := testListener(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	finishRequest := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(finishRequest)
	served := make(chan error, 1)
	go func() {
		served <- serve(ctx, listener, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			close(entered)
			<-release
			_, _ = io.WriteString(w, "finished")
		}), time.Second)
	}()
	requested := requestResult(listener.Addr().String())
	waitSignal(t, entered)
	cancel()
	deadline := time.Now().Add(time.Second)
	for {
		conn, err := net.DialTimeout("tcp", listener.Addr().String(), 50*time.Millisecond)
		if err != nil {
			break
		}
		_ = conn.Close()
		if time.Now().After(deadline) {
			t.Fatal("shutdown did not close the listening socket")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case err := <-served:
		t.Fatalf("server returned before active request finished: %v", err)
	default:
	}
	finishRequest()
	select {
	case result := <-requested:
		if result.err != nil || result.status != 200 || result.body != "finished" {
			t.Fatalf("active request = %#v", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("request did not finish")
	}
	if err := waitError(t, served); err != nil {
		t.Fatalf("shutdown = %v", err)
	}
}

func TestServeBoundsStuckRequest(t *testing.T) {
	listener := testListener(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered := make(chan struct{})
	exited := make(chan struct{})
	served := make(chan error, 1)
	go func() {
		served <- serve(ctx, listener, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			close(entered)
			<-r.Context().Done()
			close(exited)
		}), 50*time.Millisecond)
	}()
	requested := requestResult(listener.Addr().String())
	waitSignal(t, entered)
	cancel()
	if err := waitError(t, served); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown = %v, want deadline exceeded", err)
	}
	waitSignal(t, exited)
	select {
	case result := <-requested:
		if result.err == nil {
			t.Fatal("stuck request's connection stayed open")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stuck request's connection did not close")
	}
}

func TestServeReturnsListenerError(t *testing.T) {
	listener := testListener(t)
	_ = listener.Close()
	result := make(chan error, 1)
	go func() { result <- serve(context.Background(), listener, http.NotFoundHandler(), time.Second) }()
	if err := waitError(t, result); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("serve = %v, want closed listener error", err)
	}
}

func testListener(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	return listener
}

type httpResult struct {
	status int
	body   string
	err    error
}

func requestResult(addr string) <-chan httpResult {
	result := make(chan httpResult, 1)
	go func() {
		client := &http.Client{Timeout: 2 * time.Second}
		defer client.CloseIdleConnections()
		response, err := client.Get("http://" + addr + "/held")
		if err != nil {
			result <- httpResult{err: err}
			return
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		result <- httpResult{status: response.StatusCode, body: string(body), err: err}
	}()
	return result
}

func waitSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for handler")
	}
}

func waitError(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("server did not return")
		return nil
	}
}
