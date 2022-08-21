package zjcgo

import (
	"log"
	"net"
	"strings"
	"time"
)

type LoggerConfig struct {
}

func LoggerWithConfig(conf LoggerConfig, next HandlerFunc) HandlerFunc {

	return func(ctx *Context) {
		log.Println("log....")
		// Start timer
		start := time.Now()
		path := ctx.R.URL.Path
		raw := ctx.R.URL.RawQuery
		//执行业务
		next(ctx)
		// stop timer
		stop := time.Now()
		latency := stop.Sub(start)
		ip, _, _ := net.SplitHostPort(strings.TrimSpace(ctx.R.RemoteAddr))
		clientIP := net.ParseIP(ip)
		method := ctx.R.Method
		statusCode := ctx.StatusCode

		if raw != "" {
			path = path + "?" + raw
		}

		log.Printf("[msgo] %v | %3d | %13v | %15s |%-7s %#v",
			stop.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency, clientIP, method, path,
		)
	}
}

func Logging(next HandlerFunc) HandlerFunc {
	return LoggerWithConfig(LoggerConfig{}, next)
}
