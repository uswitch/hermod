package kubernetes

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type deploymentInformer struct {
	store      cache.Store
	controller cache.Controller
	client     *kubernetes.Clientset
	Context    context.Context // TODO: Make it private if not needed in any other package
}

const revision = "deployment.kubernetes.io/revision"

func NewDeploymentWatcher(client *kubernetes.Clientset) *deploymentInformer {
	deploymentInformer := &deploymentInformer{}
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
	<-stopCh
}
