package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var (
	certPath  = flag.String("cert", "", "SSL cert path")
	keyPath   = flag.String("key", "", "SSL cert key path")
	serverURL = flag.String("server", "https://localhost:8000", "Upstream server URL")
)

func main() {
	flag.Parse()

	if *certPath == "" || *keyPath == "" {
		log.Fatalf("-cert and -key flags are required")
	}

	apiURL, err := url.Parse(*serverURL)
	if err != nil {
		log.Fatal(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(apiURL)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		},
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})

	s := &http.Server{
		Addr:    "0.0.0.0:9000",
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	log.Fatal(s.ListenAndServeTLS(*certPath, *keyPath))
}
