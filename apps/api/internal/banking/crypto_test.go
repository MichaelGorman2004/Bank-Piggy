package banking

import (
	"bytes"
	"strings"
	"testing"
)

func TestVaultAuthenticationRotationAndConfidentiality(t *testing.T) {
	config := testConfig()
	vault, err := NewVault(config)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := vault.Seal("simulated-secret", "household/a/access")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(ciphertext, "simulated-secret") {
		t.Fatal("plaintext persisted")
	}
	second, err := vault.Seal("simulated-secret", "household/a/access")
	if err != nil || second == ciphertext {
		t.Fatal("nonce was reused")
	}
	if _, err := vault.Open(ciphertext, "household/b/access"); err == nil {
		t.Fatal("foreign record decrypted")
	}
	parts := strings.SplitN(ciphertext, ".", 2)
	if _, err := vault.Open(parts[0]+"."+"A"+parts[1][1:], "household/a/access"); err == nil {
		t.Fatal("tampered ciphertext decrypted")
	}
}

func TestVaultKeyRotationAndRecovery(t *testing.T) {
	config := testConfig()
	vault, err := NewVault(config)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := vault.Seal("simulated-secret", "household/a/access")
	if err != nil {
		t.Fatal(err)
	}
	config.Keys["next"] = bytes.Repeat([]byte{1}, 32)
	config.ActiveKeyID = "next"
	rotated, err := NewVault(config)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := rotated.Open(ciphertext, "household/a/access")
	if err != nil || plain != "simulated-secret" {
		t.Fatal("retained key could not recover ciphertext")
	}
	fresh, err := rotated.Seal(plain, "household/a/access")
	if err != nil || !strings.HasPrefix(fresh, "next.") {
		t.Fatal("new writes did not rotate")
	}
	delete(config.Keys, "current")
	withoutOld, err := NewVault(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := withoutOld.Open(ciphertext, "household/a/access"); err == nil {
		t.Fatal("removed key remained usable")
	}
}

func TestConfigurationFailsClosed(t *testing.T) {
	cases := []Config{testConfig(), testConfig(), testConfig(), testConfig(), testConfig()}
	cases[0].Secret = ""
	cases[1].Keys = map[string][]byte{"current": make([]byte, 16)}
	cases[2].Environment = "development"
	cases[3].Environment = "production"
	cases[4].Environment = "production"
	cases[4].ProductionApproved = true
	cases[4].LifetimeItemLimit = 11
	for _, config := range cases {
		if err := config.Validate(); err == nil {
			t.Fatal("unsafe configuration accepted")
		}
	}
	valid := testConfig()
	valid.Environment = "production"
	valid.ProductionApproved = true
	valid.LifetimeItemLimit = 2
	valid.RedirectURI = "https://household.example/plaid-return"
	valid.WebhookURL = "https://household.example/api/plaid/webhook"
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := New(Config{}, nil, nil); err != nil {
		t.Fatal("disabled provider broke manual access")
	}
}

func TestProductionRequiresSameOriginHTTPSProviderURLs(t *testing.T) {
	config := testConfig()
	config.Environment = "production"
	config.ProductionApproved = true
	config.LifetimeItemLimit = 1
	invalid := [][2]string{{"http://127.0.0.1/plaid-return", "http://127.0.0.1/api/plaid/webhook"}, {"https://app.example/plaid-return", "https://other.example/api/plaid/webhook"}, {"https://app.example/plaid-return?query=1", "https://app.example/api/plaid/webhook"}}
	for _, pair := range invalid {
		config.RedirectURI, config.WebhookURL = pair[0], pair[1]
		if err := config.Validate(); err == nil {
			t.Fatal("unsafe live provider URLs accepted")
		}
	}
}
