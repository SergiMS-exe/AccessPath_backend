package middleware

import (
	"strings"

	"accesspath/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "token requerido")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "formato de token inválido")
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Exigir HS256: rechazar cualquier otro algoritmo (evita el ataque de
			// confusion de algoritmo, p.ej. tokens firmados con "none" o RS/ES).
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != "HS256" {
				return nil, jwt.ErrTokenSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			response.Unauthorized(c, "token inválido")
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if tokenType, _ := claims["type"].(string); tokenType == "refresh" {
				response.Unauthorized(c, "token invalido")
				return
			}
			c.Set("user_id", claims["user_id"])
		}

		c.Next()
	}
}

// UserID extrae el user_id (int64) del contexto que fijo el middleware Auth.
// Los claims JWT llegan como float64 en MapClaims; esta funcion lo normaliza.
func UserID(c *gin.Context) (int64, bool) {
	raw, ok := c.Get("user_id")
	if !ok {
		return 0, false
	}
	switch v := raw.(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	}
	return 0, false
}
