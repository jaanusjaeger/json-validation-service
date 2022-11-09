package server

import (
	"context"
	"net/http"
	"time"
)

type Conf struct {
	Addr string
}

type Server struct {
	server http.Server
}

func New(conf Conf, handlers map[string]http.HandlerFunc) *Server {
	mux := http.NewServeMux()

	for pattern, handler := range handlers {
		mux.HandleFunc(pattern, handler)
	}
	// TODO mux.HandleFunc("/", NotFound)

	return &Server{
		server: http.Server{
			Addr:    conf.Addr,
			Handler: mux,
		},
	}
}

func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.server.Shutdown(ctx)
}
