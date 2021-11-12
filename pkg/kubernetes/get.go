package kubernetes

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// getReplicaSet will return associated replicaset with given deployment based on labelselctor & revision number
func getReplicaSet(ctx context.Context, client kubernetes.Interface, namespace string, labelSelector string, revisionNumber string) (appsv1.ReplicaSet, error) {
	rs, err := client.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return appsv1.ReplicaSet{}, fmt.Errorf("failed to get the replicaset: %v", err)
	}

	for _, rs := range rs.Items {
		if rs.Annotations[revision] == revisionNumber {
			return rs, nil
		}
	}

	return appsv1.ReplicaSet{}, nil
}

// getPods will return list of pods based on given label selectors
func getPods(ctx context.Context, client kubernetes.Interface, namespace string, labelSelector string) ([]corev1.Pod, error) {
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return []corev1.Pod{}, fmt.Errorf("failed to get the pods: %v", err)
	}

	return pods.Items, nil
}
