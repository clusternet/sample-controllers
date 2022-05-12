/*
Copyright 2021 The Clusternet Authors.
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
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	workloadregistry "github.com/clusternet/feedinventory-controller/pkg/registry"

	appsapi "github.com/clusternet/clusternet/pkg/apis/apps/v1alpha1"
	clusternetclientset "github.com/clusternet/clusternet/pkg/generated/clientset/versioned"
	appinformers "github.com/clusternet/clusternet/pkg/generated/informers/externalversions/apps/v1alpha1"
	applisters "github.com/clusternet/clusternet/pkg/generated/listers/apps/v1alpha1"
	"github.com/clusternet/clusternet/pkg/known"
	utils "github.com/clusternet/clusternet/pkg/utils"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

var subKind = appsapi.SchemeGroupVersion.WithKind("Subscription")

const (
	SucessSynced          = "Synced"
	ErrResourceExists     = "ErrResourceExists"
	MessageResourceExists = "Resource %q already exists and is not managed by Feedinventory"
	MessageResourceSynced = "Feedinventory synced successfully"
)

type Controller struct {
	//kubeclientset    kubernetes.Interface
	clusternetClient clusternetclientset.Interface

	subsLister     applisters.SubscriptionLister
	subsSynced     cache.InformerSynced
	finvLister     applisters.FeedInventoryLister
	finvSynced     cache.InformerSynced
	manifestLister applisters.ManifestLister
	manifestSynced cache.InformerSynced

	registry          workloadregistry.Registry
	workqueue         workqueue.RateLimitingInterface
	recorder          record.EventRecorder
	reservedNamespace string
}

// NewController returns a new controller
func NewController(
	clusternetClient clusternetclientset.Interface,
	// kubernetesClient kubernetes.Clientset,
	subsInformer appinformers.SubscriptionInformer,
	finvInformer appinformers.FeedInventoryInformer,
	manifestInformer appinformers.ManifestInformer,

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	recorder record.EventRecorder,
	registry workloadregistry.Registry,
	namespace string,
) *Controller {
	c := &Controller{
		clusternetClient:  clusternetClient,
		subsLister:        subsInformer.Lister(),
		subsSynced:        subsInformer.Informer().HasSynced,
		finvLister:        finvInformer.Lister(),
		finvSynced:        finvInformer.Informer().HasSynced,
		manifestLister:    manifestInformer.Lister(),
		manifestSynced:    manifestInformer.Informer().HasSynced,
		recorder:          recorder,
		registry:          registry,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "feedInventory"),
		reservedNamespace: namespace,
	}

	//set up an event handler for some clusternet crd change
	subsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addSubscription,
		UpdateFunc: c.updateSubscription,
	})

	finvInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteFeedInventory,
	})

	manifestInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addManifest,
		UpdateFunc: c.updateManifest,
	})

	return c
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	klog.Info("Run finv controller ... ")

	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Wait for the caches to be synced before starting workers
	if !cache.WaitForNamedCacheSync("feedInventory-controller", stopCh,
		c.subsSynced, c.finvSynced, c.manifestSynced) {
		return nil
	}

	klog.Infof("Starting %d workers for controller ...", workers)
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
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

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
		}

		// Run the syncHandler, passing it the namespace/name string of the
		// Clusternet resource to be synced.
		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("Error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.workqueue.Forget(obj)
		klog.Infof("[FeedInventory] successfully synced Subscription %q", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}
	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the subs resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	sub, err := c.subsLister.Subscriptions(ns).Get(name)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if sub.DeletionTimestamp != nil {
		return nil
	}
	//just for dividing scheduling strategy type
	if sub.Spec.SchedulingStrategy != appsapi.DividingSchedulingStrategyType {
		return nil
	}
	sub.Kind = subKind.Kind
	sub.APIVersion = subKind.GroupVersion().String()

	finv := &appsapi.FeedInventory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sub.Name,
			Namespace: sub.Namespace,
			Labels: map[string]string{
				known.ObjectCreatedByLabel:             "clusternet-hub",
				known.ConfigSubscriptionNameLabel:      sub.Name,
				known.ConfigSubscriptionNamespaceLabel: sub.Namespace,
				known.ConfigSubscriptionUIDLabel:       string(sub.UID),
			},
		},
		Spec: appsapi.FeedInventorySpec{
			Feeds: make([]appsapi.FeedOrder, len(sub.Spec.Feeds)),
		},
	}
	finv.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(sub, subKind)})

	wg := sync.WaitGroup{}
	wg.Add(len(sub.Spec.Feeds))
	for idx, feed := range sub.Spec.Feeds {
		go func(idx int, feed appsapi.Feed) {
			defer wg.Done()
			manifests, err := utils.ListManifestsBySelector(c.reservedNamespace, c.manifestLister, feed)
			if err != nil {
				klog.Error("Error of list manifest : ", err)
				return
			}
			if manifests == nil {
				klog.Errorf("Manifest %s/%s not fount", feed.Namespace, feed.Name)
			}
			gvk, err := getGroupVersionKind(manifests[0].Template.Raw)
			if err != nil {
				klog.Errorf("Error of get gvk for %s/%s", manifests[0].Namespace, manifests[0].Name)
				return
			}

			var desiredReplicas *int32
			var replicaRequirements appsapi.ReplicaRequirements
			var replicaJsonPath string

			//parse workload
			plugin, ok := c.registry[gvk]
			if ok {
				desiredReplicas, replicaRequirements, replicaJsonPath, err = plugin.Parser(manifests[0].Template.Raw)
				if err != nil {
					return
				}
			}

			finv.Spec.Feeds[idx] = appsapi.FeedOrder{
				Feed:                feed,
				DesiredReplicas:     desiredReplicas,
				ReplicaRequirements: replicaRequirements,
				ReplicaJsonPath:     replicaJsonPath,
			}
		}(idx, feed)
	}
	wg.Wait()

	_, err = c.clusternetClient.AppsV1alpha1().FeedInventories(finv.Namespace).Create(context.TODO(), finv, metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		klog.Info("[FeedInventory] feedinventory is already exists , will update it ... ")
		curFinv, err := c.clusternetClient.AppsV1alpha1().FeedInventories(finv.Namespace).Get(context.TODO(), finv.Name, metav1.GetOptions{})
		if err != nil {
			klog.Error("Error of get current feedinventory: ", err)
			return err
		}
		finv.SetResourceVersion(curFinv.ResourceVersion)
		_, err = c.clusternetClient.AppsV1alpha1().FeedInventories(finv.Namespace).Update(context.TODO(), finv, metav1.UpdateOptions{})
		if err != nil {
			klog.Error("Error of update feedinventory : ", err)
			return err
		}
		klog.Infof("[FeedInventory] update feedinventory %s/%s success ", finv.Namespace, finv.Name)
		return nil
	}
	return err
}

func getGroupVersionKind(rawData []byte) (schema.GroupVersionKind, error) {
	object := &unstructured.Unstructured{}
	if err := json.Unmarshal(rawData, object); err != nil {
		return schema.GroupVersionKind{}, err
	}
	groupVersion, err := schema.ParseGroupVersion(object.GetAPIVersion())
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	return groupVersion.WithKind(object.GetKind()), nil
}

func (c *Controller) addSubscription(obj interface{}) {
	sub := obj.(*appsapi.Subscription)
	klog.V(4).Infof("[FeedInventory] adding Subscription %q", klog.KObj(sub))
	c.enqueue(sub)
}

func (c *Controller) updateSubscription(old, cur interface{}) {
	oldSub := old.(*appsapi.Subscription)
	newSub := cur.(*appsapi.Subscription)

	// Decide whether discovery has reported a spec change.
	if reflect.DeepEqual(oldSub.Spec, newSub.Spec) {
		klog.V(4).Infof("[FeedInventory] no updates on the spec of Subscription %s, skipping syncing", klog.KObj(newSub))
		return
	}

	klog.Infof("[FeedInventory] updating Subscription %q", klog.KObj(newSub))
	c.enqueue(newSub)
}

func (c *Controller) deleteFeedInventory(obj interface{}) {
	finv := obj.(*appsapi.FeedInventory)
	klog.V(4).Infof("[FeedInventory] deleting FeedInventory %q", klog.KObj(finv))
	c.workqueue.AddRateLimited(klog.KObj(finv))
}

func (c *Controller) addManifest(obj interface{}) {
	manifest := obj.(*appsapi.Manifest)
	klog.V(4).Infof("[FeedInventory] adding Manifest %q", klog.KObj(manifest))
	c.enqueueManifest(manifest)
}

func (c *Controller) updateManifest(old, cur interface{}) {
	oldManifest := old.(*appsapi.Manifest)
	newManifest := cur.(*appsapi.Manifest)

	// Decide whether discovery has reported a spec change.
	if reflect.DeepEqual(oldManifest.Template, newManifest.Template) {
		klog.V(4).Infof("[FeedInventory] no updates on Manifest template %s, skipping syncing", klog.KObj(oldManifest))
		return
	}

	klog.V(4).Infof("[FeedInventory] updating Manifest %q", klog.KObj(oldManifest))
	c.enqueueManifest(newManifest)
}

func (c *Controller) enqueueManifest(manifest *appsapi.Manifest) {
	if manifest.DeletionTimestamp != nil {
		return
	}

	subUIDs := []string{}
	for k, v := range manifest.GetLabels() {
		if v == subKind.Kind {
			subUIDs = append(subUIDs, k)
		}
	}

	allSubs := []*appsapi.Subscription{}
	for _, subUID := range subUIDs {
		subs, err := c.subsLister.List(labels.SelectorFromSet(labels.Set{
			subUID: subKind.Kind,
		}))
		if err != nil {
			runtime.HandleError(err)
			return
		}
		allSubs = append(allSubs, subs...)
	}

	for _, sub := range allSubs {
		c.enqueue(sub)
	}
}

func (c *Controller) enqueue(sub *appsapi.Subscription) {
	key, err := cache.MetaNamespaceKeyFunc(sub)
	if err != nil {
		return
	}
	c.workqueue.AddRateLimited(key)
}
