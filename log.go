package prox

import "go.uber.org/zap"

func logger(name string) *zap.Logger {
	conf := zap.NewDevelopmentConfig()
	logger, err := conf.Build()
	if err != nil {
		panic(err)
	}

	if name == "" {
		return logger
	}

	return logger.Named(name)
}
