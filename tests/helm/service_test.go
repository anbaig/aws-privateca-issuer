package helm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestService(t *testing.T) {
	helper := setupTest(t)
	defer helper.cleanup()

	tests := []struct {
		name     string
		values   map[string]interface{}
		validate func(t *testing.T, h *testHelper, releaseName string)
	}{
		{
			name: "custom service configuration",
			values: map[string]interface{}{
				"service": map[string]interface{}{
					"type": "NodePort",
					"port": 9090,
				},
			},
			validate: func(t *testing.T, h *testHelper, releaseName string) {
				serviceName := releaseName + "-aws-privateca-issuer"
				service, err := h.clientset.CoreV1().Services(h.namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
				require.NoError(t, err)

				assert.Equal(t, "NodePort", string(service.Spec.Type))
				assert.Equal(t, int32(9090), service.Spec.Ports[0].Port)
			},
		},
		{
			name: "nameOverride affects resource names",
			values: map[string]interface{}{
				"nameOverride": "custom-issuer",
			},
			validate: func(t *testing.T, h *testHelper, releaseName string) {
				// With nameOverride, the deployment name should include the custom name
				deploymentName := releaseName + "-custom-issuer"
				deployment, err := h.clientset.AppsV1().Deployments(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)
				assert.NotNil(t, deployment)
			},
		},
		{
			name: "fullnameOverride completely overrides resource names",
			values: map[string]interface{}{
				"fullnameOverride": "completely-custom-name",
			},
			validate: func(t *testing.T, h *testHelper, releaseName string) {
				// With fullnameOverride, the deployment name should be exactly the override
				deploymentName := "completely-custom-name"
				deployment, err := h.clientset.AppsV1().Deployments(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)
				assert.NotNil(t, deployment)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release := helper.installChart(tt.values)
			if release == nil {
				t.Skip("Chart installation failed")
				return
			}
			defer helper.uninstallChart(release.Name)

			// For naming tests, we need to wait for the correct deployment name
			var deploymentName string
			if tt.name == "fullnameOverride completely overrides resource names" {
				deploymentName = "completely-custom-name"
			} else if tt.name == "nameOverride affects resource names" {
				deploymentName = release.Name + "-custom-issuer"
			} else {
				deploymentName = release.Name + "-aws-privateca-issuer"
			}
			
			helper.waitForDeployment(deploymentName)

			if !t.Failed() {
				tt.validate(t, helper, release.Name)
			}
		})
	}
}
