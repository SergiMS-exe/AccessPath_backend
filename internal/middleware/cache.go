package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func Cache(client *redis.Client, prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if client == nil {
			c.Next()
			return
		}

		key := prefix + ":" + hashKey(c.Request.URL.String())

		cached, err := client.Get(c.Request.Context(), key).Result()
		if err == nil {
			var data map[string]any
			if json.Unmarshal([]byte(cached), &data) == nil {
				c.JSON(200, data)
				c.Abort()
				return
			}
		}

		c.Next()

		if c.Writer.Status() == 200 {
			// Se podría implementar un response writer personalizado para cachear
			// Por ahora solo pasa
		}
	}
}

func hashKey(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}

func SetCache(client *redis.Client, key string, data any, ttl time.Duration) error {
	if client == nil {
		return nil
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return client.Set(nil, key, bytes, ttl).Err()
}
