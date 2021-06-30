package main

import (
	"context"

	log "github.com/sirupsen/logrus"

	kubepkg "github.com/uswitch/hermod/pkg/kubernetes"
	"github.com/uswitch/hermod/pkg/slack"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/sample-controller/pkg/signals"
)

type options struct {
	kubeconfig string
	logLevel   string
	channelID  string
}

func main() {
	log.Info("starting controller")
	opts := &options{}
	kingpin.Flag("channel-id", "ID of slack Channel to send messages too").StringVar(&opts.channelID)
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

	slackClient, err := slack.NewClient(opts.channelID)
	if err != nil {
		log.Fatalf("Error building slack client: %s", err.Error())
	}
	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	ctx := context.Background()

	watcher := kubepkg.NewDeploymentWatcher(kubeClient)

	watcher.Context = ctx
	watcher.SlackClient = slackClient

	log.Info("starting deployment watcher")

	watcher.Run(ctx, stopCh)
}
