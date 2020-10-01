package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	namespace := "default"

	podListWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), string(v1.ResourcePods), namespace, fields.Everything())
	_, controller := cache.NewInformer(
		podListWatcher,
		&v1.Pod{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod, ok := obj.(*v1.Pod)
				if ok {
					owner := pod.ObjectMeta.GetOwnerReferences()
					if len(owner) != 0 {
						fmt.Printf("Pod <%s> was added by %s <%s>.\n", pod.ObjectMeta.GetName(), owner[0].Kind, owner[0].Name)
					} else {
						fmt.Printf("Pod <%s> was added.\n", pod.ObjectMeta.GetName())
					}
				} else {
					key, err := cache.MetaNamespaceKeyFunc(obj)
					if err != nil {
						panic(err.Error())
					}
					fmt.Printf("%s was added\n", key)
				}
			},
			UpdateFunc: func(old interface{}, new interface{}) {
				podNew, okNew := new.(*v1.Pod)
				podOld, okOld := old.(*v1.Pod)
				if okNew && okOld {
					if podNew.Status.Phase == podOld.Status.Phase {
						fmt.Printf("Pod <%s> is %s phase.\n", podNew.ObjectMeta.GetName(), podNew.Status.Phase)
					} else {
						fmt.Printf("Pod <%s> is %s phase. previous: %s phase\n", podNew.ObjectMeta.GetName(), podNew.Status.Phase, podOld.Status.Phase)
					}
				} else {
					key, err := cache.MetaNamespaceKeyFunc(new)
					if err != nil {
						panic(err.Error())
					}
					fmt.Printf("%s was modified\n", key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				pod, ok := obj.(*v1.Pod)
				if ok {
					fmt.Printf("Pod <%s> was deleted.\n", pod.ObjectMeta.GetName())
				} else {
					key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
					if err != nil {
						panic(err.Error())
					}
					fmt.Printf("%s was deleted\n", key)
				}
			},
		},
	)

	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(stop)
	// Wait forever
	select {}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
