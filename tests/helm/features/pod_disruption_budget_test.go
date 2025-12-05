package helm

import (
	"context"
	"testing"

	"github.com/cert-manager/aws-privateca-issuer/tests/helm/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestPodDisruptionBudget(t *testing.T) {
	helper := testutil.SetupTest(t)
	defer helper.Cleanup()

	tests := []struct {
		name     string
		values   map[string]interface{}
		validate func(t *testing.T, h *testutil.TestHelper, releaseName string)
	}{
		{
			name: "podDisruptionBudget with maxUnavailable",
			values: map[string]interface{}{
				"podDisruptionBudget": map[string]interface{}{
					"maxUnavailable": 1,
				},
			},
			validate: func(t *testing.T, h *testutil.TestHelper, releaseName string) {
				pdbName := releaseName + "-aws-privateca-issuer"
				pdb, err := h.Clientset.PolicyV1().PodDisruptionBudgets(h.Namespace).Get(context.TODO(), pdbName, metav1.GetOptions{})
				require.NoError(t, err)

				expectedMaxUnavailable := intstr.FromInt(1)
				assert.Equal(t, &expectedMaxUnavailable, pdb.Spec.MaxUnavailable)
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
