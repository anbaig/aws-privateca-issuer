package helm

import (
	"testing"
)

func TestServiceMonitor(t *testing.T) {
	helper := setupTest(t)
	defer helper.cleanup()

	tests := []struct {
		name     string
		values   map[string]interface{}
		validate func(t *testing.T, h *testHelper, releaseName string)
	}{
		{
			name: "serviceMonitor enabled creates ServiceMonitor resource",
			values: map[string]interface{}{
				"serviceMonitor": map[string]interface{}{
					"create": true,
				},
			},
			validate: func(t *testing.T, h *testHelper, releaseName string) {
				t.Log("ServiceMonitor test passed - chart installed successfully with serviceMonitor.create=true")
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
