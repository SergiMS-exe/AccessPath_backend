package apperr_test

import (
	"errors"
	"net/http"
	"testing"

	"accesspath/pkg/apperr"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	cause := errors.New("underlying")
	ae := apperr.New("places.create", "places.failed", 500, "no se pudo crear", cause)

	assert.Equal(t, "places.create", ae.Op)
	assert.Equal(t, "places.failed", ae.Code)
	assert.Equal(t, 500, ae.HTTPStatus)
	assert.Equal(t, "no se pudo crear", ae.UserMessage)
	assert.Same(t, cause, ae.Cause)
}

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name string
		ae   *apperr.AppError
		want string
	}{
		{
			name: "nil receiver",
			ae:   nil,
			want: "<nil AppError>",
		},
		{
			name: "without cause",
			ae:   apperr.New("x.y", "x.code", 400, "msg", nil),
			want: "x.y: x.code: msg",
		},
		{
			name: "with cause",
			ae:   apperr.New("x.y", "x.code", 500, "msg", errors.New("boom")),
			want: "x.y: x.code: msg (cause: boom)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.ae.Error())
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("boom")
	ae := apperr.New("x.y", "x.code", 500, "msg", cause)

	assert.Same(t, cause, ae.Unwrap())
	assert.True(t, errors.Is(ae, cause))
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantNil  bool
		wantCode string
		wantStat int
	}{
		{
			name:    "nil returns nil",
			err:     nil,
			wantNil: true,
		},
		{
			name:     "generic error wraps as Internal 500",
			err:      errors.New("db connection refused"),
			wantCode: "internal_error",
			wantStat: http.StatusInternalServerError,
		},
		{
			name: "AppError passes through unchanged",
			err:  apperr.New("x.y", "x.code", 409, "conflict", nil),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := apperr.Wrap("op.test", tt.err)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, got.Code)
				assert.Equal(t, tt.wantStat, got.HTTPStatus)
				return
			}
			orig := tt.err.(*apperr.AppError)
			assert.Same(t, orig, got)
		})
	}
}

func TestInternal(t *testing.T) {
	cause := errors.New("db down")
	ae := apperr.Internal("places.list", cause)

	assert.Equal(t, "places.list", ae.Op)
	assert.Equal(t, "internal_error", ae.Code)
	assert.Equal(t, http.StatusInternalServerError, ae.HTTPStatus)
	assert.NotEmpty(t, ae.UserMessage, "Internal debe dar mensaje generico al cliente")
	assert.Same(t, cause, ae.Cause)
}

func TestValidation(t *testing.T) {
	ae := apperr.Validation("places.create", "latitude", "out of range")

	assert.Equal(t, "validation_error", ae.Code)
	assert.Equal(t, http.StatusBadRequest, ae.HTTPStatus)
	assert.Contains(t, ae.UserMessage, "latitude")
	assert.Contains(t, ae.UserMessage, "out of range")
	assert.Nil(t, ae.Cause, "Validation no transporta cause (validacion es del cliente)")
}

func TestBadRequest(t *testing.T) {
	ae := apperr.BadRequest("x.create", "x.duplicate_email", "email duplicado")

	assert.Equal(t, "x.create", ae.Op)
	assert.Equal(t, "x.duplicate_email", ae.Code)
	assert.Equal(t, http.StatusBadRequest, ae.HTTPStatus)
	assert.Equal(t, "email duplicado", ae.UserMessage)
}

func TestNotFound(t *testing.T) {
	ae := apperr.NotFound("places.get", "Place")

	assert.Equal(t, http.StatusNotFound, ae.HTTPStatus)
	assert.Contains(t, ae.UserMessage, "Place")
	assert.Contains(t, ae.UserMessage, "no encontrado")
}

func TestUnauthorized(t *testing.T) {
	ae := apperr.Unauthorized("users.login", "credenciales invalidas")

	assert.Equal(t, http.StatusUnauthorized, ae.HTTPStatus)
	assert.Equal(t, "unauthorized", ae.Code)
	assert.Equal(t, "credenciales invalidas", ae.UserMessage)
}

func TestPlaceClosedPermanently(t *testing.T) {
	ae := apperr.PlaceClosedPermanently()

	assert.Equal(t, "places.closed_permanently", ae.Code)
	assert.Equal(t, http.StatusBadRequest, ae.HTTPStatus)
	assert.Contains(t, ae.UserMessage, "cerrado")
}
