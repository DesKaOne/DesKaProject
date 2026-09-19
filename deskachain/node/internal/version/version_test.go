package version

import (
	"strings"
	"testing"
)

func TestInfoFallbacks(t *testing.T) {
	oldVersion, oldCommit, oldBuildDate := Version, Commit, BuildDate
	defer func() {
		Version, Commit, BuildDate = oldVersion, oldCommit, oldBuildDate
	}()
	Version, Commit, BuildDate = "", "", ""
	info := Info("DesKaChain")
	if info.Version != "dev" || info.Commit != "unknown" || info.BuildDate != "unknown" {
		t.Fatalf("unexpected fallbacks: %#v", info)
	}
	if info.Networks != "localnet,testnet" || info.Mainnet != "not available" {
		t.Fatalf("unexpected network metadata: %#v", info)
	}
}

func TestStringContainsVersionFields(t *testing.T) {
	text := String("dkcminer")
	for _, want := range []string{"dkcminer", "version:", "commit:", "built:", "go:", "os/arch:", "networks: localnet,testnet", "mainnet: not available"} {
		if !strings.Contains(text, want) {
			t.Fatalf("version string missing %q:\n%s", want, text)
		}
	}
}
