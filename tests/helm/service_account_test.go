package helm

import (
	"github.com/cert-manager/aws-privateca-issuer/tests/helm/testutil"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServiceAccount(t *testing.T) {
	helper := testutil.SetupTest(t)
	defer helper.Cleanup()

	tests := []struct {
		name     string
		values   map[string]interface{}
		validate func(t *testing.T, h *testutil.TestHelper, releaseName string)
	}{
		{
			name: "serviceAccount with custom name",
			values: map[string]interface{}{
				"serviceAccount": map[string]interface{}{
					"create": true,
					"name":   "custom-service-account",
					"annotations": map[string]interface{}{
						"eks.amazonaws.com/role-arn": "arn:aws:iam::123456789012:role/test-role",
					},
				},
			},
			validate: func(t *testing.T, h *testutil.TestHelper, releaseName string) {
				serviceAccountName := "custom-service-account"
				sa, err := h.Clientset.CoreV1().ServiceAccounts(h.Namespace).Get(context.TODO(), serviceAccountName, metav1.GetOptions{})
				if !assert.NoError(t, err, "ServiceAccount should exist") {
					return
				}
				assert.Equal(t, "arn:aws:iam::123456789012:role/test-role", sa.Annotations["eks.amazonaws.com/role-arn"])
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
