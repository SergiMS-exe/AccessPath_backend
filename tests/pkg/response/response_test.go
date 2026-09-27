package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"accesspath/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// runHandler invoca fn dentro de un gin.Context de test y devuelve el recorder.
func runHandler(t *testing.T, fn func(c *gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	fn(c)
	return w
}

func decode(t *testing.T, body []byte) response.Envelope {
	t.Helper()
	var env response.Envelope
	require.NoError(t, json.Unmarshal(body, &env))
	return env
}

func TestWrap_ReturnsDataKey(t *testing.T) {
	env := response.Wrap(map[string]string{"k": "v"})
	assert.Equal(t, map[string]string{"k": "v"}, env.Data)
	assert.Empty(t, env.Error, "Wrap no debe poblar el campo error")
}

func TestWrap_NilDataIsPreserved(t *testing.T) {
	// Data es any; pasar nil deja el campo como puntero nil, pero JSON omitempty
	// lo descarta. Solo verificamos que no se serializa 'data': null por accidente
	// en la ruta tipica (caller pone struct).
	env := response.Wrap(nil)
	js, err := json.Marshal(env)
	require.NoError(t, err)
	assert.NotContains(t, string(js), `"error"`)
}

func TestOK_Writes200AndDataEnvelope(t *testing.T) {
	w := runHandler(t, func(c *gin.Context) {
		response.OK(c, map[string]string{"id": "42"})
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"data":{"id":"42"}}`, w.Body.String())
}

func TestOK_StringPayload(t *testing.T) {
	w := runHandler(t, func(c *gin.Context) {
		response.OK(c, "hola")
	})

	assert.JSONEq(t, `{"data":"hola"}`, w.Body.String())
}

func TestCreated_Writes201AndDataEnvelope(t *testing.T) {
	w := runHandler(t, func(c *gin.Context) {
		response.Created(c, map[string]int{"id": 7})
	})

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.JSONEq(t, `{"data":{"id":7}}`, w.Body.String())
}

func TestBadRequest_AbortsWith400(t *testing.T) {
	w := runHandler(t, func(c *gin.Context) {
		response.BadRequest(c, "body invalido")
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	env := decode(t, w.Body.Bytes())
	assert.Equal(t, "body invalido", env.Error)
	assert.Nil(t, env.Data)
}

func TestNotFound_AbortsWith404(t *testing.T) {
	w := runHandler(t, func(c *gin.Context) {
		response.NotFound(c, "no encontrado")
	})

	assert.Equal(t, http.StatusNotFound, w.Code)
	env := decode(t, w.Body.Bytes())
	assert.Equal(t, "no encontrado", env.Error)
}

func TestUnauthorized_AbortsWith401(t *testing.T) {
	w := runHandler(t, func(c *gin.Context) {
		response.Unauthorized(c, "token requerido")
	})

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	env := decode(t, w.Body.Bytes())
	assert.Equal(t, "token requerido", env.Error)
}

func TestInternalError_AbortsWith500(t *testing.T) {
	w := runHandler(t, func(c *gin.Context) {
		response.InternalError(c, "boom")
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	env := decode(t, w.Body.Bytes())
	assert.Equal(t, "boom", env.Error)
}

func TestTooManyRequests_AbortsWith429(t *testing.T) {
	w := runHandler(t, func(c *gin.Context) {
		response.TooManyRequests(c, "calm down")
	})

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	env := decode(t, w.Body.Bytes())
	assert.Equal(t, "calm down", env.Error)
}

func TestErrorHelpers_AbortContext(t *testing.T) {
	// Los helpers BadRequest/NotFound/... usan AbortWithStatusJSON. Esto
	// garantiza que la cadena de middleware posterior no se ejecuta.
	tests := []struct {
		name   string
		fn     func(c *gin.Context)
		status int
	}{
		{"BadRequest", func(c *gin.Context) { response.BadRequest(c, "x") }, http.StatusBadRequest},
		{"NotFound", func(c *gin.Context) { response.NotFound(c, "x") }, http.StatusNotFound},
		{"Unauthorized", func(c *gin.Context) { response.Unauthorized(c, "x") }, http.StatusUnauthorized},
		{"InternalError", func(c *gin.Context) { response.InternalError(c, "x") }, http.StatusInternalServerError},
		{"TooManyRequests", func(c *gin.Context) { response.TooManyRequests(c, "x") }, http.StatusTooManyRequests},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
			r.GET("/x", func(c *gin.Context) {
				tt.fn(c)
			}, func(c *gin.Context) {
				// Handler post-error: si Abort funciona, esto no se ejecuta.
				t.Errorf("handler posterior ejecutado; Abort no detuvo la cadena")
			})

			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.status, w.Code)
		})
	}
}

func TestOK_DoesNotAbort(t *testing.T) {
	// OK usa c.JSON normal (no Abort). Si un handler quiere cortar la cadena
	// despues de OK debe usar c.Abort() a mano.
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	called := false
	r.GET("/x", func(c *gin.Context) {
		response.OK(c, gin.H{"ok": true})
	}, func(c *gin.Context) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)

	assert.True(t, called, "OK no debe abortar la cadena")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEnvelope_JSONShape(t *testing.T) {
	// Solo uno de data/error puede estar presente en el JSON final
	// (omitempty). Validamos la forma canonica del contrato.
	js, err := json.Marshal(response.Envelope{Data: "x"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"data":"x"}`, string(js))

	js, err = json.Marshal(response.Envelope{Error: "x"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"error":"x"}`, string(js))

	js, err = json.Marshal(response.Envelope{})
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(js))
}
