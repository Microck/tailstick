package tailscale

import "testing"

func TestBuildMachineContextIgnoresExecutableName(t *testing.T) {
	a := BuildMachineContext("Finance-Laptop", "/tmp/tailstick-linux-cli")
	b := BuildMachineContext("Finance-Laptop", "/var/lib/tailstick/tailstick-agent")
	if a != b {
		t.Fatalf("machine context should be stable across executable names: %q != %q", a, b)
	}
}
