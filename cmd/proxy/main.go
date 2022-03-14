package main

import (
	"crypto/tls"
	"flag"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	certPath  = flag.String("cert", "", "SSL cert path")
	keyPath   = flag.String("key", "", "SSL cert key path")
	serverURL = flag.String("server", "https://localhost:8000", "Upstream server URL")
)

func main() {
	flag.Parse()

	e := echo.New()

	if *certPath == "" || *keyPath == "" {
		e.Logger.Fatalf("-cert and -key flags are required")
	}

	e.Pre(middleware.HTTPSRedirect())
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Logger())
	// Do not use gzip compression on the API endpoints
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Skipper: func(c echo.Context) bool {
			return strings.HasPrefix(c.Path(), "/api")
		},
	}))
	e.Use(middleware.Recover())

	apiURL, err := url.Parse(*serverURL)
	if err != nil {
		e.Logger.Fatal(err)
	}
	targets := []*middleware.ProxyTarget{
		{
			URL: apiURL,
		},
	}

	apiRouter := e.Group("/api")
	apiRouter.Use(middleware.ProxyWithConfig(middleware.ProxyConfig{
		Balancer: middleware.NewRoundRobinBalancer(targets),
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS12,
			},
		},
		Rewrite: map[string]string{
			"/api/*": "/$1",
		},
	}))

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	s := &http.Server{
		Addr:    "0.0.0.0:9000",
		Handler: e,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	e.Logger.Fatal(s.ListenAndServeTLS(*certPath, *keyPath))
}
