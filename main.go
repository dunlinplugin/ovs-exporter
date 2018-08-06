package main

//This script listens on a given TCP port for
//HTTP REST Get messages than scraps the given
//Open vSwtich entry and gives back the stats
//in Prometheus compatible format

//Written by Megyo @ LeanNet 

import (
    "log"
    "net/http"

    "os"
	"flag"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

//the TCP port that this scripts listens
var listenPort string = ":8081"

//API connection to Kubernetes
var clientset *kubernetes.Clientset

func main() {
    //connect to Kubernetes API
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
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

    router := NewRouter()

    log.Fatal(http.ListenAndServe(listenPort, router))
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
