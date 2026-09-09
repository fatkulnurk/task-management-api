package http

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	stdhttp "net/http"
	"runtime/debug"
	"taskmanagement/internal/application/authorization"
	"taskmanagement/internal/application/errorcode"
	"taskmanagement/internal/application/token"
	"taskmanagement/internal/platform/id"
	"time"
)

func Stack(tokenService token.TokenService, logger *slog.Logger) func(stdhttp.Handler) stdhttp.Handler {
	return func(nextHandler stdhttp.Handler) stdhttp.Handler {
		return withRequestID(logging(logger, recoverJSON(logger, nextHandler)))
	}
}

type requestIDKey struct{}

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey{}).(string)
	return value
}

func withRequestID(nextHandler stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(responseWriter stdhttp.ResponseWriter, request *stdhttp.Request) {
		requestID := request.Header.Get("X-Request-Id")
		if !id.IsValid(requestID) {
			requestID = id.New()
		}
		responseWriter.Header().Set("X-Request-Id", requestID)
		nextHandler.ServeHTTP(responseWriter, request.WithContext(context.WithValue(request.Context(), requestIDKey{}, requestID)))
	})
}

func recoverJSON(logger *slog.Logger, next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(responseWriter stdhttp.ResponseWriter, request *stdhttp.Request) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			logger.LogAttrs(request.Context(), slog.LevelError, "panic", slog.String("request_id", RequestID(request.Context())), slog.String("method", request.Method), slog.String("path", request.URL.Path), slog.Any("panic", recovered), slog.String("stack", string(debug.Stack())))
			JSONResponse5xx(responseWriter, stdhttp.StatusInternalServerError)
		}()
		next.ServeHTTP(responseWriter, request)
	})
}

func logging(logger *slog.Logger, nextHandler stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(responseWriter stdhttp.ResponseWriter, request *stdhttp.Request) {
		startTime := time.Now()
		wrappedResponseWriter := middleware.NewWrapResponseWriter(responseWriter, request.ProtoMajor)
		nextHandler.ServeHTTP(wrappedResponseWriter, request)
		logLevel := slog.LevelInfo
		if wrappedResponseWriter.Status() >= 500 {
			logLevel = slog.LevelError
		} else if wrappedResponseWriter.Status() >= 400 {
			logLevel = slog.LevelWarn
		}
		logger.LogAttrs(request.Context(), logLevel, "http", slog.String("request_id", RequestID(request.Context())), slog.String("method", request.Method), slog.String("path", request.URL.Path), slog.Int("status", wrappedResponseWriter.Status()), slog.Int64("latency", time.Since(startTime).Microseconds()))
	})
}

func Authenticate(tokenService token.TokenService) func(stdhttp.Handler) stdhttp.Handler {
	return func(nextHandler stdhttp.Handler) stdhttp.Handler {
		return stdhttp.HandlerFunc(func(responseWriter stdhttp.ResponseWriter, request *stdhttp.Request) {
			authorizationHeader := request.Header.Get("Authorization")
			if len(authorizationHeader) < 8 || authorizationHeader[:7] != "Bearer " {
				JSONResponse4xx(responseWriter, 401, errorcode.Unauthorized, "authentication required")
				return
			}
			userID, err := tokenService.Verify(request.Context(), authorizationHeader[7:])
			if err != nil {
				JSONResponse4xx(responseWriter, 401, errorcode.Unauthorized, "invalid access token")
				return
			}
			nextHandler.ServeHTTP(responseWriter, request.WithContext(authorization.WithIdentity(request.Context(), authorization.Identity{UserID: userID})))
		})
	}
}
