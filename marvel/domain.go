package marvel

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// init registers the Domain so a blank import in a multi-domain host enables
// the marvel:// driver.
func init() { kit.Register(Domain{}) }

// Domain is the Marvel Comics API driver.
type Domain struct{}

// Info describes the scheme, hosts, and identity for the kit framework.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "marvel",
		Hosts:  []string{Host, GatewayHost, "www.marvel.com", "marvel.com"},
		Identity: kit.Identity{
			Binary: "marvel",
			Short:  "Read the Marvel Comics API",
			Long: `marvel reads the official Marvel Comics developer API.

Requires two environment variables:
  MARVEL_PUBLIC_KEY   your public key from developer.marvel.com
  MARVEL_PRIVATE_KEY  your private key for request signing

Quick start:
  marvel characters -n 10                    list 10 characters
  marvel characters --name-starts-with Spi   characters starting with "Spi"
  marvel character 1011334                   fetch 3-D Man
  marvel comics -n 5                         list 5 comics
  marvel comic 82967                         fetch a comic by id
  marvel search spider                       search characters and comics

Data provided by Marvel. (c) 2024 MARVEL`,
			Site: Host,
			Repo: "https://github.com/tamnd/marvel-cli",
		},
	}
}

// Register installs the client factory and all operations onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{
		Name:    "characters",
		Group:   "characters",
		List:    true,
		Summary: "List Marvel characters",
	}, listCharacters)

	kit.Handle(app, kit.OpMeta{
		Name:     "character",
		Group:    "characters",
		Single:   true,
		Resolver: true,
		URIType:  "character",
		Summary:  "Fetch a single character by ID",
		Args:     []kit.Arg{{Name: "id", Help: "character id (numeric)"}},
	}, getCharacter)

	kit.Handle(app, kit.OpMeta{
		Name:    "comics",
		Group:   "comics",
		List:    true,
		Summary: "List Marvel comics",
	}, listComics)

	kit.Handle(app, kit.OpMeta{
		Name:     "comic",
		Group:    "comics",
		Single:   true,
		Resolver: true,
		URIType:  "comic",
		Summary:  "Fetch a single comic by ID",
		Args:     []kit.Arg{{Name: "id", Help: "comic id (numeric)"}},
	}, getComic)

	kit.Handle(app, kit.OpMeta{
		Name:    "search",
		Group:   "search",
		List:    true,
		Summary: "Search characters and comics by name/title prefix",
		Args:    []kit.Arg{{Name: "query", Help: "search query"}},
	}, searchAll)
}

// newClient builds a Client from the kit-resolved Config and environment variables.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	c.FromEnv()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	client, err := NewClient(c)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// --- inputs ---

type listCharactersInput struct {
	Limit          int     `kit:"flag,inherit" help:"max results" default:"20"`
	NameStartsWith string  `kit:"flag" help:"name prefix filter"`
	Client         *Client `kit:"inject"`
}

type characterInput struct {
	ID     string  `kit:"arg" help:"character id (numeric)"`
	Client *Client `kit:"inject"`
}

type listComicsInput struct {
	Limit           int     `kit:"flag,inherit" help:"max results" default:"20"`
	TitleStartsWith string  `kit:"flag" help:"title prefix filter"`
	Client          *Client `kit:"inject"`
}

type comicInput struct {
	ID     string  `kit:"arg" help:"comic id (numeric)"`
	Client *Client `kit:"inject"`
}

type searchInput struct {
	Query  string  `kit:"arg" help:"search query"`
	Limit  int     `kit:"flag,inherit" help:"max per kind" default:"10"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func listCharacters(ctx context.Context, in listCharactersInput, emit func(Character) error) error {
	chars, err := in.Client.ListCharacters(ctx, ListCharactersOpts{
		Limit:          in.Limit,
		NameStartsWith: in.NameStartsWith,
	})
	if err != nil {
		return mapErr(err)
	}
	if len(chars) == 0 {
		return errs.NotFound("no characters found")
	}
	for _, ch := range chars {
		if err := emit(ch); err != nil {
			return err
		}
	}
	return nil
}

func getCharacter(ctx context.Context, in characterInput, emit func(*Character) error) error {
	id, err := strconv.Atoi(in.ID)
	if err != nil {
		return errs.Usage("character id must be numeric, got %q", in.ID)
	}
	ch, err := in.Client.GetCharacter(ctx, id)
	if err != nil {
		return mapErr(err)
	}
	return emit(ch)
}

func listComics(ctx context.Context, in listComicsInput, emit func(Comic) error) error {
	comics, err := in.Client.ListComics(ctx, ListComicsOpts{
		Limit:           in.Limit,
		TitleStartsWith: in.TitleStartsWith,
	})
	if err != nil {
		return mapErr(err)
	}
	if len(comics) == 0 {
		return errs.NotFound("no comics found")
	}
	for _, cm := range comics {
		if err := emit(cm); err != nil {
			return err
		}
	}
	return nil
}

func getComic(ctx context.Context, in comicInput, emit func(*Comic) error) error {
	id, err := strconv.Atoi(in.ID)
	if err != nil {
		return errs.Usage("comic id must be numeric, got %q", in.ID)
	}
	cm, err := in.Client.GetComic(ctx, id)
	if err != nil {
		return mapErr(err)
	}
	return emit(cm)
}

func searchAll(ctx context.Context, in searchInput, emit func(SearchResult) error) error {
	results, err := in.Client.Search(ctx, in.Query, in.Limit)
	if err != nil {
		return mapErr(err)
	}
	if len(results) == 0 {
		return errs.NotFound("no results for %q", in.Query)
	}
	for _, r := range results {
		if err := emit(r); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver ---

// Classify turns any accepted input into the canonical (uriType, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", errs.Usage("marvel: empty input")
	}
	// comic:<id>
	if strings.HasPrefix(input, "comic:") {
		return "comic", strings.TrimPrefix(input, "comic:"), nil
	}
	// full URL: https://gateway.marvel.com/v1/public/characters/1011334
	if u, err := url.Parse(input); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		segs := strings.Split(strings.Trim(u.Path, "/"), "/")
		for i, s := range segs {
			if (s == "characters" || s == "comics") && i+1 < len(segs) {
				utype := "character"
				if s == "comics" {
					utype = "comic"
				}
				return utype, segs[i+1], nil
			}
		}
	}
	// numeric default to character
	return "character", input, nil
}

// Locate returns the canonical web URL for a (uriType, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "character":
		return fmt.Sprintf("https://www.marvel.com/characters/%s", id), nil
	case "comic":
		return fmt.Sprintf("https://gateway.marvel.com/v1/public/comics/%s", id), nil
	}
	return "", errs.Usage("marvel has no resource type %q", uriType)
}

// mapErr converts library errors into kit error kinds with appropriate exit codes.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return errs.NotFound("%s", err.Error())
	}
	if errors.Is(err, ErrRateLimited) {
		return errs.RateLimited("%s", err.Error())
	}
	if errors.Is(err, ErrMissingKeys) {
		return errs.Usage("%s", err.Error())
	}
	return err
}
