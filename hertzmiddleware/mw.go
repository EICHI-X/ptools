package hertzmiddleware

import (
	"context"
	"net/http"

	// "net/http"

	"time"

	"github.com/EICHI-X/ptools/pmodel"
	"github.com/EICHI-X/ptools/wtrace"
	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.opentelemetry.io/otel/trace"

	hertzZerolog "github.com/hertz-contrib/logger/zerolog"
)

// RequestIDHeaderValue value for the request id header
const RequestIDHeaderValue = "X-Request-ID"

// LoggerMiddleware middleware for logging incoming requests
func LoggerMiddleware() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		start := time.Now()

		logger := hlog.DefaultLogger().(*hertzZerolog.Logger)

		// reqIdo := c.Value(RequestIDHeaderValue)
		// reqId, ok := reqIdo.([16]byte)
		spanCtx := trace.SpanContextFromContext(c)
		if !spanCtx.IsValid() || !spanCtx.HasTraceID() {
			c, spanCtx = wtrace.InitSpanToContext(c, []byte{})
			// traceID,_ := wtrace.DefaultIDGenerator().NewIDs(c)
			// traceIdUid, _ := uuid.NewUUID()
			// reqId = traceIdUid.String()

			// c = context.WithValue(c, pmodel.TraceIDKey, spanCtx.TraceID())
		}

		logger.WithField(pmodel.TraceIdLogKey, spanCtx.TraceID())
		c = logger.WithContext(c)

		defer func() {
			stop := time.Now()

			logUnwrap := logger.Unwrap()
			logUnwrap.Info().
				Str("remote_ip", ctx.ClientIP()).
				Str("method", string(ctx.Method())).
				Str("path", string(ctx.Path())).
				Str("user_agent", string(ctx.UserAgent())).
				Int("status", ctx.Response.StatusCode()).
				Dur("latency", stop.Sub(start)).
				Str("latency_human", stop.Sub(start).String()).
				Msg("request processed")
		}()

		ctx.Next(c)
	}
}
func AddEnvMiddleware() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {

		requestId := ctx.GetHeader("X-Request-Id")

		spanCtx := trace.SpanContextFromContext(c)
		if !spanCtx.IsValid() || !spanCtx.HasTraceID() {
			c, spanCtx = wtrace.InitSpanToContext(c, requestId)
			ctx.Response.Header.Set("X-Request-Id", spanCtx.TraceID().String())
			// traceID,_ := wtrace.DefaultIDGenerator().NewIDs(c)
			// traceIdUid, _ := uuid.NewUUID()
			// reqId = traceIdUid.String()

			// c = context.WithValue(c, pmodel.TraceIDKey, spanCtx.TraceID())
		} else {
			ctx.Response.Header.Set("X-Request-Id", string(requestId))
		}
		c = PassHeaderToContext(c, ctx)
		// logs.CtxInfof(c, "requestId %v, %v",string(requestId))
		ctx.Next(c)
	}
}

func PassHeaderToContext(c context.Context, ctx *app.RequestContext) context.Context {

	for idx := range pmodel.PassHeaderKeys {
		k := pmodel.PassHeaderKeys[idx]
		v := ctx.Request.Header.Get(k)
		c = metainfo.WithPersistentValue(c, k, v)
	}

	// envType := ctx.Request.Header.Get("x-env-type")
	// env := ctx.Request.Header.Get("x-env")
	// c = metainfo.WithPersistentValue(c, "x-env-type", envType)
	// c = metainfo.WithPersistentValue(c, "x-env", env)

	return c
}

func PassContextHeaderToHertzClient() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		for idx := range pmodel.PassHeaderKeys {
			k := pmodel.PassHeaderKeys[idx]
			v, ok := metainfo.GetPersistentValue(c, k)
			if ok {
				ctx.Request.Header.Set(k, v)

			}

		}
		span := trace.SpanFromContext(c)
		if span.SpanContext().TraceID().IsValid() {
			ctx.Request.Header.Set(RequestIDHeaderValue, span.SpanContext().TraceID().String())
		}
		ctx.Next(c)

	}

}

// 设置 默认Cors 中间件
func CorsAllMiddleware() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {

		ctx.Response.Header.Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,UPDATE,PATCH")
		ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
		//服务器支持的所有跨域请求的方法
		//允许跨域设置可以返回其他子段，可以自定义字段
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "*")
		// 允许浏览器（客户端）可以解析的头部 （重要）
		ctx.Response.Header.Set("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers")
		//设置缓存时间
		ctx.Response.Header.Set("Access-Control-Max-Age", "172800")
		//允许客户端传递校验信息比如 cookie (重要)
		ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
		if ctx.Request.Header.IsOptions() {
			ctx.JSON(http.StatusNoContent, "")
			ctx.Abort()
		}
		// if ctx.Request.Header.IsOptions() {
		// 	var r struct {
		// 		Code int `json:"code"`
		// 	}
		// 	r.Code = 0
		// 	ctx.JSON(http.StatusOK, r)

		// }
		ctx.Next(c)
	}
}

func GetHertzHeader(ctx context.Context) http.Header {
	header := http.Header{}
	for idx := range pmodel.PassHeaderKeys {
		k := pmodel.PassHeaderKeys[idx]
		v, ok := metainfo.GetPersistentValue(ctx, k)
		if ok {
			header.Add(k, v)

		}

	}
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().TraceID().IsValid() {
		header.Add(RequestIDHeaderValue, span.SpanContext().TraceID().String())
	}
	return header
}
