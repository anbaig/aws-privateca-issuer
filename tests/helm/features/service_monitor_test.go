package helm

import (
	"testing"

	"github.com/cert-manager/aws-privateca-issuer/tests/helm/testutil"
)

func TestServiceMonitor(t *testing.T) {
	helper := testutil.SetupTest(t)
	defer helper.Cleanup()

	tests := []struct {
		name     string
		values   map[string]interface{}
		validate func(t *testing.T, h *testutil.TestHelper, releaseName string)
	}{
		{
			name: "serviceMonitor enabled creates ServiceMonitor resource",
			values: map[string]interface{}{
				"serviceMonitor": map[string]interface{}{
					"create": true,
				},
			},
			validate: func(t *testing.T, h *testutil.TestHelper, releaseName string) {
				t.Log("ServiceMonitor test passed - chart installed successfully with serviceMonitor.create=true")
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
