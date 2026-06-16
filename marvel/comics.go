package marvel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// ListComicsOpts holds optional filters for ListComics.
type ListComicsOpts struct {
	Limit           int
	TitleStartsWith string
	Title           string // exact title match
}

// ListComics fetches up to opts.Limit comics from the Marvel API.
func (c *Client) ListComics(ctx context.Context, opts ListComicsOpts) ([]Comic, error) {
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	var out []Comic
	offset := 0
	pageSize := 100
	if opts.Limit < pageSize {
		pageSize = opts.Limit
	}

	for len(out) < opts.Limit {
		batch, total, err := c.listComicsPage(ctx, offset, pageSize, opts)
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

func (c *Client) listComicsPage(ctx context.Context, offset, limit int, opts ListComicsOpts) ([]Comic, int, error) {
	u, err := url.Parse(c.cfg.APIBaseURL + "/comics")
	if err != nil {
		return nil, 0, err
	}
	q := u.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	if opts.TitleStartsWith != "" {
		q.Set("titleStartsWith", opts.TitleStartsWith)
	}
	if opts.Title != "" {
		q.Set("title", opts.Title)
	}
	u.RawQuery = q.Encode()

	var resp apiResponse
	if err := c.getJSON(ctx, u.String(), &resp); err != nil {
		return nil, 0, err
	}
	if resp.Code != 200 && resp.Code != 0 {
		return nil, 0, fmt.Errorf("marvel API error %d: %s", resp.Code, resp.Status)
	}

	var raw []apiComic
	if err := json.Unmarshal(resp.Data.Results, &raw); err != nil {
		return nil, 0, fmt.Errorf("decode comics: %w", err)
	}
	out := make([]Comic, len(raw))
	for i, r := range raw {
		out[i] = r.toComic()
	}
	return out, resp.Data.Total, nil
}

// GetComic fetches a single comic by numeric ID.
func (c *Client) GetComic(ctx context.Context, id int) (*Comic, error) {
	rawURL := fmt.Sprintf("%s/comics/%d", c.cfg.APIBaseURL, id)
	var resp apiResponse
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}

	var raw []apiComic
	if err := json.Unmarshal(resp.Data.Results, &raw); err != nil {
		return nil, fmt.Errorf("decode comic: %w", err)
	}
	if len(raw) == 0 {
		return nil, ErrNotFound
	}
	cm := raw[0].toComic()
	return &cm, nil
}
