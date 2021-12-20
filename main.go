package main

import (
	"context"

	log "github.com/sirupsen/logrus"

	kubepkg "github.com/uswitch/hermod/pkg/kubernetes"
	sentryClient "github.com/uswitch/hermod/pkg/sentry"
	"github.com/uswitch/hermod/pkg/slack"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/sample-controller/pkg/signals"
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

	kubeConfig, err := kubepkg.CreateClientConfig(opts.kubeconfig)
	if err != nil {
		log.Fatalf("error creating kube client config: %s", err)
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	slackClient, err := slack.NewClient()
	if err != nil {
		log.Fatalf("Error building slack client: %s", err.Error())
	}
	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	ctx := context.Background()

	watcher := kubepkg.NewDeploymentWatcher(kubeClient, opts.hermodGithubRepoAnnotation, opts.hermodGithubCommitSHAAnnotation, opts.githubAnnotationWarning)

	watcher.Context = ctx
	watcher.SlackClient = slackClient

	log.Info("starting deployment watcher")

	watcher.Run(ctx, stopCh)
}
