package core

import (
	"miner-pool/config"
)

type Server struct {
	cfg      *config.Config
	postgres *Postgres
	//redis    *Redis

	proxy *Proxy
}

func NewServer(cfg *config.Config) *Server {
	s := &Server{
		cfg:      cfg,
		postgres: NewPostgres(cfg.Postgres),

		//redis: NewRedis(cfg.Redis),
	}

	return s
}

func (s *Server) Start() {
	if *s.cfg.Proxy.Enabled {
		s.proxy = NewProxy(s)
	}
}

func (s *Server) Close() {
	s.proxy.Close()
	s.postgres.Close()
}
