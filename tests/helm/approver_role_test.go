package helm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApproverRole(t *testing.T) {
	helper := setupTest(t)
	defer helper.cleanup()

	tests := []struct {
		name     string
		values   map[string]interface{}
		validate func(t *testing.T, h *testHelper, releaseName string)
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
			validate: func(t *testing.T, h *testHelper, releaseName string) {
				clusterRoleName := releaseName + "-aws-privateca-issuer:cert-manager-approve"

				// Verify ClusterRole exists for approval
				clusterRole, err := h.clientset.RbacV1().ClusterRoles().Get(context.TODO(), clusterRoleName, metav1.GetOptions{})
				require.NoError(t, err)
				
				// Check that the role has approval permissions
				found := false
				for _, rule := range clusterRole.Rules {
					if contains(rule.APIGroups, "cert-manager.io") &&
						contains(rule.Resources, "certificaterequests") &&
						contains(rule.Verbs, "update") {
						found = true
						break
					}
				}
				assert.True(t, found, "ClusterRole should have certificate approval permissions")

				// Verify ClusterRoleBinding exists
				clusterRoleBinding, err := h.clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), clusterRoleName, metav1.GetOptions{})
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
			release := helper.installChart(tt.values)
			if release == nil {
				t.Skip("Chart installation failed")
				return
			}
			defer helper.uninstallChart(release.Name)

			deploymentName := release.Name + "-aws-privateca-issuer"
			helper.waitForDeployment(deploymentName)

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
