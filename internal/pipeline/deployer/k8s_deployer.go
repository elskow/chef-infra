package deployer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/elskow/chef-infra/internal/pipeline/config"
	"github.com/elskow/chef-infra/internal/pipeline/types"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sDeployer struct {
	config    *config.DeployConfig
	logger    *zap.Logger
	k8sClient K8sClient
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
		k8sClient: NewRealK8sClient(clientset),
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
	_, err := d.k8sClient.CreateDeployment(ctx, d.config.Namespace, deployment)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			_, err = d.k8sClient.UpdateDeployment(ctx, d.config.Namespace, deployment)
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
	_, err = d.k8sClient.CreateService(ctx, d.config.Namespace, service)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			_, err = d.k8sClient.UpdateService(ctx, d.config.Namespace, service)
			if err != nil {
				return fmt.Errorf("failed to update service: %w", err)
			}
		} else {
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
	_, err = d.k8sClient.CreateIngress(ctx, d.config.Namespace, ingress)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			_, err = d.k8sClient.UpdateIngress(ctx, d.config.Namespace, ingress)
			if err != nil {
				return fmt.Errorf("failed to update ingress: %w", err)
			}
		} else {
			return fmt.Errorf("failed to create ingress: %w", err)
		}
	}

	return nil
}

func (d *K8sDeployer) Rollback(ctx context.Context, build *types.Build) error {
	d.logger.Info("rolling back deployment",
		zap.String("project", build.ProjectID))

	// Get the deployment
	deployment, err := d.k8sClient.GetDeployment(ctx, d.config.Namespace, build.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Initialize annotations map if nil
	if deployment.Annotations == nil {
		deployment.Annotations = make(map[string]string)
	}

	// Get deployment history
	revisions, err := d.k8sClient.ListReplicaSets(ctx, d.config.Namespace, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", build.ProjectID),
	})
	if err != nil {
		return fmt.Errorf("failed to get deployment history: %w", err)
	}

	if len(revisions.Items) <= 1 {
		return fmt.Errorf("no previous revision available for rollback")
	}

	// Sort ReplicaSets by revision number
	sort.Slice(revisions.Items, func(i, j int) bool {
		iRev, _ := strconv.Atoi(revisions.Items[i].Annotations["deployment.kubernetes.io/revision"])
		jRev, _ := strconv.Atoi(revisions.Items[j].Annotations["deployment.kubernetes.io/revision"])
		return iRev > jRev
	})

	// Get the previous revision (second most recent)
	previousRevision := &revisions.Items[1]

	// Update deployment with previous container specs
	deployment.Spec.Template.Spec.Containers = previousRevision.Spec.Template.Spec.Containers
	deployment.Annotations["kubernetes.io/change-cause"] = "Rollback triggered by Chef"

	// Apply the rollback
	_, err = d.k8sClient.UpdateDeployment(ctx, d.config.Namespace, deployment)
	if err != nil {
		return fmt.Errorf("failed to rollback deployment: %w", err)
	}

	return nil
}

func (d *K8sDeployer) Validate(build *types.Build) error {
	if build.ProjectID == "" {
		return fmt.Errorf("project ID is required for kubernetes deployment")
	}
	if build.ImageID == "" {
		return fmt.Errorf("image ID is required for kubernetes deployment")
	}
	if d.config.Namespace == "" {
		return fmt.Errorf("kubernetes namespace is not configured")
	}
	if d.config.IngressDomain == "" {
		return fmt.Errorf("ingress domain is not configured")
	}
	if d.config.ReplicaCount < 1 {
		return fmt.Errorf("replica count must be at least 1")
	}
	return nil
}
