package main

import (
	"github.com/gin-gonic/gin"
	"github.com/haapjari/glass-api/api"
	"github.com/haapjari/glass-api/internal/pkg/cfg"
	"github.com/haapjari/glass-api/internal/pkg/logger"
	"github.com/haapjari/glass-api/internal/pkg/srv"
)

func main() {
	c, err := cfg.NewConfig()
	if err != nil {
		panic(err)
	}

	gin.SetMode(c.GinMode)

	router := gin.Default()

	log := logger.NewLogger(logger.DebugLevel)

	api.RegisterHandlers(router, srv.NewServer(log, c))

	if err = router.Run("127.0.0.1" + ":" + c.Port); err != nil {
		log.Errorf("unable to run the server on port: %v, error: %v", c.Port, err.Error())
		return
	}
}
