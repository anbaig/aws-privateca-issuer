package helm

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/cert-manager/aws-privateca-issuer/tests/helm/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaults(t *testing.T) {
	helper := testutil.SetupTest(t)
	defer helper.Cleanup()

	// Deploy chart with no custom values - validates all defaults
	release := helper.InstallChart(map[string]interface{}{})
	if release == nil {
		t.Skip("Chart installation failed")
		return
	}
	defer helper.UninstallChart(release.Name)

	deploymentName := release.Name + "-aws-privateca-issuer"
	helper.WaitForDeployment(deploymentName)

	// Validate default values
	deployment, err := helper.Clientset.AppsV1().Deployments(helper.Namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	require.NoError(t, err)

	// Default replica count
	assert.Equal(t, int32(2), *deployment.Spec.Replicas)

	// Default revision history limit
	assert.Equal(t, int32(10), *deployment.Spec.RevisionHistoryLimit)

	// Default image (varies by test mode)
	container := deployment.Spec.Template.Spec.Containers[0]
	mode := testutil.GetTestMode()
	if mode == testutil.BetaMode {
		privateRegistry := os.Getenv("PRIVATE_REGISTRY")
		require.NotEmpty(t, privateRegistry, "PRIVATE_REGISTRY environment variable is required for beta mode")
		repoName := os.Getenv("GITHUB_REPOSITORY")
		require.NotEmpty(t, repoName, "GITHUB_REPOSITORY environment variable is required for beta mode")
		expectedRepo := privateRegistry + "/" + strings.ToLower(strings.ReplaceAll(repoName, "/", "-")) + "-test"
		assert.Contains(t, container.Image, expectedRepo)
		assert.Equal(t, "Always", string(container.ImagePullPolicy))
	} else {
		assert.Contains(t, container.Image, "localhost:5000/aws-privateca-issuer")
		assert.Equal(t, "IfNotPresent", string(container.ImagePullPolicy))
	}

	// Default resources
	assert.Equal(t, resource.MustParse("50m"), container.Resources.Limits["cpu"])
	assert.Equal(t, resource.MustParse("64Mi"), container.Resources.Limits["memory"])
	assert.Equal(t, resource.MustParse("50m"), container.Resources.Requests["cpu"])
	assert.Equal(t, resource.MustParse("64Mi"), container.Resources.Requests["memory"])

	// Default security contexts
	assert.Equal(t, int64(65532), *deployment.Spec.Template.Spec.SecurityContext.RunAsUser)
	assert.False(t, *container.SecurityContext.AllowPrivilegeEscalation)

	// Validate service defaults
	serviceName := release.Name + "-aws-privateca-issuer"
	service, err := helper.Clientset.CoreV1().Services(helper.Namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "ClusterIP", string(service.Spec.Type))
	assert.Equal(t, int32(8080), service.Spec.Ports[0].Port)

	// Validate PDB defaults
	pdb, err := helper.Clientset.PolicyV1().PodDisruptionBudgets(helper.Namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, int32(1), pdb.Spec.MaxUnavailable.IntVal)

	// Validate affinity defaults
	affinity := deployment.Spec.Template.Spec.Affinity
	require.NotNil(t, affinity)

	// Check node affinity for OS/arch requirements
	nodeAffinity := affinity.NodeAffinity
	require.NotNil(t, nodeAffinity)
	assert.Contains(t, nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values, "linux")
	assert.Contains(t, nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[1].Values, "amd64")

	// Check pod anti-affinity for spreading
	podAntiAffinity := affinity.PodAntiAffinity
	require.NotNil(t, podAntiAffinity)
	assert.Equal(t, int32(100), podAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Weight)

	// Validate topology spread constraints defaults
	topologyConstraints := deployment.Spec.Template.Spec.TopologySpreadConstraints
	require.Len(t, topologyConstraints, 1)
	assert.Equal(t, int32(1), topologyConstraints[0].MaxSkew)
	assert.Equal(t, "topology.kubernetes.io/zone", topologyConstraints[0].TopologyKey)
}
