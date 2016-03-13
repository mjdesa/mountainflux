package chasm

import (
	"bytes"
	"net"
	"sync"
	"sync/atomic"

	"github.com/valyala/fasthttp"
)

var (
	writePath     = []byte("/write")
	lineDelimiter = []byte("\n")
)

type Config struct {
	HTTPConfig *HTTPConfig
}

type HTTPConfig struct {
	// TCP address to listen to, e.g. `:8086` or `0.0.0.0:8086`
	Bind string
}

type Server struct {
	HTTPURL string

	httpListener         net.Listener
	httpRequestsAccepted uint64
	httpLinesAccepted    uint64
	httpBytesAccepted    uint64

	wg   sync.WaitGroup
	quit chan struct{}
}

// NewServer returns a new Server based on the supplied Config.
func NewServer(c Config) (*Server, error) {
	s := &Server{
		quit: make(chan struct{}),
	}

	if c.HTTPConfig != nil {
		var err error
		s.httpListener, err = net.Listen("tcp", c.HTTPConfig.Bind)
		if err != nil {
			return nil, err
		}
		s.HTTPURL = "http://" + s.httpListener.Addr().String()
	}

	return s, nil
}

// Server starts all the configured sub-servers in their own goroutines.
func (s *Server) Serve() {
	if s.httpListener != nil {
		s.wg.Add(1)
		go s.serveHTTP()
	}
}

// Close attempts to gracefully shutdown all the started sub-servers.
func (s *Server) Close() {
	close(s.quit)
	s.wg.Wait()
}

// HTTPRequestsAccepted returns the count of the number of requests accepted over HTTP.
func (s *Server) HTTPRequestsAccepted() uint64 {
	return atomic.LoadUint64(&s.httpRequestsAccepted)
}

// HTTPBytesAccepted returns the count of the number of bytes accepted over HTTP.
func (s *Server) HTTPBytesAccepted() uint64 {
	return atomic.LoadUint64(&s.httpBytesAccepted)
}

// HTTPLinesAccepted returns the count of the number of lines accepted over HTTP.
func (s *Server) HTTPLinesAccepted() uint64 {
	return atomic.LoadUint64(&s.httpLinesAccepted)
}

func (s *Server) serveHTTP() {
	// fasthttp.Server is intended to be opened forever.
	// The only obvious way to close one is to close its listener.
	// Since we want our Server to close gracefully, we'll handle the listener.

	fastServer := &fasthttp.Server{
		Handler: s.fasthttpHandler,
	}

	go func() {
		err := fastServer.Serve(s.httpListener)
		if err != nil {
			// TODO: Log the error? Restart the server?
			panic(err)
		}
	}()

	<-s.quit
	s.httpListener.Close()
	s.wg.Done()
}

func (s *Server) fasthttpHandler(ctx *fasthttp.RequestCtx) {
	atomic.AddUint64(&s.httpRequestsAccepted, 1)
	if !ctx.IsPost() || !bytes.Equal(ctx.Path(), writePath) {
		ctx.Response.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	body := ctx.PostBody()
	atomic.AddUint64(&s.httpBytesAccepted, uint64(len(body)))
	atomic.AddUint64(&s.httpLinesAccepted, uint64(bytes.Count(body, lineDelimiter)))
	ctx.Response.SetStatusCode(fasthttp.StatusNoContent)
}
