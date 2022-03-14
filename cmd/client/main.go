package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"net/url"
	"time"
)

var (
	pingAPIURL = flag.String("url", "https://localhost:9000/api/ping", "Ping API URL")
	interval   = flag.String("int", "100ms", "How fast to get updates from /ping")
	certPath   = flag.String("cert", "", "SSL cert path")
	keyPath    = flag.String("key", "", "SSL cert key path")
)

func main() {
	flag.Parse()

	if *certPath == "" || *keyPath == "" {
		log.Fatalf("-cert and -key flags are required")
	}

	u, err := url.Parse(*pingAPIURL)
	if err != nil {
		log.Fatal(err)
	}

	q := u.Query()
	q.Add("interval", *interval)
	u.RawQuery = q.Encode()

	h := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS12,
			},
		},
	}

	res, err := h.Get(u.String())
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(res.Body)
	defer res.Body.Close()

	t0 := time.Now()

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			log.Fatal(err)
		}

		var pong struct {
			Pong   bool   `json:"pong"`
			Reason string `json:"reason"`
			Date   string `json:"date"`
			Quote  string `json:"quote"`
		}

		// Check if the string we got is what we expected
		if err := json.Unmarshal(line, &pong); err != nil {
			diff := time.Since(t0)
			log.Printf("errored after %s\n", diff)
			log.Fatal(err)
		}

		j, err := json.Marshal(&pong)
		if err != nil {
			log.Fatal(err)
		}

		log.Println(string(j))
	}
}
