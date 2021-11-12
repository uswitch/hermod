package kubernetes

import (
	"os"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	clusterNameEnv = "CLUSTER_NAME"

	hermodAlertAnnotation        = "hermod.uswitch.com/alert"
	hermodSlackChannelAnnotation = "hermod.uswitch.com/slack"
)

func CreateClientConfig(kubeConfigPath string) (*rest.Config, error) {
	if kubeConfigPath == "" {
		return rest.InClusterConfig()
	}
	return clientcmd.BuildConfigFromFlags("", kubeConfigPath)
}

func getClusterName() string {
	return os.Getenv(clusterNameEnv)
}

func getSlackChannel(namespace string, indexer cache.Indexer) string {
	nsResource, _, _ := indexer.GetByKey(namespace)
	nsAnnotations, _ := meta.NewAccessor().Annotations(nsResource.(runtime.Object))

	return nsAnnotations[hermodSlackChannelAnnotation]
}

func getAlertLevel(deployment *appsv1.Deployment, indexer cache.Indexer) string {
	alertLevel := deployment.GetAnnotations()[hermodAlertAnnotation]

	if alertLevel == "" {
		nsResource, _, _ := indexer.GetByKey(deployment.Namespace)
		nsAnnotations, _ := meta.NewAccessor().Annotations(nsResource.(runtime.Object))

		alertLevel = nsAnnotations[hermodAlertAnnotation]
	}

	return alertLevel
}
