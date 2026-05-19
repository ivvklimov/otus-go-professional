package internalgrpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/logger"
)

// LoggingInterceptor логирует каждый Unary gRPC-запрос.
// Формат аналогичен HTTP middleware: IP [Timestamp] gRPC Method Status Durationms "-".
func LoggingInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Извлекаем IP клиента
		ip := "-"
		if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
			host, _, err := net.SplitHostPort(p.Addr.String())
			if err == nil && host != "" {
				ip = host
			}
		}

		// Выполняем обработчик
		resp, err := handler(ctx, req)

		// Определяем статус
		grpcCode := codes.OK
		if err != nil {
			grpcCode = status.Code(err)
		}
		httpStatus := grpcToHTTPStatus(grpcCode)

		// Длительность
		latency := time.Since(start).Milliseconds()

		// Формируем лог (аналог HTTP-формата)
		timestamp := start.Format("02/Jan/2006:15:04:05 -0700")
		errMsg := "-"
		if err != nil {
			errMsg = err.Error()
		}
		logLine := fmt.Sprintf("%s [%s] gRPC %s %d %dms \"%s\"",
			ip, timestamp, info.FullMethod, httpStatus, latency, errMsg)

		log.Info(logLine)

		return resp, err
	}
}

// grpcToHTTPStatus маппит gRPC-коды в HTTP-статусы для единообразия логов.
func grpcToHTTPStatus(code codes.Code) int {
	switch code {
	case codes.OK:
		return 200
	case codes.Canceled, codes.InvalidArgument, codes.NotFound, codes.AlreadyExists,
		codes.FailedPrecondition, codes.Aborted, codes.OutOfRange:
		return 400
	case codes.PermissionDenied, codes.Unauthenticated:
		return 403
	case codes.DeadlineExceeded:
		return 408
	case codes.Internal, codes.Unavailable, codes.DataLoss:
		return 500
	default:
		return 500
	}
}
