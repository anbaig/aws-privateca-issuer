package helm

import (
	"github.com/cert-manager/aws-privateca-issuer/tests/helm/testutil"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeploymentConfiguration(t *testing.T) {
	helper := testutil.SetupTest(t)
	defer helper.Cleanup()

	tests := []struct {
		name     string
		values   map[string]interface{}
		validate func(t *testing.T, h *testutil.TestHelper, releaseName string)
	}{
		{
			name: "custom resources and security context",
			values: map[string]interface{}{
				"resources": map[string]interface{}{
					"limits": map[string]interface{}{
						"cpu":    "100m",
						"memory": "128Mi",
					},
					"requests": map[string]interface{}{
						"cpu":    "25m",
						"memory": "32Mi",
					},
				},
				"securityContext": map[string]interface{}{
					"allowPrivilegeEscalation": false,
					"runAsNonRoot":             true,
				},
				"podSecurityContext": map[string]interface{}{
					"runAsUser": 1000,
				},
				"replicaCount":         3,
				"revisionHistoryLimit": 5,
			},
			validate: func(t *testing.T, h *testutil.TestHelper, releaseName string) {
				deploymentName := releaseName + "-aws-privateca-issuer"
				deployment, err := h.Clientset.AppsV1().Deployments(h.Namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)

				// Check replica count
				assert.Equal(t, int32(3), *deployment.Spec.Replicas)

				// Check revision history limit
				assert.Equal(t, int32(5), *deployment.Spec.RevisionHistoryLimit)

				// Check resources
				container := deployment.Spec.Template.Spec.Containers[0]
				assert.Equal(t, resource.MustParse("100m"), container.Resources.Limits["cpu"])
				assert.Equal(t, resource.MustParse("128Mi"), container.Resources.Limits["memory"])
				assert.Equal(t, resource.MustParse("25m"), container.Resources.Requests["cpu"])
				assert.Equal(t, resource.MustParse("32Mi"), container.Resources.Requests["memory"])

				// Check security context
				assert.NotNil(t, container.SecurityContext)
				assert.False(t, *container.SecurityContext.AllowPrivilegeEscalation)

				// Check pod security context
				assert.NotNil(t, deployment.Spec.Template.Spec.SecurityContext)
				assert.Equal(t, int64(1000), *deployment.Spec.Template.Spec.SecurityContext.RunAsUser)
			},
		},
		{
			name: "disableClientSideRateLimiting adds command line flag",
			values: map[string]interface{}{
				"disableClientSideRateLimiting": true,
			},
			validate: func(t *testing.T, h *testutil.TestHelper, releaseName string) {
				deploymentName := releaseName + "-aws-privateca-issuer"
				deployment, err := h.Clientset.AppsV1().Deployments(h.Namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)

				container := deployment.Spec.Template.Spec.Containers[0]
				assert.Contains(t, container.Args, "-disable-client-side-rate-limiting")
			},
		},
		{
			name: "custom image configuration",
			values: map[string]interface{}{
				"image": map[string]interface{}{
					"repository": "custom.registry.com/aws-privateca-issuer",
					"tag":        "v1.0.0",
					"pullPolicy": "Always",
				},
			},
			validate: func(t *testing.T, h *testutil.TestHelper, releaseName string) {
				deploymentName := releaseName + "-aws-privateca-issuer"
				deployment, err := h.Clientset.AppsV1().Deployments(h.Namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)

				container := deployment.Spec.Template.Spec.Containers[0]
				assert.Equal(t, "custom.registry.com/aws-privateca-issuer:v1.0.0", container.Image)
				assert.Equal(t, "Always", string(container.ImagePullPolicy))
			},
		},
		{
			name: "pod annotations and labels",
			values: map[string]interface{}{
				"podAnnotations": map[string]interface{}{
					"prometheus.io/scrape": "true",
					"prometheus.io/port":   "8080",
				},
				"podLabels": map[string]interface{}{
					"environment": "test",
					"team":        "platform",
				},
			},
			validate: func(t *testing.T, h *testutil.TestHelper, releaseName string) {
				deploymentName := releaseName + "-aws-privateca-issuer"
				deployment, err := h.Clientset.AppsV1().Deployments(h.Namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)

				podTemplate := deployment.Spec.Template
				assert.Equal(t, "true", podTemplate.Annotations["prometheus.io/scrape"])
				assert.Equal(t, "8080", podTemplate.Annotations["prometheus.io/port"])
				assert.Equal(t, "test", podTemplate.Labels["environment"])
				assert.Equal(t, "platform", podTemplate.Labels["team"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release := helper.InstallChart(tt.values)
			if release == nil {
				t.Skip("Chart installation failed")
				return
			}
			defer helper.UninstallChart(release.Name)

			deploymentName := release.Name + "-aws-privateca-issuer"
			helper.WaitForDeployment(deploymentName)

			if !t.Failed() {
				tt.validate(t, helper, release.Name)
			}
		})
	}
}
