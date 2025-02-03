package deployer

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type TestK8sClient struct {
	clientset *fake.Clientset
}

func NewTestK8sClient() *TestK8sClient {
	return &TestK8sClient{
		clientset: fake.NewSimpleClientset(),
	}
}

func (c *TestK8sClient) CreateDeployment(ctx context.Context, namespace string, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
}

func (c *TestK8sClient) UpdateDeployment(ctx context.Context, namespace string, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
}

func (c *TestK8sClient) GetDeployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *TestK8sClient) CreateService(ctx context.Context, namespace string, service *corev1.Service) (*corev1.Service, error) {
	return c.clientset.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
}

func (c *TestK8sClient) CreateIngress(ctx context.Context, namespace string, ingress *networkingv1.Ingress) (*networkingv1.Ingress, error) {
	return c.clientset.NetworkingV1().Ingresses(namespace).Create(ctx, ingress, metav1.CreateOptions{})
}

func (c *TestK8sClient) ListReplicaSets(ctx context.Context, namespace string, opts metav1.ListOptions) (*appsv1.ReplicaSetList, error) {
	return c.clientset.AppsV1().ReplicaSets(namespace).List(ctx, opts)
}

func (c *TestK8sClient) UpdateService(ctx context.Context, namespace string, service *corev1.Service) (*corev1.Service, error) {
	return c.clientset.CoreV1().Services(namespace).Update(ctx, service, metav1.UpdateOptions{})
}

func (c *TestK8sClient) GetService(ctx context.Context, namespace, name string) (*corev1.Service, error) {
	return c.clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *TestK8sClient) UpdateIngress(ctx context.Context, namespace string, ingress *networkingv1.Ingress) (*networkingv1.Ingress, error) {
	return c.clientset.NetworkingV1().Ingresses(namespace).Update(ctx, ingress, metav1.UpdateOptions{})
}

func (c *TestK8sClient) GetIngress(ctx context.Context, namespace, name string) (*networkingv1.Ingress, error) {
	return c.clientset.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *TestK8sClient) GetClientset() *fake.Clientset {
	return c.clientset
}
