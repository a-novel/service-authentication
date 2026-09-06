package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const serveTestTimeout = 5 * time.Second

type blockingWaiter struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func newBlockingWaiter() *blockingWaiter {
	return &blockingWaiter{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (w *blockingWaiter) Wait() {
	close(w.started)
	<-w.release
}

func (w *blockingWaiter) finish() {
	w.once.Do(func() { close(w.release) })
}

func TestServe(t *testing.T) {
	t.Parallel()

	t.Run("DrainsActiveRequestThenPendingMail", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		t.Cleanup(cancel)

		requestStarted := make(chan struct{})
		releaseRequest := make(chan struct{})
		waiter := newBlockingWaiter()
		t.Cleanup(waiter.finish)

		server := &http.Server{
			Addr:              freeAddress(t),
			ReadHeaderTimeout: time.Second,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				close(requestStarted)
				<-releaseRequest
				w.WriteHeader(http.StatusNoContent)
			}),
		}

		serveErrors := make(chan error, 1)
		go func() { serveErrors <- serve(ctx, cancel, server, time.Second, waiter) }()

		waitForServer(t, server.Addr)
		requestErrors := request(t, server.Addr)

		select {
		case <-requestStarted:
		case <-time.After(serveTestTimeout):
			require.FailNow(t, "HTTP request did not start")
		}

		cancel()
		close(releaseRequest)

		select {
		case <-waiter.started:
		case <-time.After(serveTestTimeout):
			require.FailNow(t, "mail drain did not start after HTTP shutdown")
		}

		select {
		case <-serveErrors:
			require.FailNow(t, "server returned while accepted mail was pending")
		case <-time.After(50 * time.Millisecond):
		}

		waiter.finish()

		select {
		case err := <-serveErrors:
			require.NoError(t, err)
		case <-time.After(serveTestTimeout):
			require.FailNow(t, "server did not finish after request and mail drained")
		}

		require.NoError(t, <-requestErrors)
	})

	t.Run("BoundsForcedShutdown", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		t.Cleanup(cancel)

		requestStarted := make(chan struct{})
		releaseRequest := make(chan struct{})
		waiter := newBlockingWaiter()

		t.Cleanup(func() {
			close(releaseRequest)
			waiter.finish()
		})

		server := &http.Server{
			Addr:              freeAddress(t),
			ReadHeaderTimeout: time.Second,
			Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				close(requestStarted)
				<-releaseRequest
			}),
		}

		serveErrors := make(chan error, 1)
		go func() { serveErrors <- serve(ctx, cancel, server, 50*time.Millisecond, waiter) }()

		waitForServer(t, server.Addr)
		requestErrors := request(t, server.Addr)

		select {
		case <-requestStarted:
		case <-time.After(serveTestTimeout):
			require.FailNow(t, "HTTP request did not start")
		}

		started := time.Now()

		cancel()

		select {
		case err := <-serveErrors:
			require.ErrorIs(t, err, context.DeadlineExceeded)
			require.Less(t, time.Since(started), time.Second)
		case <-time.After(serveTestTimeout):
			require.FailNow(t, "forced shutdown exceeded its budget")
		}

		select {
		case <-requestErrors:
		case <-time.After(serveTestTimeout):
			require.FailNow(t, "forced shutdown did not close the active request")
		}
	})

	t.Run("BoundsServingFailure", func(t *testing.T) {
		t.Parallel()

		listenerConfig := &net.ListenConfig{}
		listener, err := listenerConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, listener.Close()) })

		ctx, cancel := context.WithCancel(t.Context())
		t.Cleanup(cancel)

		server := &http.Server{
			Addr:              listener.Addr().String(),
			Handler:           http.NotFoundHandler(),
			ReadHeaderTimeout: time.Second,
		}

		started := time.Now()
		err = serve(ctx, cancel, server, 50*time.Millisecond)

		require.Error(t, err)
		require.Less(t, time.Since(started), time.Second)
	})
}

func freeAddress(t *testing.T) string {
	t.Helper()

	listenerConfig := &net.ListenConfig{}
	listener, err := listenerConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	return address
}

func waitForServer(t *testing.T, address string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), serveTestTimeout)
	defer cancel()

	dialer := &net.Dialer{}

	for {
		connection, err := dialer.DialContext(ctx, "tcp", address)
		if err == nil {
			require.NoError(t, connection.Close())

			return
		}

		select {
		case <-ctx.Done():
			require.NoError(t, ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func request(t *testing.T, address string) <-chan error {
	t.Helper()

	errs := make(chan error, 1)

	go func() {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+address, nil)
		if err != nil {
			errs <- err

			return
		}

		response, err := http.DefaultClient.Do(request)
		if response != nil {
			err = errors.Join(err, response.Body.Close())
		}

		errs <- err
	}()

	return errs
}
