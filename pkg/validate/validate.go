// Package validate expone helpers de validacion de inputs que los handlers
// usan para devolver errores BadRequest concretos cuando el cliente envia
// datos malformados o fuera de rango. Los helpers devuelven strings de error
// en espanol (visibles al cliente via UserMessage de apperr.Validation).
package validate

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"
)

// Lat valida una latitud WGS84. Devuelve error si v esta fuera de [-90, 90]
// o si es NaN/Inf.
func Lat(v float64) error {
	if isBadFloat(v) {
		return errors.New("latitud invalida")
	}
	if v < -90 || v > 90 {
		return fmt.Errorf("latitud fuera de rango: %v", v)
	}
	return nil
}

// Lng valida una longitud WGS84. Devuelve error si v esta fuera de [-180, 180]
// o si es NaN/Inf.
func Lng(v float64) error {
	if isBadFloat(v) {
		return errors.New("longitud invalida")
	}
	if v < -180 || v > 180 {
		return fmt.Errorf("longitud fuera de rango: %v", v)
	}
	return nil
}

// NonZeroFloat rechaza el 0 explicito (algunos handlers usaban
// parseFloatOrDefault(..., 0) que enmascaraba inputs malformados).
func NonZeroFloat(v float64, field string) error {
	if isBadFloat(v) {
		return fmt.Errorf("%s invalido", field)
	}
	if v == 0 {
		return fmt.Errorf("%s no puede ser 0", field)
	}
	return nil
}

// Email delega en net/mail.ParseAddress. Devuelve el mismo string
// canonicalizado en minúsculas o un error.
func Email(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("email vacio")
	}
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return fmt.Errorf("email invalido: %s", err.Error())
	}
	if addr.Address != strings.ToLower(addr.Address) {
		return errors.New("email debe estar en minusculas")
	}
	return nil
}

// MaxLen verifica que s tenga <= max runes (no bytes). Devuelve error si lo
// excede. Strings vacios pasan (la regla de "requerido" va aparte).
func MaxLen(s, field string, max int) error {
	if utf8.RuneCountInString(s) > max {
		return fmt.Errorf("%s demasiado largo (max %d)", field, max)
	}
	return nil
}

// MinLen verifica que s tenga >= min runes.
func MinLen(s, field string, min int) error {
	if utf8.RuneCountInString(s) < min {
		return fmt.Errorf("%s demasiado corto (min %d)", field, min)
	}
	return nil
}

// NonEmpty rechaza strings vacios o solo whitespace.
func NonEmpty(s, field string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("%s no puede estar vacio", field)
	}
	return nil
}

func isBadFloat(v float64) bool {
	// NaN y +/-Inf no son comparables con rangos; tratarlos como invalidos.
	return v != v || v > 1e308 || v < -1e308
}
