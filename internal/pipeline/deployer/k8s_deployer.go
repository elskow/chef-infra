package deployer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/elskow/chef-infra/internal/pipeline/config"
	"github.com/elskow/chef-infra/internal/pipeline/types"
)

type K8sDeployer struct {
	config    *config.DeployConfig
	logger    *zap.Logger
	clientset *kubernetes.Clientset
}

func NewK8sDeployer(config *config.DeployConfig, logger *zap.Logger) (*K8sDeployer, error) {
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	return &K8sDeployer{
		config:    config,
		logger:    logger,
		clientset: clientset,
	}, nil
}

func (d *K8sDeployer) Deploy(ctx context.Context, build *types.Build) error {
	pathType := networkingv1.PathTypePrefix

	// Create or update deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      build.ProjectID,
			Namespace: d.config.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &[]int32{int32(d.config.ReplicaCount)}[0],
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": build.ProjectID,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": build.ProjectID,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  build.ProjectID,
							Image: build.ImageID,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	// Apply deployment
	if _, err := d.clientset.AppsV1().Deployments(d.config.Namespace).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			_, err = d.clientset.AppsV1().Deployments(d.config.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update deployment: %w", err)
			}
		} else {
			return fmt.Errorf("failed to create deployment: %w", err)
		}
	}

	// Create service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      build.ProjectID,
			Namespace: d.config.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": build.ProjectID,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt32(80),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	// Apply service
	if _, err := d.clientset.CoreV1().Services(d.config.Namespace).Create(ctx, service, metav1.CreateOptions{}); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create service: %w", err)
		}
	}

	// Create ingress
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      build.ProjectID,
			Namespace: d.config.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: fmt.Sprintf("%s.%s", build.ProjectID, d.config.IngressDomain),
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: build.ProjectID,
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Apply ingress
	if _, err := d.clientset.NetworkingV1().Ingresses(d.config.Namespace).Create(ctx, ingress, metav1.CreateOptions{}); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create ingress: %w", err)
		}
	}

	return nil
}

func (d *K8sDeployer) Rollback(_ context.Context, _ *types.Build) error {
	// TODO: Implement rollback logic using Kubernetes deployments rollback feature
	return nil
}

func (d *K8sDeployer) Validate(build *types.Build) error {
	if build.ImageID == "" {
		return fmt.Errorf("image ID is required for kubernetes deployment")
	}
	return nil
}
