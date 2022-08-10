package main

import (
	"flag"
	"fmt"
	"os"

	clusternetfinv "github.com/clusternet/clusternet/pkg/controllers/apps/feedinventory"
	clusternet "github.com/clusternet/clusternet/pkg/generated/clientset/versioned"
	informers "github.com/clusternet/clusternet/pkg/generated/informers/externalversions"
	"github.com/clusternet/clusternet/pkg/known"
	clusternetutils "github.com/clusternet/clusternet/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	componentbaseconfig "k8s.io/component-base/config"
	"k8s.io/klog/v2"

	"github.com/clusternet/sample-controller/pkg/feedinventory"
)

var (
	kubeconfig string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig , Only if required if out-of-cluster.")
}

func main() {
	klog.InitFlags(flag.CommandLine)
	defer klog.Flush()

	flag.Parse()

	config, err := clusternetutils.LoadsKubeConfig(&componentbaseconfig.ClientConnectionConfiguration{
		Kubeconfig: kubeconfig,
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	kubeClient := kubernetes.NewForConfigOrDie(config)
	clusternetClient := clusternet.NewForConfigOrDie(config)
	clusternetInformerFactory := informers.NewSharedInformerFactory(clusternetClient, known.DefaultResync)

	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&v1core.EventSinkImpl{
		Interface: kubeClient.CoreV1().Events(""),
	})
	recorder := broadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "feedinventory-controller"})

	controller, err := clusternetfinv.NewController(
		clusternetClient,
		clusternetInformerFactory.Apps().V1alpha1().Subscriptions(),
		clusternetInformerFactory.Apps().V1alpha1().FeedInventories(),
		clusternetInformerFactory.Apps().V1alpha1().Manifests(),
		recorder,
		feedinventory.NewRegistry(),
		known.ClusternetReservedNamespace,
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	controller.Run(2, clusternetutils.GracefulStopWithContext().Done())
}
