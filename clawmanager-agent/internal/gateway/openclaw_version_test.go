package gateway

import (
	"runtime"
	"testing"
)

func TestOpenClawVersionCapabilities(t *testing.T) {
	if IsOpenClawAtLeast("2026.7.1-2", OpenClaw81Version) {
		t.Fatal("2026.7.1-2 must remain on the legacy compatibility path")
	}
	for _, version := range []string{"2026.8.1", "v2026.8.1", "2026.9.0", "2026.10.2"} {
		if !IsOpenClawAtLeast(version, OpenClaw81Version) {
			t.Fatalf("%s must use the 8.1 compatibility path", version)
		}
		capabilities := OpenClawCapabilities(version)
		want := len(openClaw81Capabilities)
		platformCapabilities := []string{"openclaw.upgrade-operation-fence-v1", "openclaw.rebuild-preserve-v1", "openclaw.rebuild-preserve-v2"}
		if runtime.GOOS == "linux" {
			want += len(platformCapabilities)
		}
		if runtime.GOOS == "linux" && version == "2026.10.2" {
			want++
		}
		if got := len(capabilities); got != want {
			t.Fatalf("capability count for %s on %s = %d, want %d", version, runtime.GOOS, got, want)
		}
		seen := make(map[string]bool, len(capabilities))
		for _, capability := range capabilities {
			if seen[capability] {
				t.Fatalf("duplicate capability %s for %s", capability, version)
			}
			seen[capability] = true
		}
		for _, capability := range platformCapabilities {
			if seen[capability] != (runtime.GOOS == "linux") {
				t.Fatalf("capability %s for %s on %s has incorrect platform scope", capability, version, runtime.GOOS)
			}
		}
		if seen["openclaw.managed-plugin-convergence-v1"] != (runtime.GOOS == "linux" && version == "2026.10.2") {
			t.Fatalf("managed plugin convergence must be restricted to the verified Linux 10.2 runtime")
		}
	}
	if got := OpenClawCapabilities("2026.7.1-2"); got != nil {
		t.Fatalf("legacy capabilities = %#v, want nil", got)
	}
}

func TestOpenClawUpgradeCapabilitiesUseCapsuleNotWorkspaceSnapshot(t *testing.T) {
	capabilities := OpenClawCapabilities(OpenClaw81Version)
	for _, forbidden := range []string{"openclaw.workspace.snapshot-v1", "openclaw.workspace.atomic-restore"} {
		for _, capability := range capabilities {
			if capability == forbidden {
				t.Fatalf("deprecated full-workspace capability %s is still advertised", forbidden)
			}
		}
	}
	for _, required := range []string{"openclaw.session-sqlite-migrate-v1", "openclaw.session-sqlite-restore-v1", "openclaw.session-sqlite-preserve-v1", "openclaw.session-continuity-v1", "openclaw.runtime-standby-v1", "openclaw.upgrade-capsule-v3", "openclaw.upgrade-preflight-v4"} {
		found := false
		for _, capability := range capabilities {
			found = found || capability == required
		}
		if !found {
			t.Fatalf("missing upgrade capability %s", required)
		}
	}
}
