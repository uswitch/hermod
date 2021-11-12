package kubernetes

import (
	"os"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const clusterNameEnv = "CLUSTER_NAME"

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

	for k, v := range nsAnnotations {
		if k == hermodSlackChannelAnnotation {
			return v
		}
	}

	return ""
}
