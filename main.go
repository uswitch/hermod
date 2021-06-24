package main

import (
	"context"

	log "github.com/sirupsen/logrus"

	kubepkg "github.com/uswitch/hermod/pkg/kubernetes"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/sample-controller/pkg/signals"
)

type options struct {
	kubeconfig string
	logLevel   string
}

func main() {
	log.Info("starting controller")
	opts := &options{}
	kingpin.Flag("kubeconfig", "Path to kubeconfig.").StringVar(&opts.kubeconfig)
	kingpin.Flag("level", "Log level: debug, info, warn, error.").Default("info").EnumVar(&opts.logLevel, "debug", "info", "warn", "error")
	kingpin.Parse()

	configureLogger(opts.logLevel)

	kubeConfig, err := kubepkg.CreateClientConfig(opts.kubeconfig)
	if err != nil {
		log.Fatalf("error creating kube client config: %s", err)
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	ctx := context.Background()

	watcher := kubepkg.NewDeploymentWatcher(kubeClient)

	watcher.Context = ctx

	log.Info("starting deployment watcher")

	watcher.Run(ctx, stopCh)
}

func configureLogger(logLevel string) {
	switch logLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	}
	log.SetFormatter(&log.JSONFormatter{})
}
