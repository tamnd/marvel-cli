package marvel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// ListCharactersOpts holds optional filters for ListCharacters.
type ListCharactersOpts struct {
	Limit           int
	NameStartsWith  string
	Name            string // exact name match
}

// ListCharacters fetches up to opts.Limit characters from the Marvel API.
func (c *Client) ListCharacters(ctx context.Context, opts ListCharactersOpts) ([]Character, error) {
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	var out []Character
	offset := 0
	pageSize := 100
	if opts.Limit < pageSize {
		pageSize = opts.Limit
	}

	for len(out) < opts.Limit {
		batch, total, err := c.listCharactersPage(ctx, offset, pageSize, opts)
		if err != nil {
			return out, err
		}
		out = append(out, batch...)
		offset += len(batch)
		if offset >= total || len(batch) == 0 {
			break
		}
	}
	if len(out) > opts.Limit {
		out = out[:opts.Limit]
	}
	return out, nil
}

func (c *Client) listCharactersPage(ctx context.Context, offset, limit int, opts ListCharactersOpts) ([]Character, int, error) {
	u, err := url.Parse(c.cfg.APIBaseURL + "/characters")
	if err != nil {
		return nil, 0, err
	}
	q := u.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	if opts.NameStartsWith != "" {
		q.Set("nameStartsWith", opts.NameStartsWith)
	}
	if opts.Name != "" {
		q.Set("name", opts.Name)
	}
	u.RawQuery = q.Encode()

	var resp apiResponse
	if err := c.getJSON(ctx, u.String(), &resp); err != nil {
		return nil, 0, err
	}
	if resp.Code != 200 && resp.Code != 0 {
		return nil, 0, fmt.Errorf("marvel API error %d: %s", resp.Code, resp.Status)
	}

	var raw []apiCharacter
	if err := json.Unmarshal(resp.Data.Results, &raw); err != nil {
		return nil, 0, fmt.Errorf("decode characters: %w", err)
	}
	out := make([]Character, len(raw))
	for i, r := range raw {
		out[i] = r.toCharacter()
	}
	return out, resp.Data.Total, nil
}

// GetCharacter fetches a single character by numeric ID.
func (c *Client) GetCharacter(ctx context.Context, id int) (*Character, error) {
	rawURL := fmt.Sprintf("%s/characters/%d", c.cfg.APIBaseURL, id)
	var resp apiResponse
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}

	var raw []apiCharacter
	if err := json.Unmarshal(resp.Data.Results, &raw); err != nil {
		return nil, fmt.Errorf("decode character: %w", err)
	}
	if len(raw) == 0 {
		return nil, ErrNotFound
	}
	ch := raw[0].toCharacter()
	return &ch, nil
}
