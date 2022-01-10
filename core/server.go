package core

import (
	"miner-pool/config"
)

type Server struct {
	cfg       *config.Config
	daemon    *Daemon
	postgres  *Postgres
	harvester *Harvester

	proxy *Proxy
}

func NewServer(cfg *config.Config) *Server {
	s := &Server{
		cfg:      cfg,

		daemon:   NewDaemon(cfg.Proxy.Daemon),
		postgres: NewPostgres(cfg.Postgres),
	}

	return s
}

func (s *Server) Start() {
	if *s.cfg.Proxy.Enabled {
		s.proxy = NewProxy(s)
	}
	if *s.cfg.Harvester.Enabled {
		s.harvester = NewHarvester(s)
	}
}

func (s *Server) Close() {
	s.proxy.Close()
	s.postgres.Close()
	s.harvester.Close()
}
