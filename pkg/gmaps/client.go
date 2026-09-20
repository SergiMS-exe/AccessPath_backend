package gmaps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"accesspath/pkg/apperr"
)

const (
	autocompleteURL = "https://maps.googleapis.com/maps/api/place/autocomplete/json"
	detailsURL      = "https://maps.googleapis.com/maps/api/place/details/json"
	detailsFields   = "place_id,name,formatted_address,geometry,types,business_status"

	// BusinessStatusClosedPermanently marca un sitio cerrado para siempre en
	// Google Places; no debe importarse a la BD.
	BusinessStatusClosedPermanently = "CLOSED_PERMANENTLY"
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
	// OPERATIONAL | CLOSED_TEMPORARILY | CLOSED_PERMANENTLY (vacio si Google no lo informa).
	BusinessStatus string `json:"business_status"`
}

// gmapsStatusHandler convierte un status textual de Google en un *apperr.AppError.
// Cada entrada es la unica fuente de verdad sobre como se traduce un status
// conocido: anadir un nuevo caso = anadir una linea. Si el status no esta
// en el map, se considera upstream error generico (502).
type gmapsStatusHandler func(op, status, errorMessage string) *apperr.AppError

// gmapsStatusHandlers es la tabla de dispatch. Mantenerla cerca de la constante
// de arriba (en este mismo archivo) facilita extenderla sin tocar la logica
// del metodo.
var gmapsStatusHandlers = map[string]gmapsStatusHandler{
	"REQUEST_DENIED": func(op, status, _ string) *apperr.AppError {
		return apperr.GmapsRequestDenied(op, fmt.Errorf("status=%s", status))
	},
	"OVER_QUERY_LIMIT": func(op, _, _ string) *apperr.AppError {
		return apperr.GmapsQuotaExceeded(op)
	},
	"INVALID_REQUEST": func(op, _, errorMessage string) *apperr.AppError {
		return apperr.GmapsInvalidRequest(op, errorMessage)
	},
}

func (c *Client) Autocomplete(ctx context.Context, query, sessionToken string) ([]AutocompleteItem, error) {
	if c.apiKey == "" {
		return nil, apperr.GmapsNotConfigured("gmaps.Autocomplete")
	}

	params := url.Values{
		"input":        {query},
		"key":          {c.apiKey},
		"sessiontoken": {sessionToken},
		"language":     {"es"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, autocompleteURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, apperr.GmapsNetwork("gmaps.Autocomplete", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, apperr.GmapsNetwork("gmaps.Autocomplete", err)
	}
	defer resp.Body.Close()

	var body struct {
		Status        string `json:"status"`
		ErrorMessage  string `json:"error_message"`
		Predictions   []struct {
			PlaceID     string `json:"place_id"`
			Description string `json:"description"`
			Structured  struct {
				MainText      string `json:"main_text"`
				SecondaryText string `json:"secondary_text"`
			} `json:"structured_formatting"`
		} `json:"predictions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, apperr.GmapsNetwork("gmaps.Autocomplete", err)
	}

	if body.Status != "OK" && body.Status != "ZERO_RESULTS" {
		if handler, ok := gmapsStatusHandlers[body.Status]; ok {
			return nil, handler("gmaps.Autocomplete", body.Status, body.ErrorMessage)
		}
		return nil, apperr.GmapsUpstream("gmaps.Autocomplete", body.Status,
			fmt.Errorf("error_message=%s", body.ErrorMessage))
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
		return nil, apperr.GmapsNotConfigured("gmaps.Details")
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
		return nil, apperr.GmapsNetwork("gmaps.Details", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, apperr.GmapsNetwork("gmaps.Details", err)
	}
	defer resp.Body.Close()

	var body struct {
		Status       string `json:"status"`
		ErrorMessage string `json:"error_message"`
		Result       struct {
			PlaceID          string   `json:"place_id"`
			Name             string   `json:"name"`
			FormattedAddress string   `json:"formatted_address"`
			Types            []string `json:"types"`
			BusinessStatus   string   `json:"business_status"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, apperr.GmapsNetwork("gmaps.Details", err)
	}
	if body.Status != "OK" {
		if handler, ok := gmapsStatusHandlers[body.Status]; ok {
			return nil, handler("gmaps.Details", body.Status, body.ErrorMessage)
		}
		return nil, apperr.GmapsUpstream("gmaps.Details", body.Status,
			fmt.Errorf("error_message=%s", body.ErrorMessage))
	}

	return &PlaceDetails{
		PlaceID:          body.Result.PlaceID,
		Name:             body.Result.Name,
		FormattedAddress: body.Result.FormattedAddress,
		Lat:              body.Result.Geometry.Location.Lat,
		Lng:              body.Result.Geometry.Location.Lng,
		Types:            body.Result.Types,
		BusinessStatus:   body.Result.BusinessStatus,
	}, nil
}