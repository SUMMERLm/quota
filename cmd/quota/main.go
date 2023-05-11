package main

import (
	"flag"
	"github.com/SUMMERLm/quota/pkg"
	clientset "github.com/SUMMERLm/quota/pkg/generated/clientset/versioned"
	informers "github.com/SUMMERLm/quota/pkg/generated/informers/externalversions"
	"github.com/SUMMERLm/quota/pkg/signals"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"time"
)

//var (
//	masterURL        string
//	kubeconfig       string
//	kubeParentConfig string
//)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	//cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	//parentCfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeParentConfig)
	cfg, err := clientcmd.BuildConfigFromFlags("", "/conf/quota/config")
	parentCfg, err := clientcmd.BuildConfigFromFlags("", "/conf/quota/parentConfig")
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	quotasClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building quota clientset: %s", err.Error())
	}

	quotaInformerFactory := informers.NewSharedInformerFactory(quotasClient, time.Second*30)

	quotasParentClient, err := clientset.NewForConfig(parentCfg)
	if err != nil {
		klog.Fatalf("Error building quota clientset: %s", err.Error())
	}

	quotaParentInformerFactory := informers.NewSharedInformerFactory(quotasParentClient, time.Second*30)

	//controller  local and master together
	controller := pkg.NewController(kubeClient, quotasClient, quotasParentClient,
		quotaInformerFactory.Serverless().V1().Quotas(),
		quotaParentInformerFactory.Serverless().V1().Quotas())

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	quotaInformerFactory.Start(stopCh)
	quotaParentInformerFactory.Start(stopCh)
	if err = controller.Run(1, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

//func init() {
//	//flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
//	//flag.StringVar(&kubeParentConfig, "kubeParentConfig", "", "Path to a kubeParentConfig. Only required if out-of-cluster.")
//	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
//}
