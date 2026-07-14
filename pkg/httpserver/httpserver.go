// Package httpserver bootstraps a Gin HTTP server with graceful shutdown
// and the standard /health, /ready, /metrics endpoints every service must
// expose.
package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// ReadyCheckFunc reports whether the service is ready to accept traffic
// (e.g. downstream dependencies are reachable).
type ReadyCheckFunc func(ctx context.Context) error

// RegisterProbes wires /health (liveness — always OK if the process is
// running), /ready (readiness — delegates to check), and /metrics
// (Prometheus) onto router.
func RegisterProbes(router *gin.Engine, check ReadyCheckFunc) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/ready", func(c *gin.Context) {
		if check == nil {
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		if err := check(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

// Server wraps http.Server with graceful shutdown semantics driven by
// context cancellation (typically tied to SIGTERM/SIGINT in cmd/main.go).
type Server struct {
	httpServer      *http.Server
	logger          *zap.Logger
	shutdownTimeout time.Duration
}

func New(port int, handler http.Handler, logger *zap.Logger, shutdownTimeout time.Duration) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		},
		logger:          logger,
		shutdownTimeout: shutdownTimeout,
	}
}

// Run starts the server and blocks until ctx is cancelled, at which point
// it drains in-flight requests (bounded by shutdownTimeout) before
// returning.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("http_server_starting", zap.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	s.logger.Info("http_server_shutting_down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	return s.httpServer.Shutdown(shutdownCtx)
}
