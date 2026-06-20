package marvel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestClient creates a Client pointing at the given test server.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	cfg := DefaultConfig()
	cfg.APIBaseURL = srv.URL
	cfg.PublicKey = "testpub"
	cfg.PrivateKey = "testpriv"
	cfg.Rate = 0
	cfg.Retries = 0
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, srv
}

// charResponse builds a minimal Marvel API character list response.
func charResponse(chars []map[string]any) []byte {
	resp := map[string]any{
		"code":   200,
		"status": "Ok",
		"data": map[string]any{
			"offset":  0,
			"limit":   len(chars),
			"total":   len(chars),
			"count":   len(chars),
			"results": chars,
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

// comicResponse builds a minimal Marvel API comic list response.
func comicResponse(comics []map[string]any) []byte {
	resp := map[string]any{
		"code":   200,
		"status": "Ok",
		"data": map[string]any{
			"offset":  0,
			"limit":   len(comics),
			"total":   len(comics),
			"count":   len(comics),
			"results": comics,
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

var sampleChar = map[string]any{
	"id":          1011334,
	"name":        "3-D Man",
	"description": "",
	"modified":    "2014-04-29T14:18:17-0400",
	"thumbnail":   map[string]any{"path": "http://example.com/img", "extension": "jpg"},
	"resourceURI": "http://example.com/chars/1011334",
	"comics":      map[string]any{"available": 12},
	"series":      map[string]any{"available": 3},
	"events":      map[string]any{"available": 1},
	"urls":        []map[string]any{{"type": "detail", "url": "https://marvel.com/chars/3d-man"}},
}

var sampleComic = map[string]any{
	"id":          82967,
	"title":       "Marvel Previews (2017)",
	"issueNumber": 0,
	"format":      "",
	"pageCount":   112,
	"description": nil,
	"isbn":        "",
	"modified":    "2019-11-07T11:30:15-0500",
	"thumbnail":   map[string]any{"path": "http://example.com/comic-img", "extension": "jpg"},
	"resourceURI": "http://example.com/comics/82967",
	"creators":    map[string]any{"available": 0},
	"characters":  map[string]any{"available": 0},
	"prices":      []map[string]any{{"type": "printPrice", "price": 0.0}},
	"urls":        []map[string]any{{"type": "detail", "url": "https://marvel.com/comics/82967"}},
}

func TestListCharacters(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(charResponse([]map[string]any{sampleChar}))
	})
	chars, err := c.ListCharacters(context.Background(), ListCharactersOpts{Limit: 5})
	if err != nil {
		t.Fatalf("ListCharacters: %v", err)
	}
	if len(chars) != 1 {
		t.Fatalf("got %d chars, want 1", len(chars))
	}
	if chars[0].Name != "3-D Man" {
		t.Errorf("name = %q, want %q", chars[0].Name, "3-D Man")
	}
	if chars[0].Comics != 12 {
		t.Errorf("comics = %d, want 12", chars[0].Comics)
	}
}

func TestListCharactersPassesNameStartsWith(t *testing.T) {
	gotParam := ""
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotParam = r.URL.Query().Get("nameStartsWith")
		w.Header().Set("Content-Type", "application/json")
		w.Write(charResponse(nil))
	})
	_, _ = c.ListCharacters(context.Background(), ListCharactersOpts{Limit: 5, NameStartsWith: "Spider"})
	if gotParam != "Spider" {
		t.Errorf("nameStartsWith = %q, want Spider", gotParam)
	}
}

func TestGetCharacter(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// Check the path contains the character id
		if r.URL.Path != fmt.Sprintf("/characters/%d", 1011334) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(charResponse([]map[string]any{sampleChar}))
	})
	ch, err := c.GetCharacter(context.Background(), 1011334)
	if err != nil {
		t.Fatalf("GetCharacter: %v", err)
	}
	if ch.Name != "3-D Man" {
		t.Errorf("name = %q, want %q", ch.Name, "3-D Man")
	}
}

func TestGetCharacterNotFound(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	_, err := c.GetCharacter(context.Background(), 999999)
	if err == nil {
		t.Fatal("expected error for not-found character, got nil")
	}
}

func TestListComics(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(comicResponse([]map[string]any{sampleComic}))
	})
	comics, err := c.ListComics(context.Background(), ListComicsOpts{Limit: 5})
	if err != nil {
		t.Fatalf("ListComics: %v", err)
	}
	if len(comics) != 1 {
		t.Fatalf("got %d comics, want 1", len(comics))
	}
	if comics[0].Title != "Marvel Previews (2017)" {
		t.Errorf("title = %q, want %q", comics[0].Title, "Marvel Previews (2017)")
	}
	if comics[0].PageCount != 112 {
		t.Errorf("pageCount = %d, want 112", comics[0].PageCount)
	}
}

func TestListComicsPassesTitleStartsWith(t *testing.T) {
	gotParam := ""
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotParam = r.URL.Query().Get("titleStartsWith")
		w.Header().Set("Content-Type", "application/json")
		w.Write(comicResponse(nil))
	})
	_, _ = c.ListComics(context.Background(), ListComicsOpts{Limit: 5, TitleStartsWith: "Spider-Man"})
	if gotParam != "Spider-Man" {
		t.Errorf("titleStartsWith = %q, want Spider-Man", gotParam)
	}
}

func TestGetComic(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(comicResponse([]map[string]any{sampleComic}))
	})
	cm, err := c.GetComic(context.Background(), 82967)
	if err != nil {
		t.Fatalf("GetComic: %v", err)
	}
	if cm.Title != "Marvel Previews (2017)" {
		t.Errorf("title = %q, want %q", cm.Title, "Marvel Previews (2017)")
	}
}

func TestMissingKeys(t *testing.T) {
	cfg := DefaultConfig()
	// No keys set
	_, err := NewClient(cfg)
	if err == nil {
		t.Fatal("expected error for missing keys, got nil")
	}
}

func TestAuthHashIncluded(t *testing.T) {
	gotQuery := ""
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write(charResponse(nil))
	})
	_, _ = c.ListCharacters(context.Background(), ListCharactersOpts{Limit: 1})
	q := gotQuery
	if q == "" {
		t.Fatal("no query string in request")
	}
	for _, p := range []string{"ts=", "apikey=", "hash="} {
		found := false
		for _, seg := range []string{q} {
			if len(seg) > 0 {
				_ = seg
				found = true
			}
		}
		_ = found
		if q == "" || (len(q) < len(p)) {
			t.Errorf("query string %q missing expected param %s", q, p)
		}
	}
	// Basic check: ts, apikey, hash all present
	for _, want := range []string{"ts=", "apikey=testpub", "hash="} {
		if !containsStr(q, want) {
			t.Errorf("query %q does not contain %q", q, want)
		}
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsRune(s, sub))
}

func containsRune(s, sub string) bool {
	for i := range s {
		if i+len(sub) <= len(s) && s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestThumbnailURL(t *testing.T) {
	thumb := apiThumbnail{Path: "http://example.com/img", Extension: "jpg"}
	got := thumbnailURL(thumb)
	want := "http://example.com/img.jpg"
	if got != want {
		t.Errorf("thumbnailURL = %q, want %q", got, want)
	}
}
