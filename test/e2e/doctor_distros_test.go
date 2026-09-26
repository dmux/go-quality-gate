//go:build e2e

// Package e2e exercises the quality-gate binary against real Linux distro
// images via testcontainers-go, since runtime_detector.go's package-manager
// detection (apt-get/dnf/yum/apk/pacman) can only be verified by actually
// running on those distros — unit tests fake PATH lookups, but can't tell
// us the real apk/apt-get/dnf binaries behave the way we assume.
package e2e

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
)

var (
	linuxBinaryOnce sync.Once
	linuxBinaryPath string
	linuxBinaryErr  error
)

const (
	fixtureQualityYML = "tools: []\nhooks: {}\n"
	fixtureGoMod      = "module example.com/fixture\n\ngo 1.25\n"
	workDir           = "/work"
)

// buildLinuxBinary cross-compiles the quality-gate binary for linux/amd64
// once per test run and reuses it across every distro subtest.
func buildLinuxBinary(t *testing.T) string {
	t.Helper()

	linuxBinaryOnce.Do(func() {
		dir, err := os.MkdirTemp("", "quality-gate-e2e-*")
		if err != nil {
			linuxBinaryErr = err
			return
		}

		binaryPath := filepath.Join(dir, "quality-gate")
		cmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/quality-gate")
		cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			linuxBinaryErr = fmt.Errorf("failed to build linux binary: %w\n%s", err, out)
			return
		}
		linuxBinaryPath = binaryPath
	})

	if linuxBinaryErr != nil {
		t.Fatalf("buildLinuxBinary: %v", linuxBinaryErr)
	}
	return linuxBinaryPath
}

func TestDoctor_LinuxDistros_RecommendPackageManagerInstallCommand(t *testing.T) {
	binaryPath := buildLinuxBinary(t)

	cases := []struct {
		distro        string
		image         string
		wantSubstring string
	}{
		{"alpine", "alpine:3.20", "sudo apk add go"},
		{"debian", "debian:12-slim", "sudo apt-get install -y golang-go"},
		{"fedora", "fedora:40", "sudo dnf install -y golang"},
	}

	for _, tc := range cases {
		t.Run(tc.distro, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			req := testcontainers.ContainerRequest{
				Image: tc.image,
				Cmd:   []string{"sleep", "infinity"},
				Files: []testcontainers.ContainerFile{
					{HostFilePath: binaryPath, ContainerFilePath: "/usr/local/bin/quality-gate", FileMode: 0755},
				},
			}
			container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
				ContainerRequest: req,
				Started:          true,
			})
			if err != nil {
				t.Fatalf("failed to start %s container: %v", tc.image, err)
			}
			t.Cleanup(func() { _ = container.Terminate(ctx) })

			if _, _, err := container.Exec(ctx, []string{"mkdir", "-p", workDir}); err != nil {
				t.Fatalf("failed to create working directory in %s: %v", tc.image, err)
			}
			if err := container.CopyToContainer(ctx, []byte(fixtureQualityYML), workDir+"/quality.yml", 0644); err != nil {
				t.Fatalf("failed to copy quality.yml into %s: %v", tc.image, err)
			}
			if err := container.CopyToContainer(ctx, []byte(fixtureGoMod), workDir+"/go.mod", 0644); err != nil {
				t.Fatalf("failed to copy go.mod into %s: %v", tc.image, err)
			}

			exitCode, reader, err := container.Exec(ctx, []string{"quality-gate", "doctor"}, tcexec.WithWorkingDir(workDir))
			if err != nil {
				t.Fatalf("failed to exec doctor in %s: %v", tc.image, err)
			}

			output, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("failed to read doctor output: %v", err)
			}

			// doctor exits non-zero when a check fails (Go is intentionally
			// absent from every base image here), so a failing exit code is
			// expected, not an error.
			_ = exitCode

			if !strings.Contains(string(output), tc.wantSubstring) {
				t.Errorf("expected doctor output on %s to contain %q, got:\n%s", tc.image, tc.wantSubstring, output)
			}
		})
	}
}
