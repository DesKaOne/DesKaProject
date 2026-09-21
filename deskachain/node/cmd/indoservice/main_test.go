package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionAndHelpDoNotRequireRPC(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"--version"}, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"indoservice", "version:", "commit:", "built:", "networks: localnet,testnet", "mainnet: not available"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("version output missing %q:\n%s", want, out.String())
		}
	}
	out.Reset()
	if err := run([]string{"--help"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "usage: indoservice") {
		t.Fatalf("help output missing usage:\n%s", out.String())
	}
}
