// Package avalanche creates a massive amount of writes against your target InfluxDB instance.
package avalanche

import (
	"fmt"
	"net/url"

	"github.com/valyala/fasthttp"
)

// HTTPWriterConfig is the configuration used to create an HTTPWriter.
type HTTPWriterConfig struct {
	// URL of the host, in form "http://example.com:8086"
	Host string

	// Name of the target database into which points will be written.
	Database string
}

// HTTPWriter is a Writer that writes to an InfluxDB HTTP server.
type HTTPWriter struct {
	client fasthttp.Client

	c   HTTPWriterConfig
	url []byte
}

// NewHTTPWriter returns a new HTTPWriter from the supplied HTTPWriterConfig.
func NewHTTPWriter(c HTTPWriterConfig) LineProtocolWriter {
	return &HTTPWriter{
		client: fasthttp.Client{
			Name: "avalanche",
		},

		c:   c,
		url: []byte(c.Host + "/write?db=" + url.QueryEscape(c.Database)),
	}
}

var post = []byte("POST")

// WriteLineProtocol writes the given byte slice to the HTTP server described in the Writer's HTTPWriterConfig.
func (w *HTTPWriter) WriteLineProtocol(body []byte) error {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethodBytes(post)
	req.Header.SetRequestURIBytes(w.url)
	req.SetBody(body)

	resp := fasthttp.AcquireResponse()
	err := w.client.Do(req, resp)
	if err == nil {
		sc := resp.StatusCode()
		if sc != fasthttp.StatusNoContent {
			err = fmt.Errorf("Invalid write response (status %d): %s", sc, resp.Body())
		}
	}

	fasthttp.ReleaseResponse(resp)
	fasthttp.ReleaseRequest(req)

	return err
}
