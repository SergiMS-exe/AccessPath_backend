package catalog

import (
	"strings"
	"unicode"
)

// Match es el resultado de cruzar una pista (nombre + coordenadas) con una
// fila del catalogo.
type Match struct {
	Item           Item
	DistanceMeters float64
	Score          float64
}

// BestMatch elige el mejor candidato de `items` para una pista (lat, lng,
// name). Devuelve false si ninguno cumple distancia y similitud minimas.
// El criterio es el del plan: < 50 m y score > 0.3.
func BestMatch(items []Item, lat, lng float64, name string, maxDistanceMeters, minScore float64) (Match, bool) {
	var best Match
	found := false

	for _, it := range items {
		dist := it.DistanceMeters(lat, lng)
		if dist > maxDistanceMeters {
			continue
		}
		score := NameScore(it.Name, name)
		if score < minScore {
			continue
		}
		if !found || score > best.Score || (score == best.Score && dist < best.DistanceMeters) {
			best = Match{Item: it, DistanceMeters: dist, Score: score}
			found = true
		}
	}
	return best, found
}

// NameScore compara dos nombres de lugar. Devuelve 0..1.
//
// No usa pg_trgm a proposito: el match se decide en Go para poder testearlo
// sin base de datos y para no depender de la extension en tiempo de ejecucion.
func NameScore(a, b string) float64 {
	na, nb := Normalize(a), Normalize(b)
	if na == "" || nb == "" {
		return 0
	}
	if na == nb {
		return 1
	}

	// "Sidreria de Diego" vs "Sidreria de Diego S.L." -> containment.
	if len([]rune(na)) >= 4 && len([]rune(nb)) >= 4 {
		if strings.Contains(na, nb) || strings.Contains(nb, na) {
			return 0.85
		}
	}

	// Fallback: solapamiento de palabras (Jaccard).
	ta, tb := tokens(na), tokens(nb)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	inter := 0
	for k := range ta {
		if _, ok := tb[k]; ok {
			inter++
		}
	}
	union := len(ta) + len(tb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// Normalize pasa el texto a minusculas, elimina acentos y puntuacion, y
// colapsa espacios. "  Cafetería   ECLIPSE " -> "cafeteria eclipse".
func Normalize(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	lastSpace := true
	for _, r := range strings.ToLower(s) {
		r = foldAccent(r)
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastSpace = false
		case !lastSpace:
			b.WriteRune(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

func tokens(s string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, f := range strings.Fields(s) {
		if len([]rune(f)) > 1 {
			out[f] = struct{}{}
		}
	}
	return out
}

func foldAccent(r rune) rune {
	switch r {
	case 'á', 'à', 'â', 'ä', 'ã', 'å':
		return 'a'
	case 'é', 'è', 'ê', 'ë':
		return 'e'
	case 'í', 'ì', 'î', 'ï':
		return 'i'
	case 'ó', 'ò', 'ô', 'ö', 'õ':
		return 'o'
	case 'ú', 'ù', 'û', 'ü':
		return 'u'
	case 'ñ':
		return 'n'
	case 'ç':
		return 'c'
	default:
		return r
	}
}
