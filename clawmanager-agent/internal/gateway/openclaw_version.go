package gateway

import (
	"runtime"
	"strconv"
	"strings"
)

const OpenClaw81Version = "2026.8.1"

var openClaw81Capabilities = []string{
	"openclaw.state.sqlite",
	"openclaw.automation.rpc",
	"openclaw.backup.sqlite",
	"openclaw.database.preflight",
	"openclaw.plugin.capability-consent",
	"openclaw.gateway.stop-confirm",
	"openclaw.workspace.writer-lease",
	"openclaw.session-sqlite-migrate-v1",
	"openclaw.session-sqlite-restore-v1",
	"openclaw.session-continuity-v1",
	"openclaw.runtime-standby-v1",
	"openclaw.upgrade-capsule-v2",
	"openclaw.upgrade-capsule-v3",
	"openclaw.session-sqlite-preserve-v1",
	"openclaw.upgrade-preflight-v3",
	"openclaw.upgrade-preflight-v4",
	"redis-team.group-hooks-v1",
}

func IsOpenClawAtLeast(version string, required string) bool {
	current := numericVersion(version)
	want := numericVersion(required)
	for index := 0; index < len(current) || index < len(want); index++ {
		var left, right int
		if index < len(current) {
			left = current[index]
		}
		if index < len(want) {
			right = want[index]
		}
		if left != right {
			return left > right
		}
	}
	return len(current) > 0
}

func OpenClawCapabilities(version string) []string {
	if !IsOpenClawAtLeast(version, OpenClaw81Version) {
		return nil
	}
	capabilities := append([]string(nil), openClaw81Capabilities...)
	if runtime.GOOS == "linux" {
		capabilities = append(capabilities, "openclaw.upgrade-operation-fence-v1", "openclaw.rebuild-preserve-v1", "openclaw.rebuild-preserve-v2")
	}
	if runtime.GOOS == "linux" && version == "2026.10.2" {
		capabilities = append(capabilities, "openclaw.managed-plugin-convergence-v1")
	}
	return capabilities
}

func numericVersion(raw string) []int {
	raw = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(raw), "v"))
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == '.' || r == '-' || r == '+' })
	values := make([]int, 0, 3)
	for _, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil {
			break
		}
		values = append(values, value)
		if len(values) == 3 {
			break
		}
	}
	return values
}
