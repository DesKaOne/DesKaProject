package wallet

import (
	"os"
	"strings"
	"testing"
)

func TestSecureStoreRoundTrip(t *testing.T) {
	path := t.TempDir() + "/wallets.enc"
	store := NewSecureStore(path, "correct horse battery staple")
	w, err := New()
	if err != nil { t.Fatal(err) }
	if err := store.Add(w); err != nil { t.Fatal(err) }
	raw, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }
	if strings.Contains(string(raw), w.PrivateKeyHex) { t.Fatal("private key is stored in plaintext") }
	loaded, ok, err := store.Find(w.Address)
	if err != nil { t.Fatal(err) }
	if !ok { t.Fatal("wallet was not found") }
	if loaded.PrivateKeyHex != w.PrivateKeyHex { t.Fatal("private key changed after encrypted round trip") }
}

func TestSecureStoreRejectsWrongPassword(t *testing.T) {
	path := t.TempDir() + "/wallets.enc"
	w, err := New()
	if err != nil { t.Fatal(err) }
	if err := NewSecureStore(path, "right").Add(w); err != nil { t.Fatal(err) }
	if _, err := NewSecureStore(path, "wrong").Load(); err == nil { t.Fatal("wrong password unexpectedly decrypted wallet store") }
}

func TestSecureStoreRequiresPassword(t *testing.T) {
	if _, err := marshalEncryptedWallets(nil, ""); err == nil { t.Fatal("empty password unexpectedly accepted") }
}
