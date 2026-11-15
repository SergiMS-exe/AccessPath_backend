package catalog

import (
	"math"
	"time"
)

// Item es una entrada del catalogo externo de lugares. Se usa tanto para el
// escaneo de filas (tags `db`) como para la respuesta JSON de los endpoints
// de busqueda y resolucion de enlaces (tags `json`).
type Item struct {
	Source          string     `db:"source" json:"source"`
	ExternalID      string     `db:"id" json:"external_id"`
	Name            string     `db:"name" json:"name"`
	Address         *string    `db:"address" json:"address,omitempty"`
	City            *string    `db:"city" json:"city,omitempty"`
	Region          *string    `db:"region" json:"region,omitempty"`
	Postcode        *string    `db:"postcode" json:"postcode,omitempty"`
	Country         *string    `db:"country" json:"country,omitempty"`
	Latitude        float64    `db:"latitude" json:"latitude"`
	Longitude       float64    `db:"longitude" json:"longitude"`
	Category        *string    `db:"category" json:"category,omitempty"`
	Subcategory     *string    `db:"subcategory" json:"subcategory,omitempty"`
	Confidence      *float64   `db:"confidence" json:"confidence,omitempty"`
	OperatingStatus *string    `db:"operating_status" json:"operating_status,omitempty"`
	Phones          []string   `db:"phones" json:"phones,omitempty"`
	Websites        []string   `db:"websites" json:"websites,omitempty"`
	RefreshedAt     *time.Time `db:"refreshed_at" json:"refreshed_at,omitempty"`
}

// SecondaryText construye la linea secundaria que se muestra bajo el nombre
// en el buscador: direccion y ciudad cuando existen.
func (i Item) SecondaryText() string {
	out := ""
	if i.Address != nil && *i.Address != "" {
		out = *i.Address
	}
	if i.City != nil && *i.City != "" {
		if out != "" {
			out += ", "
		}
		out += *i.City
	}
	return out
}

const earthRadiusMeters = 6371000.0

// DistanceMeters devuelve la distancia en metros entre el item y un punto,
// usando la formula de Haversine.
func (i Item) DistanceMeters(lat, lng float64) float64 {
	return HaversineMeters(i.Latitude, i.Longitude, lat, lng)
}

// HaversineMeters calcula la distancia en metros entre dos coordenadas WGS84.
// Se usa en Go (y no en SQL) para poder decidir el match con tests unitarios
// sin base de datos.
func HaversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusMeters * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// BoundingBox devuelve un cuadrado (en grados) que contiene el circulo de
// radio `meters` alrededor de (lat, lng). Se usa para reducir el numero de
// filas que Postgres tiene que ordenar por similitud.
func BoundingBox(lat, lng, meters float64) (minLat, maxLat, minLng, maxLng float64) {
	latDelta := meters / 111320.0
	cos := math.Cos(lat * math.Pi / 180)
	if cos < 0.01 { // cerca de los polos: evita divisiones absurdas
		cos = 0.01
	}
	lngDelta := meters / (111320.0 * cos)
	return lat - latDelta, lat + latDelta, lng - lngDelta, lng + lngDelta
}
