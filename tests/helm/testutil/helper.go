package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	LocalChartPath = "../../../charts/aws-pca-issuer"
	ProdChartRepo  = "https://cert-manager.github.io/aws-privateca-issuer"
	ReleasePrefix  = "test-release"
)

type TestMode int

const (
	LocalMode TestMode = iota
	BetaMode
	ProdMode
)

func GetTestMode() TestMode {
	switch os.Getenv("HELM_TEST_MODE") {
	case "beta":
		return BetaMode
	case "prod":
		return ProdMode
	default:
		return LocalMode
	}
}

type TestHelper struct {
	T         *testing.T
	Clientset kubernetes.Interface
	Namespace string
}

func SetupTest(t *testing.T) *TestHelper {
	// Use existing cluster setup from make target
	kubeconfig := "/tmp/pca_kubeconfig"
	t.Logf("Attempting to use kubeconfig: %s", kubeconfig)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		t.Logf("Failed to load kubeconfig: %v", err)
		t.Skipf("Skipping e2e test - no Kubernetes cluster available: %v", err)
	}

	t.Logf("Successfully loaded kubeconfig, server: %s", config.Host)

	clientset, err := kubernetes.NewForConfig(config)
	if !assert.NoError(t, err, "Failed to create Kubernetes clientset") {
		t.Skip("Cannot create Kubernetes clientset")
	}

	t.Logf("Successfully created Kubernetes clientset")

	// Create unique namespace for each test to avoid race conditions
	testNamespace := fmt.Sprintf("aws-pca-issuer-test-%d", time.Now().UnixNano())
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
		},
	}
	_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Logf("Failed to create namespace: %v", err)
		require.NoError(t, err)
	} else {
		t.Logf("Successfully created/verified namespace: %s", testNamespace)
	}

	return &TestHelper{
		T:         t,
		Clientset: clientset,
		Namespace: testNamespace,
	}
}

func (h *TestHelper) Cleanup() {
	// Clean up cluster-scoped resources first (they don't get deleted with namespace)
	h.cleanupClusterResources()

	// Then delete the namespace
	err := h.Clientset.CoreV1().Namespaces().Delete(context.TODO(), h.Namespace, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		h.T.Logf("Failed to cleanup namespace: %v", err)
	}
}

func (h *TestHelper) cleanupClusterResources() {
	// List all releases in this namespace to find cluster resources to clean up
	releases, err := h.Clientset.CoreV1().Secrets(h.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "owner=helm",
	})
	if err != nil {
		h.T.Logf("Failed to list Helm releases for cleanup: %v", err)
		return
	}

	for _, secret := range releases.Items {
		if strings.HasPrefix(secret.Name, "sh.helm.release.v1.") {
			// Extract release name from secret name
			parts := strings.Split(secret.Name, ".")
			if len(parts) >= 4 {
				releaseName := parts[3]
				h.cleanupClusterResourcesForRelease(releaseName)
			}
		}
	}
}

func (h *TestHelper) cleanupClusterResourcesForRelease(releaseName string) {
	// Clean up ClusterRole
	err := h.Clientset.RbacV1().ClusterRoles().Delete(context.TODO(), releaseName, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		h.T.Logf("Failed to cleanup ClusterRole %s: %v", releaseName, err)
	}

	// Clean up ClusterRoleBinding
	err = h.Clientset.RbacV1().ClusterRoleBindings().Delete(context.TODO(), releaseName, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		h.T.Logf("Failed to cleanup ClusterRoleBinding %s: %v", releaseName, err)
	}

	// Clean up approver ClusterRole and ClusterRoleBinding
	approverRoleName := "cert-manager-controller-approve:awspca-cert-manager-io"
	err = h.Clientset.RbacV1().ClusterRoles().Delete(context.TODO(), approverRoleName, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		h.T.Logf("Failed to cleanup approver ClusterRole %s: %v", approverRoleName, err)
	}

	err = h.Clientset.RbacV1().ClusterRoleBindings().Delete(context.TODO(), approverRoleName, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		h.T.Logf("Failed to cleanup approver ClusterRoleBinding %s: %v", approverRoleName, err)
	}
}

// GitHubRelease represents the structure of GitHub API release response
type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

// ValidateChartVersion validates that the installed chart version matches the latest GitHub release
func (h *TestHelper) ValidateChartVersion(release *release.Release) error {
	if GetTestMode() != ProdMode {
		h.T.Logf("Skipping version validation - not in production mode")
		return nil
	}

	h.T.Logf("Starting chart version validation for production mode")

	// Get latest release version from GitHub
	latestVersion, err := h.getLatestGitHubReleaseVersion()
	if err != nil {
		return fmt.Errorf("failed to fetch latest GitHub release version: %w", err)
	}

	// Get installed chart version
	installedVersion := release.Chart.Metadata.Version

	h.T.Logf("Latest GitHub release version: %s", latestVersion)
	h.T.Logf("Installed chart version: %s", installedVersion)

	// Compare versions (normalize by removing 'v' prefix if present)
	normalizedLatest := strings.TrimPrefix(latestVersion, "v")
	normalizedInstalled := strings.TrimPrefix(installedVersion, "v")

	if normalizedLatest != normalizedInstalled {
		return fmt.Errorf("version mismatch: installed chart version %s does not match latest GitHub release version %s", installedVersion, latestVersion)
	}

	h.T.Logf("Chart version validation passed: %s matches %s", installedVersion, latestVersion)
	return nil
}

// getLatestGitHubReleaseVersion fetches the latest release version from GitHub API
func (h *TestHelper) getLatestGitHubReleaseVersion() (string, error) {
	url := "https://api.github.com/repos/cert-manager/aws-privateca-issuer/releases/latest"

	h.T.Logf("Fetching latest release from: %s", url)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode GitHub API response: %w", err)
	}

	if release.TagName == "" {
		return "", fmt.Errorf("no tag_name found in GitHub API response")
	}

	h.T.Logf("Successfully fetched latest release version: %s", release.TagName)
	return release.TagName, nil
}
