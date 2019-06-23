package main

import (
	"context"
	log "github.com/sirupsen/logrus"
)

func TaskMain(ctx context.Context) error {
	log.Debugf("run main")
	return nil
}

func TaskErr(ctx context.Context, err error) error {
	log.Debugf("run main")
	return nil
}
