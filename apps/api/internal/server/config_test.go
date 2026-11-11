package server

import "testing"

func TestConfigFailsClosed(t *testing.T) {
	tests := []Config{
		{Mode: "private", ListenAddr: "127.0.0.1:8080", BaseURL: testOrigin, DatabaseURL: "database"},
		{Mode: "demo", ListenAddr: "0.0.0.0:8080", BaseURL: testOrigin, DatabaseURL: "database"},
		{Mode: "demo", ListenAddr: "127.0.0.1:8080", BaseURL: "https://public.example", DatabaseURL: "database"},
		{Mode: "private", ListenAddr: "0.0.0.0:8080", BaseURL: "http://public.example", DatabaseURL: "database", GoogleClientID: "id", GoogleClientSecret: "secret", Invites: []string{"one@example.com", "two@example.com"}},
	}
	for index, config := range tests {
		if err := config.Validate(); err == nil {
			t.Fatalf("unsafe config %d accepted", index)
		}
	}
}

func TestSecureCookies(t *testing.T) {
	app := &Server{config: Config{BaseURL: "https://private.example"}}
	if !app.secureCookies() {
		t.Fatal("HTTPS session cookie was not secure")
	}
}
