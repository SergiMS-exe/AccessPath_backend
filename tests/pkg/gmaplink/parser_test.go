package gmaplink_test

import (
	"math"
	"strings"
	"testing"

	"accesspath/pkg/gmaplink"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// URL real resuelta a partir de https://maps.app.goo.gl/43FKHoBREWRVfGwe8
const eclipseURL = "https://www.google.com/maps/place/Cafeter%C3%ADa+Eclipse/@43.4020665,-5.8097546,17z/data=!3m1!5s0xd368bf04a1c9b2d:0xb36e144b6e491a48!4m7!3m6!1s0xd368bf0484e32bf:0xb9556be5462221b7!8m2!3d43.4020665!4d-5.804991!15sChFDYWZldGVyw61hcyBTaWVyb5IBBGNhZmWaASNDaFpEU1VoTk1HOW5TMFZKUTBGblNVTlBORGsyYW1SbkVBReABAA!16s%2Fg%2F12qf__wwz?hl=es&entry=tts"

func TestParse_FullPlaceURL(t *testing.T) {
	link, err := gmaplink.Parse(eclipseURL)
	require.NoError(t, err)

	assert.Equal(t, gmaplink.KindPlace, link.Kind)
	assert.Equal(t, "Cafetería Eclipse", link.Name)
	require.NotNil(t, link.Lat)
	require.NotNil(t, link.Lng)
	assert.InDelta(t, 43.4020665, *link.Lat, 1e-7)
	assert.InDelta(t, -5.804991, *link.Lng, 1e-7)
	assert.Equal(t, "0xd368bf0484e32bf:0xb9556be5462221b7", link.FTID)
	assert.True(t, link.HasData())
	assert.False(t, link.IsShort())
}

func TestParse_PrefersPlaceCoordsOverMapCenter(t *testing.T) {
	// @... es el centro del mapa; !3d/!4d son las coordenadas del sitio.
	link, err := gmaplink.Parse(eclipseURL)
	require.NoError(t, err)
	require.NotNil(t, link.Lng)
	assert.InDelta(t, -5.804991, *link.Lng, 1e-7)
	assert.Greater(t, math.Abs(*link.Lng-(-5.8097546)), 1e-3, "no debe usar el centro del mapa")
}

func TestParse_NameWithPipes(t *testing.T) {
	raw := "https://www.google.com/maps/place/EL+BOSQUE+Centro+de+Negocios%7CCoworking/@43.4016235,-5.8150287,17z/data=!4m7!3m6!1s0xd368b7fefa99d59:0x394442f94bc0feca!8m2!3d43.4016235!4d-5.8102651"
	link, err := gmaplink.Parse(raw)
	require.NoError(t, err)

	assert.Equal(t, "EL BOSQUE Centro de Negocios Coworking", link.Name)
	require.NotNil(t, link.Lat)
	assert.InDelta(t, -5.8102651, *link.Lng, 1e-7)
	assert.Equal(t, "0xd368b7fefa99d59:0x394442f94bc0feca", link.FTID)
}

func TestParse_ShortLinkIsMarkedForExpansion(t *testing.T) {
	link, err := gmaplink.Parse("https://maps.app.goo.gl/43FKHoBREWRVfGwe8")
	require.NoError(t, err)

	assert.True(t, link.IsShort())
	assert.Equal(t, gmaplink.KindShort, link.Kind)
	assert.False(t, link.HasData())
}

func TestParse_SearchURLWithText(t *testing.T) {
	link, err := gmaplink.Parse("https://www.google.com/maps/search/?api=1&query=Cafeter%C3%ADa+Eclipse")
	require.NoError(t, err)

	assert.Equal(t, gmaplink.KindSearch, link.Kind)
	assert.Equal(t, "Cafetería Eclipse", link.Name)
	assert.False(t, link.HasCoords())
}

func TestParse_SearchURLWithCoordsOnly(t *testing.T) {
	link, err := gmaplink.Parse("https://www.google.com/maps/search/?api=1&query=43.4020665,-5.804991")
	require.NoError(t, err)

	assert.Empty(t, link.Name)
	require.NotNil(t, link.Lat)
	require.NotNil(t, link.Lng)
	assert.InDelta(t, 43.4020665, *link.Lat, 1e-7)
	assert.InDelta(t, -5.804991, *link.Lng, 1e-7)
}

func TestParse_QueryWithFTIDParam(t *testing.T) {
	link, err := gmaplink.Parse("https://www.google.com/maps/place/X/?ftid=0xaa:0xbb&q=Bar")
	require.NoError(t, err)
	assert.Equal(t, "0xaa:0xbb", link.FTID)
}

func TestParse_DirectionsLinkDoesNotExposePlaceCoords(t *testing.T) {
	raw := "https://www.google.com/maps/dir/Bonn/Berlin/@51.6456171,7.9144552,7z/data=!4m13!4m12!1m5!1m1!1s0x47bee19f7ccbda49:0x86dbf8c6685c9617!2m2!1d7.0982068!2d50.73743!1m5!1m1!1s0x47a84e373f035901:0x42120465b5e3b70!2m2!1d13.404954!2d52.5200066"
	link, err := gmaplink.Parse(raw)
	require.NoError(t, err)
	assert.Equal(t, gmaplink.KindDirection, link.Kind)
	assert.Empty(t, link.Name)
}

func TestParse_CountryDomainIsAllowed(t *testing.T) {
	link, err := gmaplink.Parse("https://maps.google.es/maps/place/Bar/@43.4,-5.8,17z")
	require.NoError(t, err)
	assert.Equal(t, gmaplink.KindPlace, link.Kind)
	assert.Equal(t, "Bar", link.Name)
}

func TestParse_RejectsForeignHost(t *testing.T) {
	_, err := gmaplink.Parse("https://evil.example.com/maps/place/X/@1,2,3z")
	require.ErrorIs(t, err, gmaplink.ErrHostNotAllowed)
}

func TestParse_RejectsLookalikeHost(t *testing.T) {
	_, err := gmaplink.Parse("https://maps.app.goo.gl.evil.com/abc")
	require.ErrorIs(t, err, gmaplink.ErrHostNotAllowed)
}

func TestParse_RejectsNonHTTPScheme(t *testing.T) {
	_, err := gmaplink.Parse("javascript:alert(1)")
	require.ErrorIs(t, err, gmaplink.ErrUnsupportedURL)
}

func TestParse_RejectsEmptyAndOversizedURL(t *testing.T) {
	_, err := gmaplink.Parse("")
	require.ErrorIs(t, err, gmaplink.ErrInvalidURL)

	long := "https://www.google.com/maps/place/" + strings.Repeat("a", gmaplink.MaxURLLength)
	_, err = gmaplink.Parse(long)
	require.ErrorIs(t, err, gmaplink.ErrInvalidURL)
}

func TestParse_AcceptsHTTPAndNormalizesToHTTPS(t *testing.T) {
	link, err := gmaplink.Parse("http://www.google.com/maps/place/Bar/@43.4,-5.8,17z")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(link.URL, "https://"))
}
