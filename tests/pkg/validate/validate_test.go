package validate_test

import (
	"math"
	"strings"
	"testing"

	"accesspath/pkg/validate"

	"github.com/stretchr/testify/assert"
)

func TestLat(t *testing.T) {
	tests := []struct {
		name    string
		in      float64
		wantErr bool
	}{
		{"valid equator", 0, false},
		{"valid north", 41.4, false},
		{"valid south", -33.8, false},
		{"valid boundary north", 90, false},
		{"valid boundary south", -90, false},
		{"over north", 90.1, true},
		{"under south", -90.1, true},
		{"way off", 200, true},
		{"NaN", math.NaN(), true},
		{"+Inf", math.Inf(1), true},
		{"-Inf", math.Inf(-1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Lat(tt.in)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLng(t *testing.T) {
	tests := []struct {
		name    string
		in      float64
		wantErr bool
	}{
		{"valid meridian", 0, false},
		{"valid east", 2.17, false},
		{"valid west", -122.4, false},
		{"valid boundary east", 180, false},
		{"valid boundary west", -180, false},
		{"over east", 180.1, true},
		{"under west", -180.1, true},
		{"way off", 500, true},
		{"NaN", math.NaN(), true},
		{"+Inf", math.Inf(1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Lng(tt.in)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNonZeroFloat(t *testing.T) {
	tests := []struct {
		name    string
		in      float64
		wantErr bool
	}{
		{"valid positive", 1.5, false},
		{"valid negative", -1.5, false},
		{"zero rejected", 0, true},
		{"NaN rejected", math.NaN(), true},
		{"+Inf rejected", math.Inf(1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.NonZeroFloat(tt.in, "field")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "field")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"valid", "user@example.com", false},
		{"valid lowercase", "user@example.com", false},
		{"with subdomain", "user@mail.example.com", false},
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"missing @", "userexample.com", true},
		{"missing user", "@example.com", true},
		{"missing domain", "user@", true},
		{"uppercase rejected", "User@Example.com", true},
		{"display name", "User <user@example.com>", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Email(tt.in)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMaxLen(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		max     int
		wantErr bool
	}{
		{"under limit", "hello", 10, false},
		{"at limit", "hello", 5, false},
		{"over limit", "hello world", 5, true},
		{"empty", "", 10, false},
		{"unicode counted as runes", "caf" + "é", 4, false},
		{"unicode over runes", "caf" + "é", 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.MaxLen(tt.in, "field", tt.max)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "field")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMinLen(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		min     int
		wantErr bool
	}{
		{"over limit", "hello world", 5, false},
		{"at limit", "hello", 5, false},
		{"under limit", "hi", 5, true},
		{"empty rejected", "", 1, true},
		{"zero min accepts anything", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.MinLen(tt.in, "field", tt.min)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNonEmpty(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"non-empty", "hello", false},
		{"whitespace only", strings.Repeat(" ", 5), true},
		{"empty", "", true},
		{"tabs and newlines", "\t\n", true},
		{"single char", "a", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.NonEmpty(tt.in, "field")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "field")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
