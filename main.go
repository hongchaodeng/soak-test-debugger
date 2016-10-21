package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/labels"
)

var (
	namespace string
	podCount  int
)

func init() {
	flag.StringVar(&namespace, "ns", "default", "")
	flag.Parse()
}

func main() {
	kubecli, err := unversioned.NewInCluster()
	if err != nil {
		panic(err)
	}

	for {
		time.Sleep(10 * time.Second)
		pods, err := kubecli.Pods(namespace).List(api.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app": "etcd",
			}),
		})
		if err != nil {
			log.Printf("fail to list pods: %v", err)
			continue
		}
		// print all pods
		for i := range pods.Items {
			pod := &pods.Items[i]
			log.Printf("pod (%v) %v", pod.Name, pod.Status.Phase)
			if pod.Status.Phase != api.PodRunning {
				continue
			}
			buf := bytes.NewBuffer(nil)
			getLogs(kubecli, namespace, pod.Name, "etcd", buf)
			log.Printf("pod (%v) logs ===\n%v\n", pod.Name, buf.String())
		}

		// print all services
		svcs, err := kubecli.Services(namespace).List(api.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app": "etcd",
			}),
		})
		if err != nil {
			log.Printf("fail to list services: %v", err)
			continue
		}

		for i := range svcs.Items {
			svc := &svcs.Items[i]
			log.Printf("svc (%v/%v) ======", svc.Name, svc.Spec.ClusterIP)
			ep, err := kubecli.Endpoints(namespace).Get(svc.Name)
			if err != nil {
				log.Printf("fail to get endpoints for svc (%s): %v", svc.Name, err)
				continue
			}
			log.Printf("endpoints of svc (%s): %v", svc.Name, ep.Subsets)
		}

	}
}

func getLogs(kubecli *unversioned.Client, ns, p, c string, out io.Writer) error {
	req := kubecli.RESTClient.Get().
		Namespace(ns).
		Resource("pods").
		Name(p).
		SubResource("log").
		Param("container", c).
		Param("tailLines", "20")

	readCloser, err := req.Stream()
	if err != nil {
		return err
	}
	defer readCloser.Close()

	_, err = io.Copy(out, readCloser)
	return err
}
