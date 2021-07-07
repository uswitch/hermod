package kubernetes

import (
	"os"

	"k8s.io/client-go/rest"
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
