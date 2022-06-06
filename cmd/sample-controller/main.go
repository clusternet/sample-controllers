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

package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clusternet/sample-controller/pkg/controller"
	workloadregistry "github.com/clusternet/sample-controller/pkg/registry"

	clusternet "github.com/clusternet/clusternet/pkg/generated/clientset/versioned"
	informers "github.com/clusternet/clusternet/pkg/generated/informers/externalversions"
	"github.com/clusternet/clusternet/pkg/known"

	corev1 "k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/controller-manager/pkg/clientbuilder"
	"k8s.io/klog/v2"
)

var (
	masterURL            string
	kubeconfig           string
	onlyOneSignalHandler = make(chan struct{})
	shutdownSignals      = []os.Signal{os.Interrupt, syscall.SIGTERM}
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	// This sample is a feedinventory controller
	rootClientBuilder := clientbuilder.SimpleControllerClientBuilder{
		ClientConfig: cfg,
	}
	kubeClient := kubernetes.NewForConfigOrDie(rootClientBuilder.ConfigOrDie("feedinventory-controller-client"))
	clusternetClient := clusternet.NewForConfigOrDie(rootClientBuilder.ConfigOrDie("feedinventory-controller-client"))

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	clusternetInformerFactory := informers.NewSharedInformerFactory(clusternetClient, known.DefaultResync)

	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&v1core.EventSinkImpl{
		Interface: kubeClient.CoreV1().Events(""),
	})
	recorder := broadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "feedinventory-controller"})
	c := controller.NewController(
		clusternetClient,
		clusternetInformerFactory.Apps().V1alpha1().Subscriptions(),
		clusternetInformerFactory.Apps().V1alpha1().FeedInventories(),
		clusternetInformerFactory.Apps().V1alpha1().Manifests(),
		recorder,
		workloadregistry.NewRegistry(),
		"clusternet-reserved",
	)
	kubeInformerFactory.Start(stopCh)
	clusternetInformerFactory.Start(stopCh)

	if err = c.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

// SetupSignalHandler registered for SIGTERM and SIGINT. A stop channel is returned
// which is closed on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
func SetupSignalHandler() (stopCh <-chan struct{}) {
	close(onlyOneSignalHandler) // panics when called twice

	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, shutdownSignals...)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig , Only if required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
