package logger

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/leigme/loki/app"
	"github.com/leigme/loki/file"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
)

func NewLoggerFile(workPath string) string {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	logPath := filepath.Join(workPath, "log")
	err := file.CreateDir(logPath)
	if err != nil {
		log.Fatalf("create log dir is err: %s\n", err)
	}
	logFile := fmt.Sprintf("%s.log", app.Name())
	return filepath.Join(logPath, logFile)
}

// GinLogger 接收gin框架默认的日志
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		p := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()

		cost := time.Since(start)
		log.Println(p,
			fmt.Sprint("status", c.Writer.Status()),
			fmt.Sprint("method", c.Request.Method),
			fmt.Sprint("path", p),
			fmt.Sprint("query", query),
			fmt.Sprint("ip", c.ClientIP()),
			fmt.Sprint("user-agent", c.Request.UserAgent()),
			fmt.Sprint("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			fmt.Sprint("cost", cost),
		)
	}
}

// GinRecovery recover掉项目可能出现的panic，并使用log记录相关日志
func GinRecovery(stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					log.Println(c.Request.URL.Path,
						fmt.Sprint("error", err),
						fmt.Sprint("request", string(httpRequest)),
					)
					// If the connection is dead, we can't write a status to it.
					c.Error(err.(error)) // nolint: err check
					c.Abort()
					return
				}

				if stack {
					log.Println("[Recovery from panic]",
						fmt.Sprint("error", err),
						fmt.Sprint("request", string(httpRequest)),
						fmt.Sprint("stack", string(debug.Stack())),
					)
				} else {
					log.Println("[Recovery from panic]",
						fmt.Sprint("error", err),
						fmt.Sprint("request", string(httpRequest)),
					)
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
