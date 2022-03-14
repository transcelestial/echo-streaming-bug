package streaming

import (
	"github.com/labstack/echo/v4"
)

// EchoHandler returns an echo streaming response handler.
func (s *httpStreamer) EchoHandler(read ReadCb) echo.HandlerFunc {
	return func(c echo.Context) error {
		res := c.Response()
		if statusCode, err := s.handleHttp(read, res, c.Request()); err != nil {
			return echo.NewHTTPError(statusCode, err.Error())
		}
		return nil
	}
}
