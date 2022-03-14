package main

import (
	"crypto/tls"
	"flag"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log "github.com/sirupsen/logrus"

	"github.com/transcelestial/echo-streaming-bug/pkg/datasource"
	"github.com/transcelestial/echo-streaming-bug/pkg/streaming"
)

var (
	certPath = flag.String("cert", "", "SSL cert path")
	keyPath  = flag.String("key", "", "SSL cert key path")
)

func main() {
	flag.Parse()

	e := echo.New()

	if *certPath == "" || *keyPath == "" {
		e.Logger.Fatalf("-cert and -key flags are required")
	}

	log.SetLevel(log.DebugLevel)

	e.Pre(middleware.HTTPSRedirect())
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Gzip())
	e.Use(middleware.Recover())

	streamer := streaming.NewHTTPStreamer()

	pongH := streamer.EchoHandler(datasource.Data)

	e.GET("/ping", pongH)

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	s := &http.Server{
		Addr:    "0.0.0.0:8000",
		Handler: e,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	e.Logger.Fatal(s.ListenAndServeTLS(*certPath, *keyPath))
}
