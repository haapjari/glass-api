package srv

import (
	"github.com/haapjari/glass-api/internal/pkg/cfg"
	"github.com/haapjari/glass-api/internal/pkg/logger"
)

// Server implements
type Server struct {
	log    logger.Logger
	config *cfg.Config
}

func NewServer(log logger.Logger, conf *cfg.Config) *Server {
	return &Server{
		log:    log,
		config: conf,
	}
}
