package streaming

import (
	"context"
	"encoding/json"
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
	EchoHandler(Params) echo.HandlerFunc
}

type httpStreamer struct{}

// EchoHandler returns an echo streaming response handler.
func (s *httpStreamer) EchoHandler(p Params) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, cancel := context.WithCancel(c.Request().Context())
		defer cancel()

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

		var d time.Duration
		params := requestParams{}
		if err := c.Bind(&params); err != nil {
			return badReqErr(err)
		} else if d, err = parseDuration(params.Interval); err != nil {
			return badReqErr(err)
		}

		res := c.Response()
		res.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		res.WriteHeader(http.StatusOK)

		jsonLogger := log.New()
		jsonLogger.SetLevel(log.GetLevel())
		lw := jsonLogger.WriterLevel(log.DebugLevel)
		defer lw.Close()

		w := io.MultiWriter(res, lw)
		enc := json.NewEncoder(w)

		ticker := time.NewTicker(d)
		defer ticker.Stop()

		// Send data before starting the ticker
		v, err := p.Read(c)
		if err != nil {
			return serverErr(err)
		}
		if err := enc.Encode(v); err != nil {
			return serverErr(err)
		}
		res.Flush()

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
					v, err := p.Read(c)
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
				return nil
			case err := <-e:
				return serverErr(err)
			case p := <-data:
				if err := enc.Encode(p); err != nil {
					log.Errorf("streaming: encode error: %s", err.Error())
					return serverErr(err)
				}
				res.Flush()
			}
		}
	}
}

type Params struct {
	// Read is the cb which provides the data to the stream and
	// it's invoked on every tick.
	Read ReadHandler
}

type ReadHandler func(c echo.Context) (interface{}, error)

type requestParams struct {
	Interval string `query:"interval"`
}

func parseDuration(s string) (d time.Duration, err error) {
	if s == "" {
		d = time.Second
		return
	}
	d, err = time.ParseDuration(s)
	return
}

func serverErr(err error) error {
	return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
}

func badReqErr(err error) error {
	return echo.NewHTTPError(http.StatusBadRequest, err.Error())
}
