package runner

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	called := false
	runner := New(func(ctx context.Context) error {
		called = true
		return nil
	})

	err := runner.Run(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	if !called {
		t.Error("function wasn't called")
	}
}

func TestErr(t *testing.T) {
	expectedErr := errors.New("test error")
	runner := Err(expectedErr)

	err := runner.Run(context.Background())
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestOK(t *testing.T) {
	runner := OK()

	err := runner.Run(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestJoin_Empty(t *testing.T) {
	runner := Join()

	err := runner.Run(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestJoin_Success(t *testing.T) {
	count := &atomic.Int32{}
	r1 := New(func(ctx context.Context) error {
		count.Add(1)
		return nil
	})
	r2 := New(func(ctx context.Context) error {
		time.Sleep(time.Millisecond * 100)
		count.Add(1)
		return nil
	})

	runner := Join(r1, r2)
	err := runner.Run(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	res := count.Load()
	if res != 2 {
		t.Errorf("expected count to be 2, got %d", res)
	}
}

func TestJoin_Error(t *testing.T) {
	expectedErr := errors.New("test error")

	runner := Join(Err(expectedErr), OK())
	err := runner.Run(context.Background())
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestJoin_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	executed := false
	r := New(func(ctx context.Context) error {
		// We'll wait for either timeout or cancellation
		select {
		case <-time.After(500 * time.Millisecond):
			executed = true
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	// Cancel the context shortly after starting
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := r.Run(ctx)
	if err == nil {
		t.Error("expected an error, got nil")
	}

	if executed {
		t.Error("function shouldn't have finished execution")
	}
}
