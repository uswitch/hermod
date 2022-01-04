package sentry

import (
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
)

func SetupSentry() {
	endpoint := os.Getenv("SENTRY_ENDPOINT")
	err := sentry.Init(sentry.ClientOptions{
		Dsn: endpoint,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
}

func Cleanup() {
	log.Info("running cleanup")
	sentry.Flush(2 * time.Second)
}
