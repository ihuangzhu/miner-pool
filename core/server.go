package core

import (
	"miner-pool/config"
)

type Server struct {
<<<<<<< Updated upstream
	cfg      *config.Config
	postgres *Postgres
	//redis    *Redis
=======
	cfg       *config.Config
	daemon    *Daemon
	postgres  *Postgres
	harvester *Harvester
>>>>>>> Stashed changes

	proxy *Proxy
}

func NewServer(cfg *config.Config) *Server {
	s := &Server{
		cfg:      cfg,
<<<<<<< Updated upstream
		postgres: NewPostgres(cfg.Postgres),

		//redis: NewRedis(cfg.Redis),
=======
		daemon:   NewDaemon(cfg.Proxy.Daemon),
		postgres: NewPostgres(cfg.Postgres),
>>>>>>> Stashed changes
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
<<<<<<< Updated upstream
=======
	s.harvester.Close()
>>>>>>> Stashed changes
}
