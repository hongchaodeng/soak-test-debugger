package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/Sirupsen/logrus"
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

	fmt.Println("initial delaying...")
	time.Sleep(30 * time.Second) // initial delay

	for {
		time.Sleep(20 * time.Second)
		pods, err := kubecli.Pods(namespace).List(api.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app": "etcd",
			}),
		})
		if err != nil {
			logrus.Errorf("fail to list pods: %v", err)
			continue
		}
		// print all pods
		for i := range pods.Items {
			pod := &pods.Items[i]
			logrus.Infof("pod (%v) ======", pod.Name)
			logrus.Infof("pod status: %v %v", pod.Status.Phase, pod.Status.Conditions)
			if pod.Status.Phase != api.PodRunning {
				continue
			}
			buf := bytes.NewBuffer(nil)
			getLogs(kubecli, namespace, pod.Name, "etcd", buf)
			logrus.Infof("pod (%v) logs === %v", pod.Name, buf.String())
		}

		// print all services
		svcs, err := kubecli.Services(namespace).List(api.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app": "etcd",
			}),
		})
		if err != nil {
			logrus.Errorf("fail to list services: %v", err)
			continue
		}

		for i := range svcs.Items {
			svc := &svcs.Items[i]
			logrus.Infof("svc (%v/%v) ======", svc.Name, svc.Spec.ClusterIP)

			r, err := kubecli.Client.Get(fmt.Sprintf("http://%s:2379", svc.Spec.ClusterIP))
			if err != nil {
				logrus.Errorf("fail to get %s:2380: %v", svc.Spec.ClusterIP, err)
				continue
			}
			r.Body.Close()

			r, err = kubecli.Client.Get(fmt.Sprintf("http://%s:2379", svc.Spec.ClusterIP))
			if err != nil {
				logrus.Errorf("fail to get %s:2379: %v", svc.Spec.ClusterIP, err)
			}
			r.Body.Close()
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
		Param("tailLines", "10")

	readCloser, err := req.Stream()
	if err != nil {
		return err
	}
	defer readCloser.Close()

	_, err = io.Copy(out, readCloser)
	return err
}
