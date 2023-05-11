/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pkg

import (
	"context"
	"fmt"
	//	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	//quotav1alpha1 "github.com/SUMMERLm/quota/pkg/apis/cluster/v1alpha1"
	clientset "github.com/SUMMERLm/quota/pkg/generated/clientset/versioned"
	quotascheme "github.com/SUMMERLm/quota/pkg/generated/clientset/versioned/scheme"
	informers "github.com/SUMMERLm/quota/pkg/generated/informers/externalversions/serverless/v1"
	listers "github.com/SUMMERLm/quota/pkg/generated/listers/serverless/v1"
)

const controllerAgentName = "hyperNode-controller"
const controllerParentAgentName = "hyperNode-Parent-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a Quota is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Quota fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Quota"
	// MessageResourceSynced is the message used for an Event fired when a Quota
	// is synced successfully
	MessageResourceSynced = "quota synced successfully"
)

// Controller is the controller implementation for Quota resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// quotaclientset is a clientset for our own API group
	quotaclientset       clientset.Interface
	quotaParentclientset clientset.Interface
	quotasLister         listers.QuotaLister
	quotasSynced         cache.InformerSynced
	quotasParentLister   listers.QuotaLister
	quotasParentSynced   cache.InformerSynced
	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue       workqueue.RateLimitingInterface
	parentworkqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder      record.EventRecorder
	parentrecoder record.EventRecorder
}

// NewController returns a new sample controller
func NewController(
	kubeclientset kubernetes.Interface,
	quotaclientset clientset.Interface,
	quotaParentclientset clientset.Interface,
	quotaInformer informers.QuotaInformer,
	quotaParentInformer informers.QuotaInformer) *Controller {

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	utilruntime.Must(quotascheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})
	parentrecoder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerParentAgentName})

	controller := &Controller{
		kubeclientset:        kubeclientset,
		quotaclientset:       quotaclientset,
		quotaParentclientset: quotaParentclientset,
		quotasLister:         quotaInformer.Lister(),
		quotasSynced:         quotaInformer.Informer().HasSynced,
		quotasParentLister:   quotaParentInformer.Lister(),
		quotasParentSynced:   quotaParentInformer.Informer().HasSynced,

		workqueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Quotas"),
		parentworkqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "QuotasParent"),
		recorder:        recorder,
		parentrecoder:   parentrecoder,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when local Quota resources change
	quotaInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueQuota,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueQuota(new)
		},
		DeleteFunc: controller.enqueueQuotaForDelete,
	})
	// Set up an event handler for when Parent Quota  resources change
	quotaParentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueQuotaParent,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueQuotaParent(new)
		},
		DeleteFunc: controller.enqueueQuotaParentForDelete,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	defer c.parentworkqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Quota controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.quotasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	klog.Info("Waiting for parent informer caches to sync")

	if ok := cache.WaitForCacheSync(stopCh, c.quotasParentSynced); !ok {
		return fmt.Errorf("failed to wait for parent caches to sync")
	}
	klog.Info("Starting local workers")
	// Launch two workers to process Quota resources
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Starting parent workers")
	for i := 0; i < workers; i++ {
		go wait.Until(c.runParentWorker, time.Second, stopCh)
	}
	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) runParentWorker() {
	for c.processNextParentWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Quota resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) processNextParentWorkItem() bool {
	objparent, shutdownparent := c.parentworkqueue.Get()
	if shutdownparent {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	errparent := func(objparent interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.parentworkqueue.Done(objparent)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = objparent.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.parentworkqueue.Forget(objparent)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", objparent))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Quota resource to be synced.
		if err := c.syncHandlerParent(key); err != nil {
			// Put the item back on the parentworkqueue to handle any transient errors.
			c.parentworkqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.parentworkqueue.Forget(objparent)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(objparent)

	if errparent != nil {
		utilruntime.HandleError(errparent)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Quota resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Quota resource with this namespace/name
	quota, err := c.quotasLister.Quotas(namespace).Get(name)
	if err != nil {
		// The Quota resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			klog.Infof("local Quota对象被删除，请在这里执行实际的删除业务: %s/%s ...", namespace, name)
			utilruntime.HandleError(fmt.Errorf("Quota '%s' in work queue no longer exists", key))
			return nil
		}
		klog.Infof(err.Error())
		return nil
	}
	specAreaName := quota.Spec.ChildName
	klog.Info(specAreaName)
	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	//if err != nil {
	//	return err
	//}

	c.recorder.Event(quota, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Quota resource
// with the current status of the resource.
func (c *Controller) syncHandlerParent(key string) error {

	//hostname, err := os.Hostname()
	clusterName, err := c.kubeclientset.CoreV1().Secrets("gaia-system").Get(context.TODO(), "parent-cluster", metav1.GetOptions{})
	//hostname := clusterName.Labels["clusters.gaia.io/cluster-name"]
	hostname := clusterName.Labels["clusters.gaia.io/cluster-name"]
	klog.Infof(hostname)
	if err != nil {
		return err
	}

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	quotaParent, err := c.quotasParentLister.Quotas(namespace).Get(name)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid parent resource key: %s", key))
	}
	//该node cr被删除，则进入err ！=nil
	if err != nil {
		// The Parent Quota resource may no longer exist, in which case delete local HyperNode cr
		// processing.
		klog.Info(quotaParent)
		if errors.IsNotFound(err) {
			klog.Infof("parent Quota 对象被删除，local执行相应业务: %s/%s ...", namespace, name)
			utilruntime.HandleError(fmt.Errorf("Quota '%s' in work queue no longer exists", key))
			//c.quotaclientset.ClusterV1alpha1().Quotas(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
			//klog.Infof("Delete target %q.\n", quotaParent.Name)
			return nil
		}
	} else {
		//该node为新建或者更新
		//根据本节点所属gaia的层级进行对应动作
		//*****关联关系:根据集群主节点主机名称进行识别，匹配对应hyperParentNode cr里面spec字段*****
		//gaia的crd获取本节点及对应父节点的关联关系，进行quota的cr刷选并在本地进行quota创建
		// Get the parent quota resource with this namespace/name
		//quotaParent, err := c.quotasParentLister.Quotas(namespace).Get(name)
		//if err != nil {
		//	panic(err)
		//}
		quotaParentCopy := quotaParent.DeepCopy()
		quotaLocal, err := c.quotasLister.Quotas(namespace).Get(name)
		if err == nil {
			//更新操作
			//Todo:判断该cr是不是还属于该集群：判断条件为spec字段，对齐新建操作。
			//if quotaParent.Spec.MyAreaName == hostname || quotaParent.Spec.SupervisorName == hostname {
			quotaLocalCopy := quotaLocal.DeepCopy()
			//quotaParentCopy.ResourceVersion = ""
			//quotaParentCopy.APIVersion = ""
			//quotaParent2.Annotations = map[string]string{}
			quotaParentCopy.UID = ""
			//quotaParentCopy.CreationTimestamp =
			klog.Info(quotaParentCopy.Name)
			klog.Info(quotaLocalCopy.Name)
			klog.Info(quotaParentCopy.Labels["quota.cluster.pml.com.cn/NodeType"])
			klog.Info(quotaParentCopy.Labels["quota.cluster.pml.com.cn/Bluetooth"])
			update := 0
			if len(quotaParentCopy.Labels) != len(quotaLocalCopy.Labels) {
				update = 1
			}
			if update == 0 {
				for key, value := range quotaParentCopy.Labels {
					klog.Info(key)
					klog.Infof(value)
					if quotaParentCopy.Labels[key] != quotaLocalCopy.Labels[key] {
						update = 1
					}
				}
			}

		}
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	return nil
}

// enqueueQuota takes a Quota resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Quota.
func (c *Controller) enqueueQuota(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// enqueueQuotaarent takes a Quota resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Quota.
func (c *Controller) enqueueQuotaParent(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.parentworkqueue.Add(key)
}

// 删除操作
func (c *Controller) enqueueQuotaForDelete(obj interface{}) {
	var key string
	var err error
	// 从缓存中删除指定对象
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	//再将key放入队列
	c.workqueue.AddRateLimited(key)
}

// 删除操作
func (c *Controller) enqueueQuotaParentForDelete(obj interface{}) {
	var key string
	var err error
	// 从缓存中删除指定对象
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	//再将key放入队列
	c.parentworkqueue.AddRateLimited(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the Quota resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that Quota resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {

		return
	}
}
