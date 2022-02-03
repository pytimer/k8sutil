package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"

	"github.com/pytimer/k8sutil/terminal"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	client kubernetes.Interface
	config *rest.Config
)

func handler(w http.ResponseWriter, r *http.Request) {
	urlValues := r.URL.Query()
	namespaces, ok := urlValues["namespace"]
	if !ok || len(namespaces) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	namespace := namespaces[0]

	podnames, ok := urlValues["pod"]
	if !ok || len(podnames) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	podName := podnames[0]

	containers, ok := urlValues["container"]
	if !ok || len(containers) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	containerName := containers[0]

	commands, ok := urlValues["shell"]
	if !ok || len(containers) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	cmd := commands[0]

	ts, err := terminal.NewTerminalSession(client, w, r, nil)
	if err != nil {
		log.Printf("unable to upgrade websocket, %v", err)
		return
	}
	defer ts.Close()

	if err := ts.Exec(config, namespace, podName, containerName, []string{cmd}, &terminal.ExecOptions{
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		TTY:    true,
	}); err != nil {
		log.Printf("unable to execute stream in container, %v", err)
		ts.Done()
		return
	}
}

func main() {
	var (
		kubeconfig *string
		err        error
	)

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	client, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	http.HandleFunc("/exec", handler)
	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	log.Println("Listen on :8080")
	http.ListenAndServe(":8080", nil)

}
