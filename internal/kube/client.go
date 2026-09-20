// Package kube provides Kubernetes discovery and remote command execution.
package kube

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client accesses the cluster selected by kubeconfig loading rules.
type Client struct {
	clientset  kubernetes.Interface
	restConfig *rest.Config
	namespace  string
	context    string
}

// Pod describes a pod for display and selection.
type Pod struct {
	Name       string
	Phase      string
	Ready      int
	Containers int
	Restarts   int32
	Age        time.Duration
}

// New loads Kubernetes configuration, honoring KUBECONFIG and the default kubeconfig path.
func New(kubeconfigPath string) (*Client, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		rules.ExplicitPath = kubeconfigPath
	}
	deferred := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		rules,
		&clientcmd.ConfigOverrides{},
	)
	restConfig, err := deferred.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	namespace, _, err := deferred.Namespace()
	if err != nil {
		return nil, fmt.Errorf("resolve current namespace: %w", err)
	}
	rawConfig, err := deferred.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig contexts: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes client: %w", err)
	}
	return &Client{
		clientset:  clientset,
		restConfig: restConfig,
		namespace:  namespace,
		context:    rawConfig.CurrentContext,
	}, nil
}

// Namespace returns the namespace selected by the active kubeconfig context.
func (c *Client) Namespace() string { return c.namespace }

// Context returns the active kubeconfig context name.
func (c *Client) Context() string { return c.context }

// Namespaces lists namespace names.
func (c *Client) Namespaces(ctx context.Context) ([]string, error) {
	result, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}
	names := make([]string, 0, len(result.Items))
	for _, namespace := range result.Items {
		names = append(names, namespace.Name)
	}
	return names, nil
}

// Pods lists pods in a namespace.
func (c *Client) Pods(ctx context.Context, namespace string) ([]Pod, error) {
	result, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods in namespace %s: %w", namespace, err)
	}
	pods := make([]Pod, 0, len(result.Items))
	for _, pod := range result.Items {
		pods = append(pods, summarizePod(pod))
	}
	return pods, nil
}

// Containers lists regular container names for a pod.
func (c *Client) Containers(ctx context.Context, namespace, pod string) ([]string, error) {
	result, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, pod, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get pod %s@%s: %w", pod, namespace, err)
	}
	containers := make([]string, 0, len(result.Spec.Containers))
	for _, container := range result.Spec.Containers {
		containers = append(containers, container.Name)
	}
	return containers, nil
}

// MatchingPods returns exact or substring matches for a pod query.
func MatchingPods(pods []Pod, query string) []Pod {
	for _, pod := range pods {
		if pod.Name == query {
			return []Pod{pod}
		}
	}
	matches := make([]Pod, 0)
	for _, pod := range pods {
		if strings.Contains(pod.Name, query) {
			matches = append(matches, pod)
		}
	}
	return matches
}

func summarizePod(pod corev1.Pod) Pod {
	ready := 0
	var restarts int32
	for _, status := range pod.Status.ContainerStatuses {
		if status.Ready {
			ready++
		}
		restarts += status.RestartCount
	}
	age := time.Duration(0)
	if !pod.CreationTimestamp.IsZero() {
		age = time.Since(pod.CreationTimestamp.Time).Round(time.Second)
	}
	return Pod{
		Name:       pod.Name,
		Phase:      string(pod.Status.Phase),
		Ready:      ready,
		Containers: len(pod.Spec.Containers),
		Restarts:   restarts,
		Age:        age,
	}
}
