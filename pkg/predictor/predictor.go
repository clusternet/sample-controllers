/*
Copyright 2022 The Clusternet Authors.
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

package predictor

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/informers"
	informer "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	appsapi "github.com/clusternet/clusternet/pkg/apis/apps/v1alpha1"
)

// PredictorServer is a server for predict request.
type PredictorServer struct {
	Port uint
	Ctx  context.Context

	k8sClient    kubernetes.Interface
	factory      informers.SharedInformerFactory
	nodeInformer informer.NodeInformer
}

// NewPredictorServer return a predictor server
func NewPredictorServer(options PredictorOptions) (*PredictorServer, error) {
	restConfig, err := clientcmd.BuildConfigFromFlags(options.MasterURL, options.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("error of create kubernetes restConfig : %v", err)
	}
	kubeClient := kubernetes.NewForConfigOrDie(restConfig)
	informerFactory := informers.NewSharedInformerFactory(kubeClient, 0)
	ctx := context.Background()
	return &PredictorServer{
		Port:         options.Port,
		Ctx:          ctx,
		k8sClient:    kubeClient,
		factory:      informerFactory,
		nodeInformer: informerFactory.Core().V1().Nodes(),
	}, nil
}

func (p *PredictorServer) Run() error {
	klog.Infof("Run predictor server with port %d ... ", p.Port)

	stopper := make(chan struct{})
	defer close(stopper)
	informer := p.nodeInformer.Informer()
	go p.factory.Start(stopper)
	if !cache.WaitForCacheSync(stopper, informer.HasSynced) {
		klog.Info("time our waiting for cache to sync")
	}

	http.HandleFunc("/accept", p.MaxAcceptableReplicas)
	http.HandleFunc("/unschedul", p.UnschedulableReplicas)
	err := http.ListenAndServe(fmt.Sprintf(":%d", p.Port), nil)
	if err != nil {
		return err
	}
	return nil
}

// MaxAcceptAbleReplicas is a http handler for max replicas reqeust
func (p *PredictorServer) MaxAcceptableReplicas(w http.ResponseWriter, r *http.Request) {
	var matchNode = make(map[string]int64)
	var require appsapi.ReplicaRequirements
	var maxReplicas int64

	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.Info("error of read request body : ", err)
	}

	err = json.Unmarshal(requestBody, &require)
	if err != nil {
		klog.Info("error of read request body : ", err)
	}
	labelSelector := labels.NewSelector()
	for k, v := range require.NodeSelector {
		requ, _ := labels.NewRequirement(k, selection.Equals, []string{v})
		labelSelector.Add(*requ)
	}
	nodeList, err := p.nodeInformer.Lister().List(labelSelector)
	if err != nil {
		klog.Info("error of list node : ", err)
	}
	for _, n := range nodeList {
		if replicas := p.checkNodeResource(n, require); replicas > 0 {
			matchNode[n.Name] = replicas
		}
	}
	for _, v := range matchNode {
		maxReplicas += v
	}
	clusterMax := p.checkClusterResource(require, nodeList, matchNode)
	if clusterMax < maxReplicas {
		maxReplicas = clusterMax
	}
	if _, err = w.Write([]byte(strconv.FormatInt(maxReplicas, 10))); err != nil {
		klog.Error(err)
	}
}

func (p *PredictorServer) UnschedulableReplicas(w http.ResponseWriter, r *http.Request) {
	// TODO: add real logic
}

func (p *PredictorServer) checkClusterResource(_ appsapi.ReplicaRequirements, nodes []*corev1.Node, matchNode map[string]int64) int64 {
	// If your cluster node set annotation "tke.cloud.tencent.com/available-ip-count" value represent available ip count
	var ipCount int64
	for _, n := range nodes {
		if _, isok := matchNode[n.Name]; isok {
			ip, _ := strconv.Atoi(n.Labels["tke.cloud.tencent.com/available-ip-count"])
			ipCount += int64(ip)
		}
	}
	// TODO: add other logic
	return ipCount
}

func (p *PredictorServer) checkNodeResource(n *corev1.Node, require appsapi.ReplicaRequirements) int64 {
	var replicas int64 = 1000
	var tt bool = false
	if len(n.Spec.Taints) != 0 {
		for _, taint := range n.Spec.Taints {
			for _, toleration := range require.Tolerations {
				if toleration.ToleratesTaint(&taint) {
					tt = true
					break
				}
			}
		}
		if !tt {
			klog.Info("there is no match node for requirements with taint/toleration")
			return 0
		}
	}

	// Some node annotation represent resource is not enough
	if n.Annotations["tke.cloud.tencent.com/res-cloud-hssd"] == "false" {
		return 0
	}
	for resourceName, resource := range require.Resources.Requests {
		if resource.Cmp(*n.Status.Capacity.Name(resourceName, resource.Format)) > 0 {
			klog.Infof("node %s resource %s(%d) is not enough for request %d",
				n.Name, resourceName, n.Status.Capacity.Name(resourceName, resource.Format).Value(), resource.Value())
			return 0
		} else {
			//Use resource Value() beause resource is too big in eks cluster, will concern int64
			//when pod request resource less then 1c , will use 1c to estimat
			multiple := n.Status.Capacity.Name(resourceName, resource.Format).Value() / resource.Value()
			klog.Infof("resource %s: node(%s) has %d, pod need %d, replicas is %d.",
				resourceName, n.Name, n.Status.Capacity.Name(resourceName, resource.Format).Value(), resource.Value(), multiple)
			if replicas > multiple {
				replicas = multiple
			}
		}
	}
	return replicas
}
