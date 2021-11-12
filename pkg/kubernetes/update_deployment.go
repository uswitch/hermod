package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// addAnnotation will add the hermod specific annotation to the deployment
func addAnnotation(ctx context.Context, client *kubernetes.Clientset, namespace string, newDeployment *appsv1.Deployment, state string) error {
	patch := map[string]interface{}{
		"metadata": map[string]map[string]string{
			"annotations": {
				hermodStateAnnotation: state,
			}}}

	marshalledPatch, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("error marshalling data to json: %v", err)
	}

	_, err = client.AppsV1().Deployments(namespace).Patch(ctx, newDeployment.Name, types.MergePatchType, marshalledPatch, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	return nil
}
