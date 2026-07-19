package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	ginmiddleware "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"go.uber.org/zap"
)

func (m *Middleware) RateLimitMiddleware(rateString string) gin.HandlerFunc {
	rate, err := limiter.NewRateFromFormatted(rateString)
	if err != nil {
		m.logger.Fatal("Failed to parse rate limit", zap.Error(err))
	}

	store := memory.NewStore()
	instance := limiter.New(store, rate)

	middleware := ginmiddleware.NewMiddleware(instance)
	return middleware
}
