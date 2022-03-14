package streaming

import (
	"net/http"
)

// HttpHandler returns an HTTP streaming response handler.
func (s *httpStreamer) HttpHandler(read ReadCb) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if statusCode, err := s.handleHttp(read, w, r); err != nil {
			http.Error(w, err.Error(), statusCode)
		}
	}
}
