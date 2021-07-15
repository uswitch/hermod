package kubernetes

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/uswitch/hermod/pkg/slack"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type deploymentInformer struct {
	store            cache.Store
	controller       cache.Controller
	client           *kubernetes.Clientset
	SlackClient      *slack.Client
	Context          context.Context // TODO: Make it private if not needed in any other package
	namespaceIndexer cache.Indexer
}

const (
	slackChannelAnnotation = "hermod.uswitch.com/slack"
	revision               = "deployment.kubernetes.io/revision"
	hermodAnnotation       = "hermod.uswitch.com/state"
	hermodPassState        = "pass"
	hermodFailState        = "fail"
	hermodProgressingState = "progressing"

	progressDeadlineExceededReason = "ProgressDeadlineExceeded"
	failedCreateReason             = "FailedCreate"
)

func NewDeploymentWatcher(client *kubernetes.Clientset) *deploymentInformer {
	deploymentInformer := &deploymentInformer{client: client}
	watcher := cache.NewListWatchFromClient(client.AppsV1().RESTClient(), "deployments", "", fields.Everything())
	deploymentInformer.store, deploymentInformer.controller = cache.NewIndexerInformer(watcher, &appsv1.Deployment{}, time.Minute, deploymentInformer, cache.Indexers{})
	return deploymentInformer
}

func (b *deploymentInformer) OnAdd(obj interface{}) {
}

func (b *deploymentInformer) OnDelete(obj interface{}) {
}

func (b *deploymentInformer) OnUpdate(old, new interface{}) {
	deploymentOld, _ := old.(*appsv1.Deployment)
	deploymentNew, _ := new.(*appsv1.Deployment)

	// get slack channel name from namespace annotation
	slackChannel := getSlackChannel(deploymentNew.Namespace, b.namespaceIndexer)
	if slackChannel == "" {
		log.Debugf("no hermod slack channel specified for namespace: %s\n", deploymentNew.Namespace)
		return
	}

	// check if resourceversion are same
	if deploymentOld.ResourceVersion == deploymentNew.ResourceVersion && deploymentNew.Annotations[hermodAnnotation] != hermodProgressingState {
		return
	}

	updateDeployment := deploymentNew.DeepCopy()

	// detecting the deployment rollout
	if deploymentOld.GetAnnotations()[revision] != deploymentNew.GetAnnotations()[revision] && deploymentNew.Annotations[hermodAnnotation] != hermodProgressingState {
		msg := fmt.Sprintf("Rolling out Deployment `%s` in namespace `%s` on `%s` cluster.", deploymentNew.Name, deploymentNew.Namespace, getClusterName())
		log.Infof(msg)
		err := addAnnotation(b.Context, b.client, deploymentNew.Namespace, updateDeployment, hermodProgressingState)
		if err != nil {
			log.Errorf("failed to add annotation: %v", err)
		}

		// send message to slack
		err = b.SlackClient.SendMessage(slackChannel, msg, slack.OrangeColor)
		if err != nil {
			log.Errorf("failed to send slack message: %v", err)
		}
		return
	}

	// Get the DeploymentCondition and sort them based on time
	deploymentNewConditions := deploymentNew.Status.Conditions
	sort.Slice(deploymentNewConditions, func(i, j int) bool {
		return deploymentNewConditions[i].LastUpdateTime.Before(&deploymentNewConditions[j].LastUpdateTime)
	})

	deploymentOldConditions := deploymentOld.Status.Conditions
	sort.Slice(deploymentOldConditions, func(i, j int) bool {
		return deploymentOldConditions[i].LastUpdateTime.Before(&deploymentOldConditions[j].LastUpdateTime)
	})

	// Successful condition
	if deploymentNewConditions[len(deploymentNewConditions)-1].Status == corev1.ConditionTrue &&
		deploymentNew.Generation == deploymentNew.Status.ObservedGeneration &&
		deploymentNew.Status.Replicas == deploymentNew.Status.ReadyReplicas &&
		deploymentNew.Status.UpdatedReplicas == deploymentNew.Status.ReadyReplicas &&
		*deploymentNew.Spec.Replicas == deploymentNew.Status.ReadyReplicas &&
		deploymentNew.Annotations[hermodAnnotation] != "" {
		if deploymentNew.Annotations[hermodAnnotation] != hermodPassState {
			err := addAnnotation(b.Context, b.client, deploymentNew.Namespace, updateDeployment, hermodPassState)
			if err != nil {
				log.Errorf("failed to add annotation: %v", err)
			}
			msg := fmt.Sprintf("Rollout for Deployment `%s` in `%s` namespace on `%s` cluster is successful.", deploymentNew.Name, deploymentNew.Namespace, getClusterName())
			log.Infof(msg)

			// send message to slack
			err = b.SlackClient.SendMessage(slackChannel, msg, slack.GreenColor)
			if err != nil {
				log.Errorf("failed to send slack message: %v", err)
			}
			return
		}
	}

	// failure condition
	if deploymentNew.Generation == deploymentNew.Status.ObservedGeneration &&
		deploymentOld.Generation == deploymentOld.Status.ObservedGeneration &&
		deploymentNew.Generation == deploymentOld.Generation &&
		deploymentNewConditions[len(deploymentNewConditions)-1].Reason != deploymentOldConditions[len(deploymentOldConditions)-1].Reason &&
		(deploymentNewConditions[len(deploymentNewConditions)-1].Reason == progressDeadlineExceededReason ||
			deploymentNewConditions[len(deploymentNewConditions)-1].Reason == failedCreateReason) {
		if deploymentNew.Annotations[hermodAnnotation] != hermodFailState {
			err := addAnnotation(b.Context, b.client, deploymentNew.Namespace, updateDeployment, hermodFailState)
			if err != nil {
				log.Errorf("failed to add annotation: %v", err)
			}
			errorMsg, err := getErrorEvents(b.Context, b.client, deploymentNew.Namespace, updateDeployment)
			if err != nil {
				log.Errorf("failed to get the error events: %v", err)
			}
			log.Info(errorMsg)

			// send message to slack
			err = b.SlackClient.SendMessage(slackChannel, errorMsg, slack.RedColor)
			if err != nil {
				log.Errorf("failed to send slack message: %v", err)
			}

			return
		}
	}

}

func (b *deploymentInformer) Run(ctx context.Context, stopCh <-chan struct{}) {
	go b.controller.Run(ctx.Done())
	cache.WaitForCacheSync(ctx.Done(), b.controller.HasSynced)
	log.Info("deployment cache controller synced")

	b.namespaceIndexer = watchNamespaces(ctx, b.client)

	<-stopCh
}

func watchNamespaces(context context.Context, client *kubernetes.Clientset) cache.Indexer {
	listWatcher := cache.NewListWatchFromClient(client.CoreV1().RESTClient(), "namespaces", "", fields.Everything())
	indexer, informer := cache.NewIndexerInformer(listWatcher, &corev1.Namespace{}, 0, cache.ResourceEventHandlerFuncs{}, cache.Indexers{})

	go informer.Run(context.Done())

	if !cache.WaitForCacheSync(context.Done(), informer.HasSynced) {
		log.Errorf("Timed out waiting for caches to sync")
		return nil
	}

	log.Info("namespace cache controller synced")

	return indexer
}

func getSlackChannel(namespace string, indexer cache.Indexer) string {
	nsResource, _, _ := indexer.GetByKey(namespace)
	nsAnnotations, _ := meta.NewAccessor().Annotations(nsResource.(runtime.Object))

	for k, v := range nsAnnotations {
		if k == slackChannelAnnotation {
			return v
		}
	}

	return ""

}

func getErrorEvents(ctx context.Context, client *kubernetes.Clientset, namespace string, newDeployment *appsv1.Deployment) (string, error) {

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

	// construct error message
	var errorString []string

	errorText := fmt.Sprintf("Rollout for Deployment `%s` (RS: `%s`) in %s namespace failed after `%v` seconds on the `%s` cluster.\nGot the following errors:", newDeployment.Name, rs.Name, newDeployment.Namespace, *newDeployment.Spec.ProgressDeadlineSeconds, getClusterName())
	errorString = append(errorString, errorText)

	// Get errors from Replicaset
	if len(pods) == 0 {
		rsConditions := rs.Status.Conditions
		sort.Slice(rsConditions, func(i, j int) bool {
			return rsConditions[i].LastTransitionTime.Before(&rsConditions[j].LastTransitionTime)
		})
		errorString = append(errorString, fmt.Sprintf("```%v```", rsConditions[len(rsConditions)-1].Message))
	} else {

		// Map is to avoid duplicate errors
		reasonMessageMap := make(map[string]string)
		for _, pod := range pods {
			// look for error message in init Containers
			reasonMessageMap = getResMsg(pod.Status.InitContainerStatuses, reasonMessageMap)

			// look for error message in Containers
			reasonMessageMap = getResMsg(pod.Status.ContainerStatuses, reasonMessageMap)

		}

		for reason, message := range reasonMessageMap {
			errorString = append(errorString, fmt.Sprintf("```\n* %s - %s\n```", reason, message))
		}
	}

	return strings.Join(errorString, "\n"), nil

}

func getResMsg(containerStatus []corev1.ContainerStatus, reasonMessageMap map[string]string) map[string]string {
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

// getReplicaSet will return associated replicaset with given deployment based on labelselctor & revision number
func getReplicaSet(ctx context.Context, client *kubernetes.Clientset, namespace string, labelSelector string, revisionNumber string) (appsv1.ReplicaSet, error) {

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
func getPods(ctx context.Context, client *kubernetes.Clientset, namespace string, labelSelector string) ([]corev1.Pod, error) {
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})

	if err != nil {
		return []corev1.Pod{}, fmt.Errorf("failed to get the pods: %v", err)
	}
	return pods.Items, nil
}

// addAnnotation will add the hermod specific annotation to the deployment
func addAnnotation(ctx context.Context, client *kubernetes.Clientset, namespace string, newDeployment *appsv1.Deployment, state string) error {
	ann := newDeployment.ObjectMeta.Annotations
	ann[hermodAnnotation] = state
	newDeployment.ObjectMeta.Annotations = ann

	_, err := client.AppsV1().Deployments(namespace).Update(ctx, newDeployment, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
