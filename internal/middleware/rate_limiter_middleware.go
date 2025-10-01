package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func RateLimiter(max int, expiration time.Duration) fiber.Handler {
	if max == 0 {
		max = 50
	}
	if expiration == 0 {
		expiration = 1 * time.Minute
	}
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: expiration,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"code":    fiber.StatusTooManyRequests,
				"message": "Terlalu banyak permintaan",
			})
		},
		LimiterMiddleware: limiter.SlidingWindow{},
	})
}
