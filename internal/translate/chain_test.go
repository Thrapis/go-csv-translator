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

type batchFake struct {
	calls  int
	prefix string
}

func (b *batchFake) Translate(_ context.Context, text, _, _ string) (string, error) {
	b.calls++
	return b.prefix + text, nil
}
func (b *batchFake) TranslateBatch(_ context.Context, texts []string, _, _ string) ([]string, error) {
	b.calls++
	out := make([]string, len(texts))
	for i, s := range texts {
		out[i] = b.prefix + s
	}
	return out, nil
}

func TestBatchUsesBatchMethod(t *testing.T) {
	f := &batchFake{prefix: "T:"}
	got, err := Batch(context.Background(), f, []string{"a", "b", "c"}, "en", "be")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != "T:a" || got[2] != "T:c" {
		t.Fatalf("got %v", got)
	}
	if f.calls != 1 {
		t.Errorf("expected 1 batched call, got %d", f.calls)
	}
}

func TestBatchLoopsPlainTranslator(t *testing.T) {
	primary := &fake{out: "x"}
	got, err := Batch(context.Background(), primary, []string{"a", "b"}, "en", "be")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "x" {
		t.Fatalf("got %v", got)
	}
	if primary.calls != 2 {
		t.Errorf("expected 2 per-string calls, got %d", primary.calls)
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
