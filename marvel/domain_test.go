package marvel

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

// These tests are offline: they exercise the URI driver's pure string functions
// and the host wiring. The HTTP behaviour is covered in marvel_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "marvel" {
		t.Errorf("Scheme = %q, want marvel", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "marvel" {
		t.Errorf("Identity.Binary = %q, want marvel", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct{ in, typ, id string }{
		{"1011334", "character", "1011334"},
		{"comic:82967", "comic", "82967"},
		{"https://gateway.marvel.com/v1/public/characters/1011334", "character", "1011334"},
		{"https://gateway.marvel.com/v1/public/comics/82967", "comic", "82967"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil || typ != tc.typ || id != tc.id {
			t.Errorf("Classify(%q) = (%q, %q, %v), want (%q, %q, nil)",
				tc.in, typ, id, err, tc.typ, tc.id)
		}
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("character", "1011334")
	want := "https://www.marvel.com/characters/1011334"
	if err != nil || got != want {
		t.Errorf("Locate character = (%q, %v), want (%q, nil)", got, err, want)
	}

	got, err = Domain{}.Locate("comic", "82967")
	want = "https://gateway.marvel.com/v1/public/comics/82967"
	if err != nil || got != want {
		t.Errorf("Locate comic = (%q, %v), want (%q, nil)", got, err, want)
	}
}

// TestHostWiring mounts the driver in a kit Host and checks the round trip.
func TestHostWiring(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}

	ch := &Character{
		ID:   1011334,
		Name: "3-D Man",
		URL:  "https://www.marvel.com/characters/3-d-man",
	}
	u, err := h.Mint(ch)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	if want := "marvel://character/1011334"; u.String() != want {
		t.Errorf("Mint = %q, want %q", u.String(), want)
	}

	got, err := h.ResolveOn("marvel", "1234")
	if err != nil || got.String() != "marvel://character/1234" {
		t.Errorf("ResolveOn = (%q, %v), want marvel://character/1234", got.String(), err)
	}
}
