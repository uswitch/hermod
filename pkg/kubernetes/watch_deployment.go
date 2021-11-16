package kubernetes

import (
	"context"
	"fmt"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/uswitch/hermod/pkg/slack"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
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

	hermodGithubRepoAnnotation      string
	hermodGithubCommitSHAAnnotation string
	githubAnnotationWarning         bool
}

const (
	revision = "deployment.kubernetes.io/revision"

	hermodStateAnnotation = "hermod.uswitch.com/state"

	hermodPassState        = "pass"
	hermodFailState        = "fail"
	hermodProgressingState = "progressing"

	hermodAlertFailure = "failure"

	failedCreateReason             = "FailedCreate"
	progressDeadlineExceededReason = "ProgressDeadlineExceeded"
)

func NewDeploymentWatcher(client *kubernetes.Clientset, hermodGithubRepoAnnotation, hermodGithubCommitSHAAnnotation string, githubAnnotationWarning bool) *deploymentInformer {
	deploymentInformer := &deploymentInformer{
		client:                          client,
		hermodGithubRepoAnnotation:      hermodGithubRepoAnnotation,
		hermodGithubCommitSHAAnnotation: hermodGithubCommitSHAAnnotation,
		githubAnnotationWarning:         githubAnnotationWarning,
	}
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

	// check if resourceversion are same
	if deploymentOld.ResourceVersion == deploymentNew.ResourceVersion && deploymentNew.Annotations[hermodStateAnnotation] != hermodProgressingState {
		return
	}

	// get slack channel name from namespace annotation
	slackChannel, err := getSlackChannel(deploymentNew.Namespace, b.namespaceIndexer)
	if err != nil {
		log.Errorf("failed to get slack channel for namespace: %s\n", err)
		return
	}

	if slackChannel == "" {
		log.Debugf("no hermod slack channel specified for namespace: %s\n", deploymentNew.Namespace)
		return
	}

	updateDeployment := deploymentNew.DeepCopy()

	alertLevel, err := getAlertLevel(deploymentNew, b.namespaceIndexer)
	if err != nil {
		log.Errorf("failed to get alert level for deployment: %s\n", err)
		return
	}

	// detecting the deployment rollout
	if deploymentOld.GetAnnotations()[revision] != deploymentNew.GetAnnotations()[revision] && deploymentNew.Annotations[hermodStateAnnotation] != hermodProgressingState {
		msg := fmt.Sprintf("*Rolling out Deployment `%s` in namespace `%s` on `%s` cluster.*", deploymentNew.Name, deploymentNew.Namespace, getClusterName())
		log.Infof(msg)
		err := addAnnotation(b.Context, b.client, deploymentNew.Namespace, updateDeployment, hermodProgressingState)
		if err != nil {
			log.Errorf("failed to add annotation: %v", err)
		}

		// Send message if alertLevel isn't set to Failure only
		if alertLevel != hermodAlertFailure {
			// send message to slack
			err = b.SlackClient.SendMessage(slackChannel, msg, slack.OrangeColor)
			if err != nil {
				log.Errorf("failed to send slack message: %v", err)
			}

			return
		}
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
		deploymentNew.Annotations[hermodStateAnnotation] != "" {

		if deploymentNew.Annotations[hermodStateAnnotation] != hermodPassState {
			err := addAnnotation(b.Context, b.client, deploymentNew.Namespace, updateDeployment, hermodPassState)
			if err != nil {
				log.Errorf("failed to add annotation: %v", err)
			}
			msg := fmt.Sprintf("*Rollout for Deployment `%s` in `%s` namespace on `%s` cluster is successful.*", deploymentNew.Name, deploymentNew.Namespace, getClusterName())
			log.Infof(msg)

			// Send message if alertLevel isn't set to Failure only
			if alertLevel != hermodAlertFailure {
				// send message to slack
				err = b.SlackClient.SendMessage(slackChannel, msg, slack.GreenColor)
				if err != nil {
					log.Errorf("failed to send slack message: %v", err)
				}
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

		if deploymentNew.Annotations[hermodStateAnnotation] != hermodFailState {
			err := addAnnotation(b.Context, b.client, deploymentNew.Namespace, updateDeployment, hermodFailState)
			if err != nil {
				log.Errorf("failed to add annotation: %v", err)
			}
			errorMsg, err := getErrorEvents(b.Context, b.client, deploymentNew.Namespace, updateDeployment)
			if err != nil {
				log.Errorf("failed to get the error events: %v", err)
			}

			repo := deploymentNew.GetAnnotations()[b.hermodGithubRepoAnnotation]
			sha := deploymentNew.GetAnnotations()[b.hermodGithubCommitSHAAnnotation]
			if repo != "" && sha != "" {
				commit := fmt.Sprintf("*Commit:* %s/commit/%s", repo, sha)
				pullRequest := fmt.Sprintf("*Pull Request, if applicable:* %s/pulls/?q=%s", repo, sha)

				errorMsg = errorMsg + fmt.Sprintf("\n\n%s\n\n%s", commit, pullRequest)
			} else if b.githubAnnotationWarning {
				errorMsg = errorMsg + "\n\n" + ":warning: *Could not find annotations" + fmt.Sprintf(" `%s` and `%s` ", b.hermodGithubRepoAnnotation, b.hermodGithubCommitSHAAnnotation) + ", cannot link to Commit or Pull Request*"
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
