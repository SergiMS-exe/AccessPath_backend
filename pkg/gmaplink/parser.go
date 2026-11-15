// Package gmaplink lee enlaces de Google Maps compartidos por el usuario y
// extrae nombre, coordenadas y feature id (ftid).
//
// No descarga la ficha del sitio ni usa ninguna API de Google: el enlace ya
// contiene esos datos. Solo se sigue la redireccion de los enlaces cortos
// (maps.app.goo.gl) para llegar a la URL canonica.
package gmaplink

import (
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// Errores de validacion de enlaces.
var (
	ErrInvalidURL      = errors.New("invalid url")
	ErrUnsupportedURL  = errors.New("unsupported url scheme")
	ErrHostNotAllowed  = errors.New("host not allowed")
	ErrNoLinkData      = errors.New("link does not contain place data")
	ErrTooManyRedirect = errors.New("too many redirects")
	ErrBlockedAddress  = errors.New("connection to a non-public address blocked")
)

// Tipos de enlace reconocidos.
const (
	KindPlace     = "place"
	KindSearch    = "search"
	KindDirection = "directions"
	KindShort     = "short"
	KindUnknown   = "unknown"
)

// MaxURLLength limita el tamano del enlace aceptado (evita abusos y logs
// gigantes). Los enlaces canonicos de Google con parametro data= pueden ser
// largos, pero no suelen pasar de 1 KB.
const MaxURLLength = 2048

var (
	rePlacePath = regexp.MustCompile(`/maps/place/([^/]+)`)
	reFTID      = regexp.MustCompile(`!1s(0x[0-9a-fA-F]+:0x[0-9a-fA-F]+)`)
	reDataCoord = regexp.MustCompile(`!3d(-?\d+(?:\.\d+)?)!4d(-?\d+(?:\.\d+)?)`)
	reAtCoord   = regexp.MustCompile(`@(-?\d+(?:\.\d+)?),(-?\d+(?:\.\d+)?)`)
	reLatLng    = regexp.MustCompile(`^(-?\d+(?:\.\d+)?),\s*(-?\d+(?:\.\d+)?)$`)
	reGoogleTLD = regexp.MustCompile(`^(?:www|maps|consent)\.google\.(?:com|[a-z]{2,3}(?:\.[a-z]{2})?)$`)
)

// shortHosts son los acortadores que hay que expandir con una peticion HTTP.
var shortHosts = map[string]struct{}{
	"maps.app.goo.gl": {},
	"goo.gl":          {},
}

// Link es el resultado de leer un enlace de Google Maps.
type Link struct {
	// URL es la URL canonica (tras expandir el acortador, si aplica).
	URL string
	// Kind indica el tipo de enlace: place, search, directions, short, unknown.
	Kind string
	// Name es el nombre del sitio, si el enlace lo incluye.
	Name string
	// Lat/Lng son las coordenadas exactas del sitio (nil si no constan).
	Lat *float64
	Lng *float64
	// FTID es el feature id interno de Google (0x...:0x...), util solo para
	// deduplicar. No se usa para consultar a Google.
	FTID string
}

// HasCoords informa de si el enlace trae coordenadas.
func (l *Link) HasCoords() bool { return l != nil && l.Lat != nil && l.Lng != nil }

// HasData informa de si hay algo que ofrecer al usuario (nombre o coords).
func (l *Link) HasData() bool {
	return l != nil && (l.Name != "" || l.HasCoords() || l.FTID != "")
}

// IsShort indica que el enlace necesita seguir la redireccion.
func (l *Link) IsShort() bool { return l != nil && l.Kind == KindShort }

// Parse valida y parsea una URL de Google Maps. No hace peticiones de red:
// para enlaces cortos devuelve Kind=short y hay que usar Resolver.Resolve.
func Parse(raw string) (*Link, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > MaxURLLength {
		return nil, ErrInvalidURL
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, ErrInvalidURL
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, ErrUnsupportedURL
	}
	if !AllowedHost(u.Hostname()) {
		return nil, ErrHostNotAllowed
	}

	// Normalizamos a https: el resolver solo habla https.
	u.Scheme = "https"
	if _, ok := shortHosts[strings.ToLower(u.Hostname())]; ok {
		return &Link{URL: u.String(), Kind: KindShort}, nil
	}

	link := &Link{URL: u.String(), Kind: kindOf(u.Path)}
	link.Name = extractName(u)

	// El parametro data= viaja dentro del path (/maps/place/X/@lat,lng,17z/data=!...).
	if c := reFTID.FindStringSubmatch(u.Path + "!" + u.RawQuery + "!" + u.Fragment); len(c) == 2 {
		link.FTID = c[1]
	} else if ftid := u.Query().Get("ftid"); ftid != "" {
		link.FTID = ftid
	}

	// En /maps/dir/ las coordenadas !3d/!4d son de la ruta, no de un sitio.
	if link.Kind != KindDirection {
		if lat, lng, ok := matchCoords(reDataCoord, u.RawQuery+"!"+u.Path+"!"+u.Fragment); ok {
			link.Lat, link.Lng = lat, lng
		}
	}
	if !link.HasCoords() {
		if lat, lng, ok := matchCoords(reAtCoord, u.Path+"!"+u.RawQuery); ok {
			link.Lat, link.Lng = lat, lng
		}
	}
	if !link.HasCoords() {
		if lat, lng, ok := coordsFromQuery(u.Query()); ok {
			link.Lat, link.Lng = lat, lng
		}
	}

	return link, nil
}

// AllowedHost indica si el host pertenece a Google Maps (o a sus acortadores).
// Se aplica tanto a la URL de entrada como a cada salto de redireccion.
func AllowedHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if host == "" {
		return false
	}
	if _, ok := shortHosts[host]; ok {
		return true
	}
	if host == "google.com" || strings.HasSuffix(host, ".google.com") {
		return true
	}
	// Dominios por pais: google.es, google.co.uk, google.com.ar...
	if reGoogleTLD.MatchString(host) {
		return true
	}
	return strings.HasSuffix(host, ".google.com")
}

func kindOf(path string) string {
	switch {
	case strings.Contains(path, "/maps/place"):
		return KindPlace
	case strings.Contains(path, "/maps/search"):
		return KindSearch
	case strings.Contains(path, "/maps/dir"):
		return KindDirection
	default:
		return KindUnknown
	}
}

func extractName(u *url.URL) string {
	if m := rePlacePath.FindStringSubmatch(u.Path); len(m) == 2 {
		if name, err := url.PathUnescape(m[1]); err == nil {
			return cleanText(name)
		}
		return cleanText(m[1])
	}
	// Enlaces de busqueda: ?q=Cafeteria+Eclipse
	if q := u.Query().Get("q"); q != "" {
		if reLatLng.MatchString(q) {
			return ""
		}
		return cleanText(q)
	}
	if q := u.Query().Get("query"); q != "" {
		if reLatLng.MatchString(q) {
			return ""
		}
		return cleanText(q)
	}
	return ""
}

func coordsFromQuery(q url.Values) (*float64, *float64, bool) {
	keys := []string{"query", "q", "ll", "destination", "center"}
	for _, k := range keys {
		raw := strings.TrimSpace(q.Get(k))
		if raw == "" {
			continue
		}
		if m := reLatLng.FindStringSubmatch(raw); len(m) == 3 {
			if lat, lng, ok := toCoords(m[1], m[2]); ok {
				return lat, lng, true
			}
		}
	}
	return nil, nil, false
}

func matchCoords(re *regexp.Regexp, s string) (*float64, *float64, bool) {
	m := re.FindStringSubmatch(s)
	if len(m) != 3 {
		return nil, nil, false
	}
	return toCoords(m[1], m[2])
}

func toCoords(latRaw, lngRaw string) (*float64, *float64, bool) {
	lat, errLat := strconv.ParseFloat(latRaw, 64)
	lng, errLng := strconv.ParseFloat(lngRaw, 64)
	if errLat != nil || errLng != nil {
		return nil, nil, false
	}
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return nil, nil, false
	}
	return &lat, &lng, true
}

// cleanText decodifica el nombre y normaliza espacios/simbolos de relleno.
func cleanText(s string) string {
	s = strings.ReplaceAll(s, "+", " ")
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = strings.TrimSpace(s)
	// Google a veces concatena categorias con "|" en el nombre del path.
	s = strings.ReplaceAll(s, "|", " ")
	return strings.Join(strings.Fields(s), " ")
}
