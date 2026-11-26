package server

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func awaitRun(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(15 * time.Second):
		t.Fatal("server did not stop")
		return nil
	}
}

func TestServerConfig(t *testing.T) {
	cfg := testConfig("*")
	cfg.Server.Port = "3001"
	srv := New(cfg, pingerFunc(func(context.Context) error { return nil }), "test")
	if srv.Addr() != ":3001" {
		t.Errorf("Addr = %s", srv.Addr())
	}
	c := srv.app.Config()
	if c.ReadTimeout != 15*time.Second || c.WriteTimeout != 15*time.Second || c.IdleTimeout != 60*time.Second {
		t.Errorf("timeouts = %v %v %v", c.ReadTimeout, c.WriteTimeout, c.IdleTimeout)
	}
}

func TestServerAlreadyCanceled(t *testing.T) {
	srv := New(testConfig("*"), pingerFunc(func(context.Context) error { return nil }), "test")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx) }()
	if err := awaitRun(t, done); err != nil {
		t.Fatal(err)
	}
}

func TestServerBindFailure(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	srv := New(testConfig("*"), pingerFunc(func(context.Context) error { return nil }), "test")
	srv.addr = listener.Addr().String()
	if err := srv.Run(context.Background()); err == nil {
		t.Fatal("expected bind failure")
	}
}

func TestServerRejectsMalformedRequest(t *testing.T) {
	srv := New(testConfig("*"), pingerFunc(func(context.Context) error { return nil }), "test")
	srv.addr = "127.0.0.1:0"
	addr := make(chan string, 1)
	srv.app.Hooks().OnListen(func(data fiber.ListenData) error {
		addr <- net.JoinHostPort(data.Host, data.Port)
		return nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx) }()
	t.Cleanup(func() { cancel(); awaitRun(t, done) })
	var listenAddr string
	select {
	case listenAddr = <-addr:
	case err := <-done:
		// La limpieza ya no debe esperar un segundo resultado.
		done <- err
		t.Fatalf("server failed to start: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not start")
	}
	conn, err := net.DialTimeout("tcp", listenAddr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(conn, "GET /health HTTP/1.1\r\nHost: localhost\r\nContent-Length: invalid\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("malformed request: got %d, want 400", resp.StatusCode)
	}
}

func TestServerDrainsInFlightRequest(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	srv := New(testConfig("*"), pingerFunc(func(context.Context) error { close(entered); <-release; return nil }), "test")
	srv.addr = "127.0.0.1:0"
	addr := make(chan string, 1)
	srv.app.Hooks().OnListen(func(data fiber.ListenData) error { addr <- net.JoinHostPort(data.Host, data.Port); return nil })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx) }()
	var listenAddr string
	select {
	case listenAddr = <-addr:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not start")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	requestDone := make(chan error, 1)
	go func() {
		resp, err := client.Get("http://" + listenAddr + "/api/v1/status")
		if err == nil {
			_, err = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode != 200 {
				err = errors.New("in-flight request did not finish successfully")
			}
		}
		requestDone <- err
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("request did not reach handler")
	}
	cancel()
	select {
	case err := <-done:
		close(release)
		t.Fatalf("server stopped before request completed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	if err := <-requestDone; err != nil {
		t.Fatal(err)
	}
	if err := awaitRun(t, done); err != nil {
		t.Fatal(err)
	}
}

func TestServerShutdownDeadline(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	srv := New(testConfig("*"), pingerFunc(func(context.Context) error { close(entered); <-release; return nil }), "test")
	srv.addr = "127.0.0.1:0"
	addr := make(chan string, 1)
	srv.app.Hooks().OnListen(func(data fiber.ListenData) error { addr <- net.JoinHostPort(data.Host, data.Port); return nil })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx) }()
	var listenAddr string
	select {
	case listenAddr = <-addr:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not start")
	}
	client := &http.Client{Timeout: 14 * time.Second}
	defer client.CloseIdleConnections()
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		resp, err := client.Get("http://" + listenAddr + "/api/v1/status")
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("request did not reach handler")
	}
	start := time.Now()
	cancel()
	err := awaitRun(t, done)
	close(release)
	<-requestDone
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("shutdown error = %v", err)
	}
	if elapsed := time.Since(start); elapsed < 9*time.Second || elapsed > 13*time.Second {
		t.Errorf("shutdown duration = %v", elapsed)
	}
}
