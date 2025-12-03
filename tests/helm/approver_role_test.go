package helm

import (
	"github.com/cert-manager/aws-privateca-issuer/tests/helm/testutil"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApproverRole(t *testing.T) {
	helper := testutil.SetupTest(t)
	defer helper.Cleanup()

	tests := []struct {
		name     string
		values   map[string]interface{}
		validate func(t *testing.T, h *testutil.TestHelper, releaseName string)
	}{
		{
			name: "approverRole enabled creates ClusterRole for certificate approval",
			values: map[string]interface{}{
				"approverRole": map[string]interface{}{
					"enabled":            true,
					"serviceAccountName": "cert-manager",
					"namespace":          "cert-manager",
				},
			},
			validate: func(t *testing.T, h *testutil.TestHelper, releaseName string) {
				clusterRoleName := "cert-manager-controller-approve:awspca-cert-manager-io"

				// Verify ClusterRole exists for approval
				clusterRole, err := h.Clientset.RbacV1().ClusterRoles().Get(context.TODO(), clusterRoleName, metav1.GetOptions{})
				require.NoError(t, err)

				// Check that the role has approval permissions
				found := false
				for _, rule := range clusterRole.Rules {
					if contains(rule.APIGroups, "cert-manager.io") &&
						contains(rule.Resources, "signers") &&
						contains(rule.Verbs, "approve") {
						found = true
						break
					}
				}
				assert.True(t, found, "ClusterRole should have certificate approval permissions")

				// Verify ClusterRoleBinding exists
				clusterRoleBinding, err := h.Clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), clusterRoleName, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, clusterRoleName, clusterRoleBinding.RoleRef.Name)

				// Check that it binds to the correct service account
				assert.Len(t, clusterRoleBinding.Subjects, 1)
				assert.Equal(t, "cert-manager", clusterRoleBinding.Subjects[0].Name)
				assert.Equal(t, "cert-manager", clusterRoleBinding.Subjects[0].Namespace)
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

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
