package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run validate-image-version.go <stage> <version>")
		os.Exit(1)
	}

	stage := os.Args[1]
	expectedVersion := os.Args[2]

	var image string
	switch stage {
	case "local":
		image = "localhost:5000/aws-privateca-issuer:latest"
	case "beta":
		image = "public.ecr.aws/cert-manager-aws-privateca-issuer/cert-manager-aws-privateca-issuer-test:latest"
	case "prod":
		image = "public.ecr.aws/cert-manager-aws-privateca-issuer/cert-manager-aws-privateca-issuer-test:latest"
	default:
		fmt.Printf("Error: Invalid STAGE '%s'. Must be one of: local, beta, prod\n", stage)
		fmt.Println("Usage: go run validate-image-version.go <stage> <version>")
		os.Exit(1)
	}

	if expectedVersion == "" {
		fmt.Println("Error: VERSION cannot be empty")
		fmt.Println("Usage: go run validate-image-version.go <stage> <version>")
		os.Exit(1)
	}

	fmt.Printf("Stage: %s\n", stage)
	fmt.Printf("Expected Version: %s\n", expectedVersion)
	fmt.Printf("Image: %s\n", image)

	actualVersion := extractImageVersion(image)
	fmt.Printf("Detected Version: %s\n", actualVersion)

	if actualVersion == expectedVersion {
		fmt.Println("Status: ✓ MATCH")
	} else {
		fmt.Println("Status: ✗ MISMATCH")
		fmt.Printf("Error: Expected version %s but detected %s\n", expectedVersion, actualVersion)
		os.Exit(1)
	}
}

func extractImageVersion(image string) string {
	return extractEmbeddedVersion(image)
}

func extractEmbeddedVersion(image string) string {
	cmd := exec.Command("docker", "create", "--name", "temp-version", image)
	if err := cmd.Run(); err != nil {
		return "unknown"
	}
	defer exec.Command("docker", "rm", "temp-version").Run()

	cmd = exec.Command("docker", "cp", "temp-version:/manager", "/tmp/manager")
	if err := cmd.Run(); err != nil {
		return "unknown"
	}
	defer os.Remove("/tmp/manager")

	cmd = exec.Command("strings", "/tmp/manager")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	versionRegex := regexp.MustCompile(`PlugInVersion=(v\d+\.\d+\.\d+)`)
	for _, line := range strings.Split(string(output), "\n") {
		if match := versionRegex.FindStringSubmatch(line); len(match) > 1 {
			return match[1]
		}
	}

	return "unknown"
}
