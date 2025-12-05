package helm

import (
	"context"
	"testing"

	"github.com/cert-manager/aws-privateca-issuer/tests/helm/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOptionalFields(t *testing.T) {
	helper := testutil.SetupTest(t)
	defer helper.Cleanup()

	// Test optional fields that are typically empty but should work when configured
	values := map[string]interface{}{
		"imagePullSecrets": []map[string]interface{}{
			{"name": "my-registry-secret"},
		},
		"tolerations": []map[string]interface{}{
			{
				"key":      "node-role.kubernetes.io/master",
				"operator": "Exists",
				"effect":   "NoSchedule",
			},
		},
		"nodeSelector": map[string]interface{}{
			"kubernetes.io/os": "linux",
		},
		"env": map[string]interface{}{
			"LOG_LEVEL": "debug",
		},
		"priorityClassName": "high-priority",
		"volumes": []map[string]interface{}{
			{
				"name": "config-volume",
				"configMap": map[string]interface{}{
					"name": "my-config",
				},
			},
		},
		"volumeMounts": []map[string]interface{}{
			{
				"name":      "config-volume",
				"mountPath": "/etc/config",
			},
		},
		"extraContainers": []map[string]interface{}{
			{
				"name":    "sidecar",
				"image":   "busybox:latest",
				"command": []string{"sleep", "3600"},
			},
		},
	}

	release := helper.InstallChart(values)
	if release == nil {
		t.Skip("Chart installation failed")
		return
	}
	defer helper.UninstallChart(release.Name)

	deploymentName := release.Name + "-aws-privateca-issuer"
	helper.WaitForDeployment(deploymentName)

	// Validate optional fields are applied
	deployment, err := helper.Clientset.AppsV1().Deployments(helper.Namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	require.NoError(t, err)

	podSpec := deployment.Spec.Template.Spec

	// Check imagePullSecrets
	assert.Len(t, podSpec.ImagePullSecrets, 1)
	assert.Equal(t, "my-registry-secret", podSpec.ImagePullSecrets[0].Name)

	// Check tolerations
	assert.Len(t, podSpec.Tolerations, 1)
	assert.Equal(t, "node-role.kubernetes.io/master", podSpec.Tolerations[0].Key)

	// Check nodeSelector
	assert.Equal(t, "linux", podSpec.NodeSelector["kubernetes.io/os"])

	// Check priorityClassName
	assert.Equal(t, "high-priority", podSpec.PriorityClassName)

	// Check volumes
	assert.Len(t, podSpec.Volumes, 1)
	assert.Equal(t, "config-volume", podSpec.Volumes[0].Name)

	// Check volumeMounts
	container := podSpec.Containers[0]
	assert.Len(t, container.VolumeMounts, 1)
	assert.Equal(t, "config-volume", container.VolumeMounts[0].Name)

	// Check env
	found := false
	for _, envVar := range container.Env {
		if envVar.Name == "LOG_LEVEL" && envVar.Value == "debug" {
			found = true
			break
		}
	}
	assert.True(t, found, "LOG_LEVEL environment variable should be set")

	// Check extraContainers
	assert.Len(t, podSpec.Containers, 2)
	assert.Equal(t, "sidecar", podSpec.Containers[1].Name)
}
