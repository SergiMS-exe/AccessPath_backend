package middleware

import (
	"time"

	"accesspath/pkg/apperr"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// KeyFunc deriva el identificador de rate limit para un request. Lo habitual
// es usar la IP del cliente (limita por origen) o el user_id del token (limita
// por usuario autenticado).
type KeyFunc func(c *gin.Context) string

// RateLimit aplica un limite fijo de `limit` requests por ventana `window`.
// Backend: Redis INCR + EXPIRE. Si Redis es nil, falla la conexion o el
// comando, el middleware hace fail-open (deja pasar) para no degradar el
// servicio si Redis cae. Esto sigue la misma politica de degradacion que el
// middleware cache (ver internal/middleware/cache.go).
func RateLimit(rdb *redis.Client, key KeyFunc, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil || key == nil {
			c.Next()
			return
		}

		k := "rl:" + key(c)
		ctx := c.Request.Context()

		count, err := rdb.Incr(ctx, k).Result()
		if err != nil {
			// Fail-open: si Redis falla, dejamos pasar para no romper el servicio.
			c.Next()
			return
		}
		if count == 1 {
			// Primera vez que vemos la key en esta ventana, fijamos el TTL.
			// Ignoramos error a proposito: si el EXPIRE falla, la key persistira,
			// pero la siguiente vez que el backend reinicie Redis ya estara limpia.
			_ = rdb.Expire(ctx, k, window).Err()
		}

		if count > int64(limit) {
			ae := apperr.TooManyRequests(opFromFullPath(c))
			ae.UserMessage = "Demasiadas solicitudes. Vuelve a intentarlo en unos minutos."
			// Headers estandar para que clientes inteligentes sepan el cooldown.
			c.Header("Retry-After", RetryAfterSeconds(window))
			apperr.RespondInternal(c, ae)
			c.Abort()
			return
		}

		c.Next()
	}
}

// KeyByIP usa c.ClientIP() como identificador. Util cuando el endpoint es
// publico (login, register) y queremos limitar por origen.
func KeyByIP(c *gin.Context) string {
	return "ip:" + c.ClientIP()
}

// KeyByUserID usa el user_id del contexto (puesto por el middleware Auth).
// Si no hay user_id (rutas publicas), cae a la IP. Util para endpoints
// autenticados donde queremos limite por usuario, no por IP.
func KeyByUserID(c *gin.Context) string {
	uid, ok := UserID(c)
	if ok {
		return "u:" + ItoaForLimit(uid)
	}
	return "ip:" + c.ClientIP()
}

func opFromFullPath(c *gin.Context) string {
	if p := c.FullPath(); p != "" {
		return p
	}
	return "rate_limit"
}

// RetryAfterSeconds devuelve el cooldown en segundos como string para la
// cabecera HTTP Retry-After. Si window es < 1s, devuelve "1" (minimo razonable).
// Exportada para tests dedicados en tests/.
func RetryAfterSeconds(window time.Duration) string {
	secs := int(window.Seconds())
	if secs < 1 {
		secs = 1
	}
	return ItoaForLimit(int64(secs))
}

// ItoaForLimit es un strconv.Itoa acotado al rango de int64. Vive separado
// del strconv estandar para que el formato concreto este bajo control de
// tests. Exportada por la misma razon que RetryAfterSeconds.
func ItoaForLimit(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
