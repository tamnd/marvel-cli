package marvel

import "encoding/json"

// Character is a Marvel character record.
type Character struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Comics      int    `json:"comics"`
	Series      int    `json:"series"`
	Events      int    `json:"events"`
	Modified    string `json:"modified"`
	URL         string `json:"url"`
	Thumbnail   string `json:"thumbnail"`
}

// Comic is a Marvel comic record.
type Comic struct {
	ID              int     `json:"id"`
	Title           string  `json:"title"`
	IssueNumber     float64 `json:"issue_number"`
	Format          string  `json:"format"`
	PageCount       int     `json:"page_count"`
	Description     string  `json:"description"`
	ISBN            string  `json:"isbn"`
	CreatorsCount   int     `json:"creators"`
	CharactersCount int     `json:"characters"`
	Price           float64 `json:"price"`
	Modified        string  `json:"modified"`
	URL             string  `json:"url"`
	Thumbnail       string  `json:"thumbnail"`
}

// SearchResult is a merged record from searching both characters and comics.
type SearchResult struct {
	Kind string `json:"kind"`
	ID   int    `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// --- raw API response types ---

type apiResponse struct {
	Code   int    `json:"code"`
	Status string `json:"status"`
	Data   struct {
		Offset  int             `json:"offset"`
		Limit   int             `json:"limit"`
		Total   int             `json:"total"`
		Count   int             `json:"count"`
		Results json.RawMessage `json:"results"`
	} `json:"data"`
}

type apiThumbnail struct {
	Path      string `json:"path"`
	Extension string `json:"extension"`
}

type apiURL struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type apiAvailable struct {
	Available int `json:"available"`
}

type apiCharacter struct {
	ID          int          `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Modified    string       `json:"modified"`
	Thumbnail   apiThumbnail `json:"thumbnail"`
	ResourceURI string       `json:"resourceURI"`
	Comics      apiAvailable `json:"comics"`
	Series      apiAvailable `json:"series"`
	Events      apiAvailable `json:"events"`
	URLs        []apiURL     `json:"urls"`
}

type apiComic struct {
	ID          int          `json:"id"`
	Title       string       `json:"title"`
	IssueNumber float64      `json:"issueNumber"`
	Format      string       `json:"format"`
	PageCount   int          `json:"pageCount"`
	Description *string      `json:"description"`
	ISBN        string       `json:"isbn"`
	Modified    string       `json:"modified"`
	Thumbnail   apiThumbnail `json:"thumbnail"`
	ResourceURI string       `json:"resourceURI"`
	Creators    apiAvailable `json:"creators"`
	Characters  apiAvailable `json:"characters"`
	Prices      []struct {
		Type  string  `json:"type"`
		Price float64 `json:"price"`
	} `json:"prices"`
	URLs []apiURL `json:"urls"`
}

// thumbnailURL builds a full image URL from the Marvel thumbnail object.
func thumbnailURL(t apiThumbnail) string {
	if t.Path == "" {
		return ""
	}
	return t.Path + "." + t.Extension
}

// extractURL returns the detail URL from the Marvel URLs array.
func extractURL(urls []apiURL) string {
	for _, u := range urls {
		if u.Type == "detail" {
			return u.URL
		}
	}
	if len(urls) > 0 {
		return urls[0].URL
	}
	return ""
}

// extractPrice returns the print price from Marvel's prices array.
func extractPrice(prices []struct {
	Type  string  `json:"type"`
	Price float64 `json:"price"`
}) float64 {
	for _, p := range prices {
		if p.Type == "printPrice" {
			return p.Price
		}
	}
	return 0
}

// toCharacter converts a raw API character to the public type.
func (a apiCharacter) toCharacter() Character {
	return Character{
		ID:          a.ID,
		Name:        a.Name,
		Description: a.Description,
		Comics:      a.Comics.Available,
		Series:      a.Series.Available,
		Events:      a.Events.Available,
		Modified:    a.Modified,
		URL:         extractURL(a.URLs),
		Thumbnail:   thumbnailURL(a.Thumbnail),
	}
}

// toComic converts a raw API comic to the public type.
func (a apiComic) toComic() Comic {
	desc := ""
	if a.Description != nil {
		desc = *a.Description
	}
	return Comic{
		ID:              a.ID,
		Title:           a.Title,
		IssueNumber:     a.IssueNumber,
		Format:          a.Format,
		PageCount:       a.PageCount,
		Description:     desc,
		ISBN:            a.ISBN,
		CreatorsCount:   a.Creators.Available,
		CharactersCount: a.Characters.Available,
		Price:           extractPrice(a.Prices),
		Modified:        a.Modified,
		URL:             extractURL(a.URLs),
		Thumbnail:       thumbnailURL(a.Thumbnail),
	}
}
