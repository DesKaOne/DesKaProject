package p2p

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"deskachain/internal/chain"
	"deskachain/internal/config"
)

func testHandshakeForIdentity(t *testing.T, identity NodeIdentity, profile config.NetworkConfig) Handshake {
	t.Helper()
	profile.GenesisHash = chain.GenesisBlockForNetwork(profile).Hash
	hs := Handshake{
		NetworkName:         profile.NetworkName,
		NetworkID:           profile.NetworkID,
		ChainID:             profile.ChainID,
		ProtocolVersion:     profile.ProtocolVersion,
		P2PProtocolVersion:  profile.P2PProtocolVersion,
		MinProtocolVersion:  profile.MinProtocolVersion,
		GenesisHash:         profile.GenesisHash,
		Height:              12,
		TipHash:             "tip-hash",
		CumulativeWork:      12345,
		NodeID:              identity.NodeID,
		IdentityVersion:     NodeIdentityVersion,
		NodePublicKey:       hex.EncodeToString(identity.PublicKey),
		P2PListen:            "127.0.0.1:9331",
		P2PAdvertise:         "127.0.0.1:9331",
	}
	signature, err := SignHandshake(identity, hs)
	if err != nil {
		t.Fatal(err)
	}
	hs.NodeSignature = signature
	return hs
}

func TestLoadOrCreateNodeIdentityPersistsKeyMaterial(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "node_id")

	first, err := LoadOrCreateNodeIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := LoadOrCreateNodeIdentity(path)
	if err != nil {
		t.Fatal(err)
	}

	if first.NodeID != second.NodeID {
		t.Fatal("node id changed between identity loads")
	}
	if !bytes.Equal(first.PrivateKey, second.PrivateKey) {
		t.Fatal("private key changed between identity loads")
	}
	if !bytes.Equal(first.PublicKey, second.PublicKey) {
		t.Fatal("public key changed between identity loads")
	}

	info, err := os.Stat(path + ".ed25519")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatalf("node identity key is too permissive: %o", info.Mode().Perm())
	}
}

func TestAuthenticatedHandshakeRoundTrip(t *testing.T) {
	profile := config.Localnet()
	identity, err := LoadOrCreateNodeIdentity(filepath.Join(t.TempDir(), "node_id"))
	if err != nil {
		t.Fatal(err)
	}
	hs := testHandshakeForIdentity(t, identity, profile)

	if err := VerifyHandshakeIdentity(hs); err != nil {
		t.Fatalf("valid identity rejected: %v", err)
	}
	if err := ValidateHandshake(profile, hs); err != nil {
		t.Fatalf("valid authenticated handshake rejected: %v", err)
	}
}

func TestAuthenticatedHandshakeRejectsTampering(t *testing.T) {
	profile := config.Localnet()
	identity, err := LoadOrCreateNodeIdentity(filepath.Join(t.TempDir(), "node_id"))
	if err != nil {
		t.Fatal(err)
	}
	hs := testHandshakeForIdentity(t, identity, profile)

	tampered := hs
	tampered.Height++
	if err := VerifyHandshakeIdentity(tampered); err == nil || !strings.Contains(err.Error(), "invalid node identity signature") {
		t.Fatalf("expected signature rejection after height tamper, got %v", err)
	}

	tampered = hs
	tampered.NodeID = "attacker"
	if err := VerifyHandshakeIdentity(tampered); err == nil || !strings.Contains(err.Error(), "invalid node identity signature") {
		t.Fatalf("expected signature rejection after node id tamper, got %v", err)
	}

	tampered = hs
	tampered.NodePublicKey = strings.Repeat("00", ed25519.PublicKeySize)
	if err := VerifyHandshakeIdentity(tampered); err == nil || !strings.Contains(err.Error(), "invalid node identity signature") {
		t.Fatalf("expected signature rejection after public key tamper, got %v", err)
	}
}

func TestValidateHandshakeRequiresAuthenticatedNodeWhenConfigured(t *testing.T) {
	profile := config.Localnet()
	profile.RequireAuthenticatedNode = true
	peer := Handshake{
		NetworkID:        profile.NetworkID,
		ChainID:          profile.ChainID,
		GenesisHash:      chain.GenesisBlockForNetwork(profile).Hash,
		ProtocolVersion:  profile.ProtocolVersion,
		MinProtocolVersion: profile.MinProtocolVersion,
	}
	if err := ValidateHandshake(profile, peer); err == nil || !strings.Contains(err.Error(), "authenticated node identity required") {
		t.Fatalf("expected authentication requirement, got %v", err)
	}
}

func TestValidateHandshakeKeepsUnsignedCompatibilityWhenOptional(t *testing.T) {
	profile := config.Localnet()
	peer := Handshake{
		NetworkID:          profile.NetworkID,
		ChainID:            profile.ChainID,
		GenesisHash:        chain.GenesisBlockForNetwork(profile).Hash,
		ProtocolVersion:    profile.ProtocolVersion,
		MinProtocolVersion: profile.MinProtocolVersion,
	}
	if err := ValidateHandshake(profile, peer); err != nil {
		t.Fatalf("optional unsigned handshake rejected: %v", err)
	}
}
