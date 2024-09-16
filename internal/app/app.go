package app

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/suchimauz/golang-project-template/internal/config"
	"github.com/suchimauz/golang-project-template/pkg/logger"
)

func Run() {
	// Initialize config
	cfg, err := config.NewConfig()
	if err != nil {
		logger.Errorf("[ENV] %s", err.Error())

		return
	}

	spew.Dump(cfg)
}
