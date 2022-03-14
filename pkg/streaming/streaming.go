package streaming

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
)

func NewHTTPStreamer() HTTPStreamer {
	return &httpStreamer{}
}

type HTTPStreamer interface {
	EchoHandler(ReadCb) echo.HandlerFunc
	HttpHandler(ReadCb) http.HandlerFunc
}

type ReadCb func() (interface{}, error)

type httpStreamer struct{}

func (*httpStreamer) handleHttp(read ReadCb, w http.ResponseWriter, r *http.Request) (int, error) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	flusher, ok := w.(http.Flusher)
	if !ok {
		err := errors.New("streaming: w does not implement the http.Flusher interface")
		log.Error(err)
		return http.StatusInternalServerError, err
	}

	data := make(chan interface{})
	e := make(chan error)

	// Drain the chans (not great, but ok)
	defer func() {
		log.Debug("streaming: draining chans")
		select {
		case <-data:
		default:
		}
		select {
		case <-e:
		default:
		}
	}()

	log.Debug("streaming: req headers", r.Header)

	d, err := parseDuration(r.URL.Query().Get("interval"))
	if err != nil {
		return http.StatusBadRequest, err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	jsonLogger := log.New()
	jsonLogger.SetLevel(log.GetLevel())
	lw := jsonLogger.WriterLevel(log.DebugLevel)
	defer lw.Close()

	mw := io.MultiWriter(w, lw)
	enc := json.NewEncoder(mw)

	ticker := time.NewTicker(d)
	defer ticker.Stop()

	// Send data before starting the ticker
	v, err := read()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if err := enc.Encode(v); err != nil {
		return http.StatusInternalServerError, err
	}
	flusher.Flush()

	go func(data chan<- interface{}, e chan<- error) {
		// Sender must close the chans
		defer func() {
			log.Debug("streaming: close chans")
			close(data)
			close(e)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				v, err := read()
				if err != nil {
					log.Errorf("streaming: read error: %s", err.Error())
					e <- err
					return
				}
				log.Tracef("streaming: send data: %v", v)
				data <- v
			}
		}
	}(data, e)

	for {
		select {
		case <-ctx.Done():
			log.Debug("streaming: request cancelled")
			return http.StatusOK, nil
		case err := <-e:
			return http.StatusInternalServerError, err
		case p := <-data:
			if err := enc.Encode(p); err != nil {
				log.Errorf("streaming: encode error: %s", err.Error())
				return http.StatusInternalServerError, err
			}
			flusher.Flush()
		}
	}
}

func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return time.Second, nil
	}
	return time.ParseDuration(s)
}
