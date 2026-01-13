package kubernetes

import (
	"context"
	"fmt"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

func getErrorEvents(ctx context.Context, client kubernetes.Interface, namespace string, newDeployment *appsv1.Deployment) (string, error) {

	// Get Pod Labels
	podLabels := newDeployment.Spec.Template.Labels

	labelSelector := labels.FormatLabels(podLabels)

	// Find Replicaset based on labels and given revision in annotation
	rs, err := getReplicaSet(ctx, client, namespace, labelSelector, newDeployment.Annotations[revision])
	if err != nil {
		return fmt.Sprintf("failed to get the replicaset: %v", err), err
	}

	// Get Replicaset Labels
	rslabels := rs.GetLabels()
	labelSelector = labels.FormatLabels(rslabels)

	// Get list of Pods based on ReplicaSet labels
	pods, err := getPods(ctx, client, namespace, labelSelector)
	if err != nil {
		return fmt.Sprintf("failed to get the pods: %v", err), err
	}

	// Get errors from Replicaset
	var errorList []string

	if len(pods) == 0 {
		rsConditions := rs.Status.Conditions
		sort.Slice(rsConditions, func(i, j int) bool {
			return rsConditions[i].LastTransitionTime.Before(&rsConditions[j].LastTransitionTime)
		})
		errorList = append(errorList, fmt.Sprintf("```%v```", rsConditions[len(rsConditions)-1].Message))
	} else {
		// Map is to avoid duplicate errors
		reasonMessageMap := make(map[string]string)
		for _, pod := range pods {
			// look for error message in init Containers
			reasonMessageMap = getReasonMessageMapFromStatuses(pod.Status.InitContainerStatuses, reasonMessageMap)

			// look for error message in Containers
			reasonMessageMap = getReasonMessageMapFromStatuses(pod.Status.ContainerStatuses, reasonMessageMap)

			// look for error message in Pod Conditions
			reasonMessageMap = getReasonMessageMapFromPodConditions(pod.Status.Conditions, reasonMessageMap)
		}

		// Sort keys for deterministic ordering
		reasons := make([]string, 0, len(reasonMessageMap))
		for reason := range reasonMessageMap {
			reasons = append(reasons, reason)
		}
		sort.Strings(reasons)

		for _, reason := range reasons {
			errorList = append(errorList, fmt.Sprintf("```\n* %s - %s\n```", reason, reasonMessageMap[reason]))
		}
	}

	// construct error message
	var errorString []string

	// delayed rollout
	if len(errorList) == 0 {
		errorText := fmt.Sprintf("*Deployment `%s` (RS: `%s`) in `%s` namespace failed to reach desired replicas within `%v` seconds on the `%s` cluster, only `%v/%v` replicas are ready.*\n", newDeployment.Name, rs.Name, newDeployment.Namespace, *newDeployment.Spec.ProgressDeadlineSeconds, getClusterName(), newDeployment.Status.ReadyReplicas, *newDeployment.Spec.Replicas)
		return errorText, nil
	}

	// errored rollout
	errorText := fmt.Sprintf("*Rollout for Deployment `%s` (RS: `%s`) in `%s` namespace failed after `%v` seconds on the `%s` cluster.*\n\n*Retrieved the following errors:*", newDeployment.Name, rs.Name, newDeployment.Namespace, *newDeployment.Spec.ProgressDeadlineSeconds, getClusterName())

	errorString = append(errorString, errorText)
	errorString = append(errorString, errorList...)

	return strings.Join(errorString, "\n"), nil
}

func getReasonMessageMapFromPodConditions(conditions []corev1.PodCondition, reasonMessageMap map[string]string) map[string]string {
	for _, condition := range conditions {
		// There are 3 types of Status: True, False, Unknown
		// True means pod is all good, hence we are avoiding that block here
		if condition.Status != corev1.ConditionTrue {
			reasonMessageMap[condition.Reason] = condition.Message
		}
	}

	return reasonMessageMap
}

func getReasonMessageMapFromStatuses(containerStatus []corev1.ContainerStatus, reasonMessageMap map[string]string) map[string]string {
	for _, status := range containerStatus {
		if status.State.Waiting != nil {
			if status.State.Waiting.Reason == "ContainerCreating" {
				continue
			}
			reasonMessageMap[status.State.Waiting.Reason] = status.State.Waiting.Message
		}
	}

	return reasonMessageMap
}
