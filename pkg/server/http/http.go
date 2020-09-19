package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.elastic.co/apm/module/apmgin"
	gintrace "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	server *http.Server
}

func New(log *zap.Logger, devMode bool, server *http.Server) *Server {
	if !devMode {
		gin.SetMode(gin.ReleaseMode)
	}
	// Instantiate gin router.
	r := gin.New()
	r.Use(apmgin.Middleware(r))
	r.Use(ginLogger(log))
	r.Use(gintrace.Middleware("cloudnative"))

	// Setup probes handlers.
	r.GET("/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ready",
		})
	})

	r.GET("/alive", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "alive",
		})
	})

	if server == nil {
		server = &http.Server{Addr: ":8080"}
	}
	server.Handler = r

	return &Server{
		server: server,
	}
}

// Gin logger middleware.
func ginLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()
		if len(c.Errors) > 0 {
			for _, err := range c.Errors.Errors() {
				log.Error(err)
			}
			return
		}
		log.Info(path,
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("userAgent", c.Request.UserAgent()),
			zap.Int("bodySize", c.Writer.Size()),
			zap.Duration("latency", time.Since(start)),
		)
	}
}

func (srv *Server) Gin() *gin.Engine {
	return srv.server.Handler.(*gin.Engine)
}

func (srv *Server) Start(ctx context.Context, log *zap.Logger, errs chan<- error) error {
	// Add context to the logger.
	log = log.Named("http")

	// Any error is semantically wrapped.
	wrapError := func(err error) error {
		return fmt.Errorf("http server: %w", err)
	}

	// Start the HTTP server.
	go func() {
		log.Info("start listening on :8080")
		if err := srv.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error(err.Error())
			errs <- wrapError(err)
			return
		}
		errs <- nil
	}()

	// Wait for the context cancellation to shutdown the server.
	go func() {
		<-ctx.Done()
		if err := srv.server.Shutdown(context.Background()); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				errs <- nil
				return
			}
			log.Error(err.Error())
			errs <- wrapError(err)
		} else {
			log.Info("shutdown")
			errs <- nil
		}
	}()

	return nil
}
