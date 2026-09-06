package translate

import (
	"context"
	"errors"
	"testing"
)

type fake struct {
	out   string
	err   error
	calls int
}

func (f *fake) Translate(_ context.Context, _, _, _ string) (string, error) {
	f.calls++
	return f.out, f.err
}

func TestChainFirstSucceeds(t *testing.T) {
	primary := &fake{out: "ok"}
	backup := &fake{out: "unused"}
	c := NewChain(nil, primary, backup)

	got, err := c.Translate(context.Background(), "x", "en", "be")
	if err != nil || got != "ok" {
		t.Fatalf("got (%q, %v), want (%q, nil)", got, err, "ok")
	}
	if backup.calls != 0 {
		t.Errorf("backup should not have been called")
	}
}

func TestChainFallsBack(t *testing.T) {
	primary := &fake{err: errors.New("down")}
	backup := &fake{out: "from-backup"}
	c := NewChain(nil, primary, backup)

	got, err := c.Translate(context.Background(), "x", "en", "be")
	if err != nil || got != "from-backup" {
		t.Fatalf("got (%q, %v), want (%q, nil)", got, err, "from-backup")
	}
	if primary.calls != 1 || backup.calls != 1 {
		t.Errorf("calls: primary=%d backup=%d, want 1/1", primary.calls, backup.calls)
	}
}

func TestChainAllFail(t *testing.T) {
	last := errors.New("also down")
	c := NewChain(nil, &fake{err: errors.New("down")}, &fake{err: last})

	_, err := c.Translate(context.Background(), "x", "en", "be")
	if !errors.Is(err, last) {
		t.Fatalf("want wrapped last error, got %v", err)
	}
}
