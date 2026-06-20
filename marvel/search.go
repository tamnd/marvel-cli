package marvel

import (
	"context"
	"fmt"
)

// Search searches characters by name prefix and comics by title prefix,
// returning merged SearchResult records.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	var out []SearchResult

	chars, err := c.ListCharacters(ctx, ListCharactersOpts{
		Limit:          limit,
		NameStartsWith: query,
	})
	if err == nil {
		for _, ch := range chars {
			out = append(out, SearchResult{
				Kind: "character",
				ID:   ch.ID,
				Name: ch.Name,
				URL:  ch.URL,
			})
		}
	}

	comics, err := c.ListComics(ctx, ListComicsOpts{
		Limit:           limit,
		TitleStartsWith: query,
	})
	if err == nil {
		for _, cm := range comics {
			out = append(out, SearchResult{
				Kind: "comic",
				ID:   cm.ID,
				Name: cm.Title,
				URL:  fmt.Sprintf("https://gateway.marvel.com/v1/public/comics/%d", cm.ID),
			})
		}
	}

	return out, nil
}
