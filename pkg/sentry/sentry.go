package sentry

import (
	"log"
	"os"

	"github.com/getsentry/sentry-go"
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
