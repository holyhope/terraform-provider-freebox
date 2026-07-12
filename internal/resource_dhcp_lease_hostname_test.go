package internal

import (
	"testing"

	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

func TestNormalizeMAC(t *testing.T) {
	t.Parallel()

	if got := normalizeMAC("2e:7e:55:60:5a:66"); got != "2E:7E:55:60:5A:66" {
		t.Fatalf("normalizeMAC() = %q, want %q", got, "2E:7E:55:60:5A:66")
	}
}

func TestDhcpLeaseHostname(t *testing.T) {
	t.Parallel()

	lease := freeboxTypes.DHCPStaticLeaseInfo{
		Mac:      "2E:7E:55:60:5A:66",
		Hostname: "2E:7E:55:60:5A:66",
	}

	if got := dhcpLeaseHostname(lease, "vm-ubuntu-terraform"); got != "vm-ubuntu-terraform" {
		t.Fatalf("dhcpLeaseHostname() = %q, want configured hostname", got)
	}

	lease.Hostname = "real-hostname"
	if got := dhcpLeaseHostname(lease, "vm-ubuntu-terraform"); got != "real-hostname" {
		t.Fatalf("dhcpLeaseHostname() = %q, want API hostname", got)
	}

	if got := dhcpLeaseHostname(lease, ""); got != "real-hostname" {
		t.Fatalf("dhcpLeaseHostname() with empty fallback = %q, want API hostname", got)
	}
}

func TestDhcpLeaseModelPreservesMACCase(t *testing.T) {
	t.Parallel()

	var model dhcpLeaseModel
	model.fromDHCPStaticLeaseInfo(freeboxTypes.DHCPStaticLeaseInfo{
		ID:  "lease-id",
		Mac: "3E:AB:B7:EF:A2:50",
		IP:  "192.168.1.10",
	}, "", "3e:ab:b7:ef:a2:50")

	if got := model.Mac.ValueString(); got != "3e:ab:b7:ef:a2:50" {
		t.Fatalf("model.Mac = %q, want configured lowercase MAC", got)
	}
}
