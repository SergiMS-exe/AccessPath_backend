// Package apperr define el tipo AppError y helpers para construir, loguear y
// responder errores HTTP de forma consistente en todo el backend.
//
// El objetivo es que el handler NUNCA pierda el error original: cada fallo se
// loguea con contexto (op, code, http_status, request_id, causa) y llega al
// cliente con un mensaje user-friendly en el envelope { "error": "..." }.
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError es el error de aplicacion. Implementa `error` y se desenvuelve con
// errors.Is/As (Cause).
type AppError struct {
	// Op describe la operacion logica donde se origino el error, en notacion
	// "recurso.accion" (p.ej. "places.search", "gmaps.autocomplete").
	Op string

	// Code es el identificador estable y machine-readable (p.ej.
	// "gmaps.request_denied", "places.not_found"). NO cambia segun idioma.
	Code string

	// HTTPStatus es el codigo a devolver al cliente (400/401/404/429/500/502/503).
	HTTPStatus int

	// UserMessage es el texto que ve el usuario en el movil. En espanol,
	// pensado para mostrar literal en una SnackBar/Toast. Si esta vacio se
	// usa un fallback generico segun HTTPStatus.
	UserMessage string

	// Cause es el error original (p.ej. el error de la lib HTTP, de pgx, etc).
	// Se preserva para errores. Cause NO se serializa al response, solo al log.
	Cause error

	// Meta es metadata adicional para el log (NO se envia al cliente).
	// Evitar meter datos sensibles (tokens, api keys).
	Meta map[string]any
}

func (e *AppError) Error() string {
	if e == nil {
		return "<nil AppError>"
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %s (cause: %v)", e.Op, e.Code, e.UserMessage, e.Cause)
	}
	return fmt.Sprintf("%s: %s: %s", e.Op, e.Code, e.UserMessage)
}

func (e *AppError) Unwrap() error { return e.Cause }

// New construye un AppError generico.
func New(op, code string, status int, user string, cause error) *AppError {
	return &AppError{
		Op:          op,
		Code:        code,
		HTTPStatus:  status,
		UserMessage: user,
		Cause:       cause,
	}
}

// Wrap envuelve un error. Si ya es *AppError lo devuelve tal cual (para no
// perder code/http_status). Si no, lo convierte en Internal 500.
func Wrap(op string, err error) *AppError {
	if err == nil {
		return nil
	}
	var ae *AppError
	if errors.As(err, &ae) {
		return ae
	}
	return Internal(op, err)
}

// Internal error 500 generico. El UserMessage es generico: el cliente lo
// mostrara literalmente, por eso no expone detalles internos.
func Internal(op string, cause error) *AppError {
	return New(op, "internal_error", http.StatusInternalServerError,
		"Error interno del servidor. Intentalo de nuevo.", cause)
}

// Validation error 400.
func Validation(op, field, detail string) *AppError {
	return New(op, "validation_error", http.StatusBadRequest,
		fmt.Sprintf("Datos invalidos (%s): %s", field, detail), nil)
}

// BadRequest error 400 generico.
func BadRequest(op, code, user string) *AppError {
	return New(op, code, http.StatusBadRequest, user, nil)
}

// NotFound error 404.
func NotFound(op, what string) *AppError {
	return New(op, "not_found", http.StatusNotFound,
		fmt.Sprintf("%s no encontrado", what), nil)
}

// Unauthorized error 401.
func Unauthorized(op, user string) *AppError {
	return New(op, "unauthorized", http.StatusUnauthorized, user, nil)
}

// --- Dominio gmaps ---

// GmapsNotConfigured: API key vacia en el backend. Es un error de
// configuracion del servidor, no del usuario. 503.
func GmapsNotConfigured(op string) *AppError {
	return New(op, "gmaps.not_configured", http.StatusServiceUnavailable,
		"El buscador no esta disponible. Contacta con el administrador.", nil)
}

// GmapsRequestDenied: Google devolvio REQUEST_DENIED (key invalida, Places API
// no habilitada, sin facturacion). Es un problema de configuracion. 502.
func GmapsRequestDenied(op string, cause error) *AppError {
	return New(op, "gmaps.request_denied", http.StatusBadGateway,
		"El buscador de Google no esta disponible ahora mismo.", cause)
}

// GmapsQuotaExceeded: cuota mensual superada. 429.
func GmapsQuotaExceeded(op string) *AppError {
	return New(op, "gmaps.quota_exceeded", http.StatusTooManyRequests,
		"Has alcanzado el limite mensual de busquedas. Vuelve a intentarlo en unos dias.", nil)
}

// GmapsInvalidRequest: parametros invalidos hacia Google (input vacio, etc).
// 502 porque la peticion viene del usuario pero el problema es nuestro.
func GmapsInvalidRequest(op, detail string) *AppError {
	return New(op, "gmaps.invalid_request", http.StatusBadGateway,
		"Busqueda invalida.", fmt.Errorf("detail=%s", detail))
}

// GmapsUpstream: cualquier otro status de Google (UNKNOWN_ERROR, etc). 502.
func GmapsUpstream(op, gmapsStatus string, cause error) *AppError {
	meta := map[string]any{"gmaps_status": gmapsStatus}
	var c error
	if cause != nil {
		c = cause
	}
	ae := New(op, "gmaps.upstream_error", http.StatusBadGateway,
		"El buscador de Google no responde. Intentalo de nuevo.", c)
	ae.Meta = meta
	return ae
}

// GmapsNetwork: fallo de red/DNS/TLS al llamar a Google. 502.
func GmapsNetwork(op string, cause error) *AppError {
	return New(op, "gmaps.network_error", http.StatusBadGateway,
		"No se ha podido conectar con el buscador. Revisa tu conexion.", cause)
}

// --- Dominio places ---

// PlaceClosedPermanently: sitio cerrado segun Google. 400 (no tiene sentido
// importarlo).
func PlaceClosedPermanently() *AppError {
	return New("places.import", "places.closed_permanently", http.StatusBadRequest,
		"Este sitio esta cerrado permanentemente.", nil)
}

// PlaceSearchFailed: fallback cuando el search falla por algo no clasificado.
// Conserva la causa en el log.
func PlaceSearchFailed(cause error) *AppError {
	return New("places.search", "places.search_failed", http.StatusInternalServerError,
		"No se ha podido realizar la busqueda. Intentalo de nuevo.", cause)
}