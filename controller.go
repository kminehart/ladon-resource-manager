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

package main

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/ory/ladon"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	ladonv1alpha1 "github.com/kminehart/ladon-resource-manager/pkg/apis/ladoncontroller/v1alpha1"
	clientset "github.com/kminehart/ladon-resource-manager/pkg/client/clientset/versioned"
	ladonscheme "github.com/kminehart/ladon-resource-manager/pkg/client/clientset/versioned/scheme"
	informers "github.com/kminehart/ladon-resource-manager/pkg/client/informers/externalversions"
	listers "github.com/kminehart/ladon-resource-manager/pkg/client/listers/ladoncontroller/v1alpha1"
)

const controllerAgentName = "ladon-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a Policy is synced
	SuccessSynced = "Synced"

	// MessageResourceSynced is the message used for an Event fired when a Policy
	// is synced successfully
	MessageResourceSynced = "Policy synced successfully"
)

// Controller is the controller implementation for Policy resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface

	// ladonclientset is a clientset for our own API group
	ladonclientset clientset.Interface

	policiesLister listers.PolicyLister
	policiesSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	// warden is responsible for creating, updating, and deleting
	// ladon polices.
	warden ladon.Warden
}

// NewController returns a new ladon controller
func NewController(
	kubeclientset kubernetes.Interface,
	ladonclientset clientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	ladonInformerFactory informers.SharedInformerFactory,
	warden ladon.Warden) *Controller {

	// obtain references to shared index informers for the Deployment and Policy
	// types.
	policyInformer := ladonInformerFactory.Ladoncontroller().V1alpha1().Policies()

	// Create event broadcaster
	// Add ladon-controller types to the default Kubernetes Scheme so Events can be
	// logged for ladon-controller types.
	ladonscheme.AddToScheme(scheme.Scheme)
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:  kubeclientset,
		ladonclientset: ladonclientset,
		policiesLister: policyInformer.Lister(),
		policiesSynced: policyInformer.Informer().HasSynced,
		workqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "policies"),
		recorder:       recorder,
		warden:         warden,
	}

	glog.Info("Setting up event handlers")
	// Set up an event handler for when Policy resources change
	policyInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueuePolicy,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueuePolicy(new)
		},
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	glog.Info("Starting Policy controller")

	// Wait for the caches to be synced before starting workers
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.policiesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	glog.Info("Starting workers")
	// Launch two workers to process Policy resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	glog.Info("Started workers")
	<-stopCh
	glog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
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
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Policy resource to be synced.
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		glog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Policy resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Policy resource with this namespace/name
	policy, err := c.policiesLister.Policies(namespace).Get(name)
	if err != nil {
		// The Policy resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("policy '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}
	glog.V(4).Infof("Handling policy %+v", policy)

	// -- This, and updatepoliciestatus, may be completely useless
	// Finally, we update the status block of the Policy resource to reflect the
	// current state of the world
	err = c.updatepoliciestatus(policy)
	if err != nil {
		return err
	}
	c.recorder.Event(policy, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *Controller) updatepoliciestatus(policy *ladonv1alpha1.Policy) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	policyCopy := policy.DeepCopy()
	// -- This line was removed as it is unlikely we'll be able to sync the status of a ladon resource
	// policyCopy.Status.AvailableReplicas = deployment.Status.AvailableReplicas
	// // Until #38113 is merged, we must use Update instead of UpdateStatus to
	// // update the Status block of the Policy resource. UpdateStatus will not
	// // allow changes to the Spec of the resource, which is ideal for ensuring
	// // nothing other than resource status has been updated.
	_, err := c.ladonclientset.LadoncontrollerV1alpha1().Policies(policy.Namespace).Update(policyCopy)
	return err
}

// enqueuePolicy takes a Policy resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Policy.
func (c *Controller) enqueuePolicy(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}
