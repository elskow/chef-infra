package deployer

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/elskow/chef-infra/internal/pipeline/config"
	"github.com/elskow/chef-infra/internal/pipeline/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testCase struct {
	name        string
	build       *types.Build
	setupMocks  func(*K8sDeployer, *TestK8sClient)
	validate    func(*testing.T, *K8sDeployer, *TestK8sClient, error)
	expectError bool
}

func TestK8sDeployer_Deploy(t *testing.T) {
	tests := []testCase{
		{
			name: "successful deployment",
			build: &types.Build{
				ID:        "test-app-1",
				ProjectID: "test-app",
				ImageID:   "test-image:latest",
			},
			validate: func(t *testing.T, d *K8sDeployer, client *TestK8sClient, err error) {
				assert.NoError(t, err)

				deployment, err := client.GetDeployment(context.TODO(), "default", "test-app")
				require.NoError(t, err)
				assert.Equal(t, "test-app", deployment.Name)
				assert.Equal(t, "test-image:latest", deployment.Spec.Template.Spec.Containers[0].Image)

				// Verify service
				svc, err := client.GetService(context.TODO(), "default", "test-app")
				require.NoError(t, err)
				assert.Equal(t, "test-app", svc.Name)
				assert.Equal(t, int32(80), svc.Spec.Ports[0].Port)

				// Verify ingress
				ing, err := client.GetIngress(context.TODO(), "default", "test-app")
				require.NoError(t, err)
				assert.Equal(t, "test-app", ing.Name)
				assert.Equal(t, "test-app.test.local", ing.Spec.Rules[0].Host)
			},
		},
		{
			name: "update existing deployment",
			build: &types.Build{
				ID:        "test-app-2",
				ProjectID: "test-app",
				ImageID:   "test-image:v2",
			},
			setupMocks: func(d *K8sDeployer, client *TestK8sClient) {
				deployment := createTestDeployment("test-app", "test-image:v1")
				_, err := client.CreateDeployment(context.TODO(), "default", deployment)
				require.NoError(t, err)
			},
			validate: func(t *testing.T, d *K8sDeployer, client *TestK8sClient, err error) {
				assert.NoError(t, err)

				deployment, err := client.GetDeployment(context.TODO(), "default", "test-app")
				require.NoError(t, err)
				assert.Equal(t, "test-image:v2", deployment.Spec.Template.Spec.Containers[0].Image)
			},
		},
		{
			name: "invalid build config",
			build: &types.Build{
				ID:        "test-app-3",
				ProjectID: "test-app",
				ImageID:   "", // Invalid: empty image ID
			},
			expectError: true,
			validate: func(t *testing.T, d *K8sDeployer, client *TestK8sClient, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "image ID is required")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testClient := NewTestK8sClient()
			deployer := &K8sDeployer{
				config: &config.DeployConfig{
					Platform:      "kubernetes",
					Namespace:     "default",
					IngressDomain: "test.local",
					ReplicaCount:  1,
				},
				logger:    zap.NewNop(),
				k8sClient: testClient,
			}

			if tt.setupMocks != nil {
				tt.setupMocks(deployer, testClient)
			}

			// Validate build first
			if err := deployer.Validate(tt.build); err != nil {
				if !tt.expectError {
					t.Fatalf("unexpected validation error: %v", err)
				}
				return
			}

			// Execute deployment
			err := deployer.Deploy(context.TODO(), tt.build)

			// Run validations
			tt.validate(t, deployer, testClient, err)
		})
	}
}

func TestK8sDeployer_Rollback(t *testing.T) {
	tests := []testCase{
		{
			name: "successful rollback",
			build: &types.Build{
				ID:        "test-app-1",
				ProjectID: "test-app",
				ImageID:   "test-image:v2",
			},
			setupMocks: func(d *K8sDeployer, client *TestK8sClient) {
				// Create initial deployment with initialized annotations
				deployment := createTestDeployment("test-app", "test-image:v1")
				revisionLimit := int32(5)
				deployment.Spec.RevisionHistoryLimit = &revisionLimit
				deployment.Annotations = map[string]string{
					"kubernetes.io/change-cause": "Initial deployment",
				}
				_, err := client.CreateDeployment(context.TODO(), "default", deployment)
				require.NoError(t, err)

				// Create ReplicaSet for v1
				rs1 := createTestReplicaSet("test-app-v1", "test-image:v1", "1")
				_, err = client.GetClientset().AppsV1().ReplicaSets("default").Create(context.TODO(), rs1, metav1.CreateOptions{})
				require.NoError(t, err)

				// Update to v2
				deployment.Spec.Template.Spec.Containers[0].Image = "test-image:v2"
				deployment.Annotations["kubernetes.io/change-cause"] = "Updated to v2"
				_, err = client.UpdateDeployment(context.TODO(), "default", deployment)
				require.NoError(t, err)

				// Create ReplicaSet for v2
				rs2 := createTestReplicaSet("test-app-v2", "test-image:v2", "2")
				_, err = client.GetClientset().AppsV1().ReplicaSets("default").Create(context.TODO(), rs2, metav1.CreateOptions{})
				require.NoError(t, err)
			},
			validate: func(t *testing.T, d *K8sDeployer, client *TestK8sClient, err error) {
				assert.NoError(t, err)

				deployment, err := client.GetDeployment(context.TODO(), "default", "test-app")
				require.NoError(t, err)
				assert.Equal(t, "test-image:v1", deployment.Spec.Template.Spec.Containers[0].Image)
				assert.Contains(t, deployment.Annotations["kubernetes.io/change-cause"], "Rollback")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testClient := NewTestK8sClient()
			deployer := &K8sDeployer{
				config: &config.DeployConfig{
					Platform:      "kubernetes",
					Namespace:     "default",
					IngressDomain: "test.local",
					ReplicaCount:  1,
				},
				logger:    zap.NewNop(),
				k8sClient: testClient,
			}

			if tt.setupMocks != nil {
				tt.setupMocks(deployer, testClient)
			}

			err := deployer.Rollback(context.TODO(), tt.build)
			tt.validate(t, deployer, testClient, err)
		})
	}
}

func createTestDeployment(name, image string) *appsv1.Deployment {
	replicas := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: map[string]string{},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: image,
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
}

func createTestReplicaSet(name, image, revision string) *appsv1.ReplicaSet {
	replicas := int32(1)
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": strings.TrimSuffix(name, fmt.Sprintf("-v%s", revision)),
			},
			Annotations: map[string]string{
				"deployment.kubernetes.io/revision": revision,
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": strings.TrimSuffix(name, fmt.Sprintf("-v%s", revision)),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": strings.TrimSuffix(name, fmt.Sprintf("-v%s", revision)),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  strings.TrimSuffix(name, fmt.Sprintf("-v%s", revision)),
							Image: image,
						},
					},
				},
			},
		},
	}
}
