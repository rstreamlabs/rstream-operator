// See LICENSE file in the project root for license information.

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func readReleaseFile(t *testing.T, path string) string {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(payload)
}

func runReleaseCommand(t *testing.T, directory string, name string, args ...string) string {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run %s: %v\n%s", name, err, output)
	}
	return string(output)
}

func TestReleaseCandidateRestoresFromDownloadDirectory(t *testing.T) {
	workingDirectory := t.TempDir()
	version := "1.2.3"
	candidateDirectory := filepath.Join(workingDirectory, "source", version)
	if err := os.MkdirAll(candidateDirectory, 0o755); err != nil {
		t.Fatalf("create candidate directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(candidateDirectory, "payload.txt"), []byte("candidate\n"), 0o644); err != nil {
		t.Fatalf("write candidate payload: %v", err)
	}
	manifest := runReleaseCommand(t, candidateDirectory, "shasum", "-a", "256", "payload.txt")
	if err := os.WriteFile(filepath.Join(candidateDirectory, "release-manifest.sha256"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write candidate manifest: %v", err)
	}
	downloadDirectory := filepath.Join(workingDirectory, "download")
	if err := os.MkdirAll(downloadDirectory, 0o755); err != nil {
		t.Fatalf("create download directory: %v", err)
	}
	archiveName := "release-candidate-" + version + ".tar.gz"
	archivePath := filepath.Join(downloadDirectory, archiveName)
	runReleaseCommand(t, workingDirectory, "tar", "-czf", archivePath, "-C", filepath.Join(workingDirectory, "source"), version)
	archiveChecksum := runReleaseCommand(t, downloadDirectory, "shasum", "-a", "256", archiveName)
	if err := os.WriteFile(archivePath+".sha256", []byte(archiveChecksum), 0o644); err != nil {
		t.Fatalf("write archive checksum: %v", err)
	}
	promotionDirectory := filepath.Join(workingDirectory, "promotion")
	if err := os.MkdirAll(promotionDirectory, 0o755); err != nil {
		t.Fatalf("create promotion directory: %v", err)
	}
	restoreScript, err := filepath.Abs(".github/scripts/restore-release-candidate.sh")
	if err != nil {
		t.Fatalf("resolve restore script: %v", err)
	}
	runReleaseCommand(t, promotionDirectory, restoreScript, version, archivePath)
	restoredPayload, err := os.ReadFile(filepath.Join(promotionDirectory, "out", "release-candidate", version, "payload.txt"))
	if err != nil {
		t.Fatalf("read restored payload: %v", err)
	}
	if string(restoredPayload) != "candidate\n" {
		t.Fatalf("unexpected restored payload: %q", restoredPayload)
	}
}

func TestReleaseCandidateDoesNotPublish(t *testing.T) {
	workflow := readReleaseFile(t, ".github/workflows/publish-helm.yml")
	for _, required := range []string{"name: Build release candidate", "actions/upload-artifact@"} {
		if !strings.Contains(workflow, required) {
			t.Errorf("candidate workflow is missing %q", required)
		}
	}
	for _, forbidden := range []string{"workflow_dispatch:", "docker/login-action@", "helm push"} {
		if strings.Contains(workflow, forbidden) {
			t.Errorf("candidate workflow contains publication behavior %q", forbidden)
		}
	}
}

func TestReleasePromotionIsApprovedAndFinalizedLast(t *testing.T) {
	workflow := readReleaseFile(t, ".github/workflows/promote-release.yml")
	for _, required := range []string{
		"workflow_dispatch:",
		"environment: stable-release",
		"Publish and verify operator image",
		"Publish and verify Helm chart",
		"Publish GitHub release",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("promotion workflow is missing %q", required)
		}
	}
	chart := strings.Index(workflow, "Publish and verify Helm chart")
	githubRelease := strings.Index(workflow, "Publish GitHub release")
	if chart == -1 || githubRelease <= chart {
		t.Fatal("GitHub release can become public before package verification")
	}
}

func TestReleasePleaseCreatesTaggedDraft(t *testing.T) {
	payload, err := os.ReadFile("release-please-config.json")
	if err != nil {
		t.Fatalf("read release-please config: %v", err)
	}
	var config struct {
		Packages map[string]struct {
			Draft            bool `json:"draft"`
			ForceTagCreation bool `json:"force-tag-creation"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(payload, &config); err != nil {
		t.Fatalf("parse release-please config: %v", err)
	}
	root := config.Packages["."]
	if !root.Draft || !root.ForceTagCreation {
		t.Fatalf("release-please must create a tagged draft: %#v", root)
	}
}
