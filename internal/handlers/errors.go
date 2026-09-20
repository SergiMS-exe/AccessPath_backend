package handlers

import (
	"errors"

	"accesspath/internal/services"
	"accesspath/pkg/apperr"

	"github.com/gin-gonic/gin"
)

// serviceErrorMapping asocia sentinels (o tipos) de services con el AppError
// que hay que construir cuando el handler los recibe. Mantener esto como un
// slice + loop evita un switch y permite extenderlo solo anadiendo entradas.
type serviceErrorMapping struct {
	target  error                       // sentinel o *T para errors.As (documental)
	matcher func(err error) bool        // error.Is/As concreto
	build   func(op string, err error) *apperr.AppError
}

// serviceErrorMappings lista los errores que reconocemos. El orden importa
// porque se itera secuencialmente: el primero que coincida gana.
var serviceErrorMappings = []serviceErrorMapping{
	{
		target: services.ErrInvalidCredentials,
		matcher: func(t error) bool {
			return errors.Is(t, services.ErrInvalidCredentials)
		},
		build: func(op string, _ error) *apperr.AppError {
			return apperr.Unauthorized(op, "credenciales invalidas")
		},
	},
	{
		target: services.ErrEmailAlreadyUsed,
		matcher: func(t error) bool {
			return errors.Is(t, services.ErrEmailAlreadyUsed)
		},
		build: func(op string, _ error) *apperr.AppError {
			return apperr.BadRequest(op, "users.email_taken", "el email ya esta registrado")
		},
	},
	{
		target: services.ErrOptionMismatch,
		matcher: func(t error) bool {
			return errors.Is(t, services.ErrOptionMismatch)
		},
		build: func(op string, _ error) *apperr.AppError {
			return apperr.BadRequest(op, "contributions.option_mismatch",
				"answer option does not belong to criterion")
		},
	},
	{
		target: services.ErrContributionNotFound,
		matcher: func(t error) bool {
			return errors.Is(t, services.ErrContributionNotFound)
		},
		build: func(op string, _ error) *apperr.AppError {
			return apperr.NotFound(op, "Contribution")
		},
	},
	{
		target: services.ErrNotOwner,
		matcher: func(t error) bool {
			return errors.Is(t, services.ErrNotOwner)
		},
		build: func(op string, _ error) *apperr.AppError {
			return apperr.Unauthorized(op, "no autorizado")
		},
	},
	{
		target: services.ErrEmptySubmission,
		matcher: func(t error) bool {
			return errors.Is(t, services.ErrEmptySubmission)
		},
		build: func(op string, _ error) *apperr.AppError {
			return apperr.BadRequest(op, "submissions.empty",
				"se requiere un comentario o al menos una foto")
		},
	},
	{
		target: services.ErrGmapsQuotaExceeded,
		matcher: func(t error) bool {
			return errors.Is(t, services.ErrGmapsQuotaExceeded)
		},
		build: func(op string, _ error) *apperr.AppError {
			return apperr.GmapsQuotaExceeded(op)
		},
	},
	{
		target: services.ErrPlaceClosedPermanently,
		matcher: func(t error) bool {
			return errors.Is(t, services.ErrPlaceClosedPermanently)
		},
		build: func(op string, _ error) *apperr.AppError {
			return apperr.PlaceClosedPermanently()
		},
	},
	{
		// Tipo (no sentinel): ErrInvalidNeedKey se compara con errors.As.
		target: (*services.ErrInvalidNeedKey)(nil),
		matcher: func(t error) bool {
			var invalid services.ErrInvalidNeedKey
			return errors.As(t, &invalid)
		},
		build: func(op string, err error) *apperr.AppError {
			var invalid services.ErrInvalidNeedKey
			errors.As(err, &invalid)
			return apperr.BadRequest(op, "profile.invalid_need", invalid.Error())
		},
	},
}

// matchServiceError recorre el registry buscando el primer match.
func matchServiceError(err error) (func(op string, err error) *apperr.AppError, bool) {
	for _, m := range serviceErrorMappings {
		if m.matcher(err) {
			return m.build, true
		}
	}
	return nil, false
}

// Respond es el punto de entrada unico desde los handlers. Clasifica `err`:
//
//   1. Si es nil, no hace nada y devuelve false.
//   2. Si matchea un sentinel de services, construye el *AppError
//      correspondiente.
//   3. Si no, lo trata como Internal 500 (conservando la causa).
//
// En todos los casos registra un log estructurado y escribe el envelope de
// respuesta. Devuelve true si respondio (utile para `if Respond(c,err){return}`).
func Respond(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	// Si ya es *AppError, pasa tal cual (preserva code/http_status ya tipados).
	var ae *apperr.AppError
	if errors.As(err, &ae) {
		apperr.RespondInternal(c, ae)
		return true
	}

	if build, ok := matchServiceError(err); ok {
		ae = build(opFromRequest(c), err)
	} else {
		ae = apperr.Internal(opFromRequest(c), err)
	}
	apperr.RespondInternal(c, ae)
	return true
}

// opFromRequest devuelve un op estable para agrupar logs cuando el error no
// trae uno. Usa el path de la ruta (no la URL con query), p.ej.
// "places.search".
func opFromRequest(c *gin.Context) string {
	if p := c.FullPath(); p != "" {
		return p
	}
	return "http"
}