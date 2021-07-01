package kubernetes

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/uswitch/hermod/pkg/slack"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
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
	slackChannelAnnotation = "com.uswitch.hermod/slack"
	revision               = "deployment.kubernetes.io/revision"
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

	slackChannel := getSlackChannel(deploymentNew.Namespace, b.namespaceIndexer)
	if slackChannel == "" {
		log.Debugf("no hermod slack channel specified for namespace: %s\n", deploymentNew.Namespace)
		return
	}
	// check if it's a new deployment for now just print

	if deploymentOld.GetAnnotations()[revision] != deploymentNew.GetAnnotations()[revision] {
		fmt.Println("deploying:", deploymentNew.Name, "in namespace:", deploymentNew.Namespace)
	}

	// follow new deployment and check status of replicas match what is expected to sure a successful rollout
	// for now just print the outcome, this will eventually be passed to a new function.

	if deploymentNew.Status.Replicas == deploymentNew.Status.ReadyReplicas && deploymentNew.Status.UpdatedReplicas == deploymentNew.Status.Replicas {
		fmt.Println("deployment successfull")
		return
	} else {
		fmt.Println("deployment failed:", deploymentNew.Status.Conditions)
		return
	}
}

func (b *deploymentInformer) Run(ctx context.Context, stopCh <-chan struct{}) {
	go b.controller.Run(ctx.Done())
	cache.WaitForCacheSync(ctx.Done(), b.controller.HasSynced)
	log.Info("cache controller synced")

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
