package main

import (
	"crypto/tls"
	"flag"
	"net/http"

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

	if *certPath == "" || *keyPath == "" {
		log.Fatalf("-cert and -key flags are required")
	}

	log.SetLevel(log.DebugLevel)

	mux := http.NewServeMux()
	streamer := streaming.NewHTTPStreamer()

	pongH := streamer.HttpHandler(datasource.Data)

	mux.HandleFunc("/ping", pongH)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	s := &http.Server{
		Addr:    "0.0.0.0:8000",
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	log.Fatal(s.ListenAndServeTLS(*certPath, *keyPath))
}
