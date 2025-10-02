package middleware

import (
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"

	"lister/internal/logger"

	"github.com/gin-gonic/gin"
)

func Recovery(logger *logger.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if ne, ok := recovered.(*net.OpError); ok {
			if se, ok := ne.Err.(*os.SyscallError); ok {
				if strings.Contains(strings.ToLower(se.Error()), "broken pipe") ||
					strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
					c.Abort()
					return
				}
			}
		}

		httpRequest, _ := httputil.DumpRequest(c.Request, false)
		if gin.IsDebugging() {
			logger.Error("[Recovery] panic recovered:\n%s\n%s\n%s", string(httpRequest), recovered, string(debug.Stack()))
		} else {
			logger.Error("[Recovery] panic recovered: %s", recovered)
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}
