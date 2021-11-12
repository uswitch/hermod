package kubernetes

import (
	"fmt"
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

func getSlackChannel(namespace string, indexer cache.Indexer) (string, error) {
	nsResource, _, err := indexer.GetByKey(namespace)
	if err != nil {
		return "", fmt.Errorf("failed to get namespace from cache: %s", err)
	}

	nsAnnotations, err := meta.NewAccessor().Annotations(nsResource.(runtime.Object))
	if err != nil {
		return "", fmt.Errorf("failed to get annotations from namespace: %s", err)
	}

	return nsAnnotations[hermodSlackChannelAnnotation], nil
}

func getAlertLevel(deployment *appsv1.Deployment, indexer cache.Indexer) (string, error) {
	alertLevel := deployment.GetAnnotations()[hermodAlertAnnotation]

	if alertLevel == "" {
		nsResource, _, err := indexer.GetByKey(deployment.Namespace)
		if err != nil {
			return "", fmt.Errorf("failed to get namespace from cache: %s", err)
		}

		nsAnnotations, err := meta.NewAccessor().Annotations(nsResource.(runtime.Object))
		if err != nil {
			return "", fmt.Errorf("failed to get annotations from namespace: %s", err)
		}

		alertLevel = nsAnnotations[hermodAlertAnnotation]
	}

	return alertLevel, nil
}
