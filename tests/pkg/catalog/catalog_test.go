package catalog_test

import (
	"testing"

	"accesspath/pkg/catalog"

	"github.com/stretchr/testify/assert"
)

func TestNormalize_SpanishText(t *testing.T) {
	cases := map[string]string{
		"  Cafetería   ECLIPSE ":                 "cafeteria eclipse",
		"EL BOSQUE Centro de Negocios|Coworking": "el bosque centro de negocios coworking",
		"Sidrería de Diego, S.L.":                "sidreria de diego s l",
		"La Tayuela (Sidrería)":                  "la tayuela sidreria",
		"Peña Tú":                                "pena tu",
		"":                                       "",
	}
	for in, want := range cases {
		assert.Equal(t, want, catalog.Normalize(in), "entrada: %q", in)
	}
}

func TestNameScore(t *testing.T) {
	assert.InDelta(t, 1.0, catalog.NameScore("Cafetería Eclipse", "cafeteria eclipse"), 1e-9)
	assert.InDelta(t, 0.85, catalog.NameScore("Sidreria de Diego", "Sidreria de Diego S.L."), 1e-9)
	assert.Greater(t, catalog.NameScore("El Bosque Centro de Negocios", "El Bosque Coworking"), 0.2)
	assert.Zero(t, catalog.NameScore("", "Bar"))
	assert.Less(t, catalog.NameScore("Farmacia", "Bar Pepe"), 0.3)
}

func TestHaversineMeters(t *testing.T) {
	// Mismo punto.
	assert.InDelta(t, 0.0, catalog.HaversineMeters(43.4021, -5.804946, 43.4021, -5.804946), 1e-6)

	// 0.001 grados de latitud ~ 111.2 m.
	d := catalog.HaversineMeters(43.4021, -5.804946, 43.4031, -5.804946)
	assert.InDelta(t, 111.2, d, 2.0)
}

func TestBoundingBox_ContainsPointAndNeighbours(t *testing.T) {
	minLat, maxLat, minLng, maxLng := catalog.BoundingBox(43.4021, -5.804946, 50)

	assert.Less(t, minLat, 43.4021)
	assert.Greater(t, maxLat, 43.4021)
	assert.Less(t, minLng, -5.804946)
	assert.Greater(t, maxLng, -5.804946)
	// Un punto a ~30 m sigue dentro.
	assert.Greater(t, maxLat, 43.4023)
}

func TestBestMatch_PrefersCloseAndSimilar(t *testing.T) {
	items := []catalog.Item{
		{ExternalID: "lejos", Name: "Cafetería Eclipse", Latitude: 43.4050, Longitude: -5.804946}, // ~320 m
		{ExternalID: "cerca", Name: "Cafeteria Eclipse", Latitude: 43.40215, Longitude: -5.804946},
		{ExternalID: "otro", Name: "Bar Pepe", Latitude: 43.40211, Longitude: -5.804946},
	}

	m, ok := catalog.BestMatch(items, 43.4021, -5.804946, "Cafetería Eclipse", 50, 0.3)
	assert.True(t, ok)
	assert.Equal(t, "cerca", m.Item.ExternalID)
	assert.Less(t, m.DistanceMeters, 50.0)
	assert.InDelta(t, 1.0, m.Score, 1e-9)
}

func TestBestMatch_RejectsFarOrDifferent(t *testing.T) {
	items := []catalog.Item{
		{ExternalID: "lejos", Name: "Cafetería Eclipse", Latitude: 43.4050, Longitude: -5.804946},
		{ExternalID: "otro", Name: "Bar Pepe", Latitude: 43.4021, Longitude: -5.804946},
	}

	_, ok := catalog.BestMatch(items, 43.4021, -5.804946, "Cafetería Eclipse", 50, 0.3)
	assert.False(t, ok)
}

func TestItemSecondaryText(t *testing.T) {
	addr := "Calle Leopoldo Lugones, 24"
	city := "Lugones"

	assert.Equal(t, "Calle Leopoldo Lugones, 24, Lugones",
		catalog.Item{Address: &addr, City: &city}.SecondaryText())
	assert.Equal(t, "", catalog.Item{}.SecondaryText())
}
