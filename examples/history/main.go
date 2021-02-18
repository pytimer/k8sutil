package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/pytimer/k8sutil/history"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var (
		kubeconfig *string
		kind       string
		namespace  string
		name       string
	)

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.StringVar(&kind, "kind", "Deployment", "The resource kind you want to view")
	flag.StringVar(&namespace, "namespace", metav1.NamespaceDefault, "The resource namespace")
	flag.StringVar(&name, "name", "", "The resource name")
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	viewer := history.ViewerFor(client, schema.GroupKind{Group: "apps", Kind: kind})
	latest, olds, err := viewer.ViewHistory(namespace, name)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("New Revision: %d\n", latest[0].Revision)
	for _, o := range olds {
		fmt.Printf("Old Revision: %d\n", o.Revision)
	}
}

func intToPointer(i int) *int {
	return &i
}
