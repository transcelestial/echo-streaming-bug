package main

import (
	"crypto/tls"
	"flag"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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

	e.Pre(middleware.HTTPSRedirect())
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Gzip())
	e.Use(middleware.Recover())

	streamer := streaming.NewHTTPStreamer()

	pongH := streamer.EchoHandler(streaming.Params{
		Read: func(c echo.Context) (interface{}, error) {
			return struct {
				Pong   bool   `json:"pong"`
				Reason string `json:"reason"`
				Date   string `json:"date"`
				Quote  string `json:"quote"`
			}{
				true,
				"the bug does not seem to be here :( where is it then?",
				"2022-03-12T11:34:26.722022402Z",
				"dumbass!",
			}, nil
		},
	})

	e.GET("/ping", pongH)

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	s := &http.Server{
		Addr:    ":8000",
		Handler: e,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	e.Logger.Fatal(s.ListenAndServeTLS(*certPath, *keyPath))
}
