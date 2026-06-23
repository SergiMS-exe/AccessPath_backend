package gmaps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

const (
	autocompleteURL = "https://maps.googleapis.com/maps/api/place/autocomplete/json"
	detailsURL      = "https://maps.googleapis.com/maps/api/place/details/json"
	detailsFields   = "place_id,name,formatted_address,geometry,types"
)

var ErrNoAPIKey = errors.New("google maps api key not configured")

type Client struct {
	apiKey string
	http   *http.Client
}

func New(apiKey string) *Client {
	return &Client{apiKey: apiKey, http: &http.Client{}}
}

type AutocompleteItem struct {
	PlaceID       string `json:"place_id"`
	Description   string `json:"description"`
	MainText      string `json:"main_text"`
	SecondaryText string `json:"secondary_text"`
}

type PlaceDetails struct {
	PlaceID          string   `json:"place_id"`
	Name             string   `json:"name"`
	FormattedAddress string   `json:"formatted_address"`
	Lat              float64  `json:"lat"`
	Lng              float64  `json:"lng"`
	Types            []string `json:"types"`
}

func (c *Client) Autocomplete(ctx context.Context, query, sessionToken string) ([]AutocompleteItem, error) {
	if c.apiKey == "" {
		return nil, ErrNoAPIKey
	}

	params := url.Values{
		"input":        {query},
		"key":          {c.apiKey},
		"sessiontoken": {sessionToken},
		"language":     {"es"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, autocompleteURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body struct {
		Status      string `json:"status"`
		Predictions []struct {
			PlaceID     string `json:"place_id"`
			Description string `json:"description"`
			Structured  struct {
				MainText      string `json:"main_text"`
				SecondaryText string `json:"secondary_text"`
			} `json:"structured_formatting"`
		} `json:"predictions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.Status != "OK" && body.Status != "ZERO_RESULTS" {
		return nil, fmt.Errorf("google autocomplete: status %s", body.Status)
	}

	items := make([]AutocompleteItem, 0, len(body.Predictions))
	for _, p := range body.Predictions {
		items = append(items, AutocompleteItem{
			PlaceID:       p.PlaceID,
			Description:   p.Description,
			MainText:      p.Structured.MainText,
			SecondaryText: p.Structured.SecondaryText,
		})
	}
	return items, nil
}

func (c *Client) Details(ctx context.Context, placeID, sessionToken string) (*PlaceDetails, error) {
	if c.apiKey == "" {
		return nil, ErrNoAPIKey
	}

	params := url.Values{
		"place_id":     {placeID},
		"key":          {c.apiKey},
		"sessiontoken": {sessionToken},
		"fields":       {detailsFields},
		"language":     {"es"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, detailsURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body struct {
		Status string `json:"status"`
		Result struct {
			PlaceID          string   `json:"place_id"`
			Name             string   `json:"name"`
			FormattedAddress string   `json:"formatted_address"`
			Types            []string `json:"types"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.Status != "OK" {
		return nil, fmt.Errorf("google place details: status %s", body.Status)
	}

	return &PlaceDetails{
		PlaceID:          body.Result.PlaceID,
		Name:             body.Result.Name,
		FormattedAddress: body.Result.FormattedAddress,
		Lat:              body.Result.Geometry.Location.Lat,
		Lng:              body.Result.Geometry.Location.Lng,
		Types:            body.Result.Types,
	}, nil
}
