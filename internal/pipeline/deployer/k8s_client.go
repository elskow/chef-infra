package deployer

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// K8sClient interface abstracts kubernetes client operations
type K8sClient interface {
	CreateDeployment(ctx context.Context, namespace string, deployment *appsv1.Deployment) (*appsv1.Deployment, error)
	UpdateDeployment(ctx context.Context, namespace string, deployment *appsv1.Deployment) (*appsv1.Deployment, error)
	GetDeployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error)
	CreateService(ctx context.Context, namespace string, service *corev1.Service) (*corev1.Service, error)
	UpdateService(ctx context.Context, namespace string, service *corev1.Service) (*corev1.Service, error)
	GetService(ctx context.Context, namespace, name string) (*corev1.Service, error)
	CreateIngress(ctx context.Context, namespace string, ingress *networkingv1.Ingress) (*networkingv1.Ingress, error)
	UpdateIngress(ctx context.Context, namespace string, ingress *networkingv1.Ingress) (*networkingv1.Ingress, error)
	GetIngress(ctx context.Context, namespace, name string) (*networkingv1.Ingress, error)
	ListReplicaSets(ctx context.Context, namespace string, opts metav1.ListOptions) (*appsv1.ReplicaSetList, error)
}

type RealK8sClient struct {
	clientset kubernetes.Interface
}

func NewRealK8sClient(clientset kubernetes.Interface) *RealK8sClient {
	return &RealK8sClient{clientset: clientset}
}

func (c *RealK8sClient) CreateDeployment(ctx context.Context, namespace string, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
}

func (c *RealK8sClient) UpdateDeployment(ctx context.Context, namespace string, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
}

func (c *RealK8sClient) GetDeployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *RealK8sClient) CreateService(ctx context.Context, namespace string, service *corev1.Service) (*corev1.Service, error) {
	return c.clientset.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
}

func (c *RealK8sClient) CreateIngress(ctx context.Context, namespace string, ingress *networkingv1.Ingress) (*networkingv1.Ingress, error) {
	return c.clientset.NetworkingV1().Ingresses(namespace).Create(ctx, ingress, metav1.CreateOptions{})
}

func (c *RealK8sClient) ListReplicaSets(ctx context.Context, namespace string, opts metav1.ListOptions) (*appsv1.ReplicaSetList, error) {
	return c.clientset.AppsV1().ReplicaSets(namespace).List(ctx, opts)
}

func (c *RealK8sClient) UpdateService(ctx context.Context, namespace string, service *corev1.Service) (*corev1.Service, error) {
	return c.clientset.CoreV1().Services(namespace).Update(ctx, service, metav1.UpdateOptions{})
}

func (c *RealK8sClient) GetService(ctx context.Context, namespace, name string) (*corev1.Service, error) {
	return c.clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *RealK8sClient) UpdateIngress(ctx context.Context, namespace string, ingress *networkingv1.Ingress) (*networkingv1.Ingress, error) {
	return c.clientset.NetworkingV1().Ingresses(namespace).Update(ctx, ingress, metav1.UpdateOptions{})
}

func (c *RealK8sClient) GetIngress(ctx context.Context, namespace, name string) (*networkingv1.Ingress, error) {
	return c.clientset.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
}
