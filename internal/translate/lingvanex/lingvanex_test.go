package lingvanex

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stub mimics the Python server: echoes each newline-delimited input line
// prefixed with the language pair.
func stub() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var out []string
		for _, l := range strings.Split(string(b), "\n") {
			out = append(out, r.URL.Query().Get("to")+":"+l)
		}
		io.WriteString(w, strings.Join(out, "\n"))
	}))
}

func newClient(url string) *Client { return &Client{baseURL: url, http: http.DefaultClient} }

func TestTranslate(t *testing.T) {
	s := stub()
	defer s.Close()
	got, err := newClient(s.URL).Translate(context.Background(), "hello", "ru", "be")
	if err != nil {
		t.Fatal(err)
	}
	if got != "be:hello" {
		t.Errorf("got %q", got)
	}
}

func TestTranslateBatch(t *testing.T) {
	s := stub()
	defer s.Close()
	c := newClient(s.URL)

	got, err := c.TranslateBatch(context.Background(), []string{"one", "two", "three"}, "ru", "be")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"be:one", "be:two", "be:three"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTranslateBatchStripsNewlinesFromInput(t *testing.T) {
	s := stub()
	defer s.Close()
	// an input with an embedded newline would desync the line protocol; the
	// client must flatten it first so count in == count out.
	got, err := newClient(s.URL).TranslateBatch(context.Background(),
		[]string{"a\nb", "c"}, "ru", "be")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "be:a b" || got[1] != "be:c" {
		t.Errorf("got %v", got)
	}
}
