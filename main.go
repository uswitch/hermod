package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/getsentry/sentry-go"
	kubepkg "github.com/uswitch/hermod/pkg/kubernetes"
	sentryClient "github.com/uswitch/hermod/pkg/sentry"
	"github.com/uswitch/hermod/pkg/slack"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/client-go/kubernetes"
)

type options struct {
	kubeconfig string
	logLevel   string

	hermodGithubRepoAnnotation      string
	hermodGithubCommitSHAAnnotation string

	githubAnnotationWarning bool
}

func main() {
	log.Info("starting controller")
	opts := &options{}
	kingpin.Flag("kubeconfig", "Path to kubeconfig.").StringVar(&opts.kubeconfig)
	kingpin.Flag("level", "Log level: debug, info, warn, error.").Default("info").EnumVar(&opts.logLevel, "debug", "info", "warn", "error")
	kingpin.Flag("repo-url-annotation", "Annotation used to retrieve Github repository the deployment is from").Default("hermod.uswitch.com/gitrepo").StringVar(&opts.hermodGithubRepoAnnotation)
	kingpin.Flag("commit-sha-annotation", "Annotation used to retrieve Git SHA responsible for the latest deployment").Default("hermod.uswitch.com/gitsha").StringVar(&opts.hermodGithubCommitSHAAnnotation)
	kingpin.Flag("git-annotation-warning", "Warn for missing `repo-url-annotation` and `commit-sha-annotation values").BoolVar(&opts.githubAnnotationWarning)
	kingpin.Parse()

	configureLogger(opts.logLevel)

	sentryClient.SetupSentry()

	defer sentryClient.Cleanup()

	kubeConfig, err := kubepkg.CreateClientConfig(opts.kubeconfig)
	if err != nil {
		message := fmt.Sprintf("error creating kube client config: %s", err)
		sentry.CaptureMessage(message)
		sentryClient.Cleanup()
		log.Fatalf(message)
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		message := fmt.Sprintf("Error building kubernetes clientset: %s", err.Error())
		sentry.CaptureMessage(message)
		sentryClient.Cleanup()
		log.Fatalf(message)
	}

	slackClient, err := slack.NewClient()
	if err != nil {
		message := fmt.Sprintf("Error building slack client: %s", err.Error())
		sentry.CaptureMessage(message)
		sentryClient.Cleanup()
		log.Fatalf(message)
	}

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGTERM, syscall.SIGINT)

	ctx := context.Background()

	watcher := kubepkg.NewDeploymentWatcher(kubeClient, opts.hermodGithubRepoAnnotation, opts.hermodGithubCommitSHAAnnotation, opts.githubAnnotationWarning)

	watcher.Context = ctx
	watcher.SlackClient = slackClient

	log.Info("starting deployment watcher")

	watcher.Run(ctx, stopCh)
}
