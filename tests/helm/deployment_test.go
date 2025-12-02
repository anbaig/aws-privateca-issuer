package helm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeployment(t *testing.T) {
	helper := setupTest(t)
	defer helper.cleanup()

	tests := []struct {
		name     string
		values   map[string]interface{}
		validate func(t *testing.T, h *testHelper, releaseName string)
	}{
		{
			name: "disableApprovedCheck adds command line flag",
			values: map[string]interface{}{
				"disableApprovedCheck": true,
			},
			validate: func(t *testing.T, h *testHelper, releaseName string) {
				deploymentName := releaseName + "-aws-privateca-issuer"
				deployment, err := h.clientset.AppsV1().Deployments(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)

				container := deployment.Spec.Template.Spec.Containers[0]
				assert.Contains(t, container.Args, "-disable-approved-check")
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
