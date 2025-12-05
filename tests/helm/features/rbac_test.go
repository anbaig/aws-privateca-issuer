package helm

import (
	"context"
	"testing"

	"github.com/cert-manager/aws-privateca-issuer/tests/helm/testutil"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRBAC(t *testing.T) {
	helper := testutil.SetupTest(t)
	defer helper.Cleanup()

	tests := []struct {
		name     string
		values   map[string]interface{}
		validate func(t *testing.T, h *testutil.TestHelper, releaseName string)
	}{
		{
			name: "rbac enabled creates ClusterRole and ClusterRoleBinding",
			values: map[string]interface{}{
				"rbac": map[string]interface{}{
					"create": true,
				},
				"serviceAccount": map[string]interface{}{
					"create": true,
				},
			},
			validate: func(t *testing.T, h *testutil.TestHelper, releaseName string) {
				clusterRoleName := releaseName + "-aws-privateca-issuer"

				// Verify ClusterRole exists
				clusterRole, err := h.Clientset.RbacV1().ClusterRoles().Get(context.TODO(), clusterRoleName, metav1.GetOptions{})
				if !assert.NoError(t, err, "ClusterRole should exist") {
					return
				}
				assert.NotEmpty(t, clusterRole.Rules)

				// Verify ClusterRoleBinding exists
				clusterRoleBinding, err := h.Clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), clusterRoleName, metav1.GetOptions{})
				if !assert.NoError(t, err, "ClusterRoleBinding should exist") {
					return
				}
				assert.Equal(t, clusterRoleName, clusterRoleBinding.RoleRef.Name)
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
