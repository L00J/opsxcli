package sshconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParse_Basic(t *testing.T) {
	content := `
Host prod
    HostName 192.168.1.10
    User admin
    Port 2222
    IdentityFile ~/.ssh/prod_key
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(cfg.Hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(cfg.Hosts))
	}

	h := cfg.Hosts[0]
	if h.Alias != "prod" {
		t.Errorf("Alias = %q, want prod", h.Alias)
	}
	if h.HostName != "192.168.1.10" {
		t.Errorf("HostName = %q, want 192.168.1.10", h.HostName)
	}
	if h.User != "admin" {
		t.Errorf("User = %q, want admin", h.User)
	}
	if h.Port != 2222 {
		t.Errorf("Port = %d, want 2222", h.Port)
	}
	if h.IdentityFile == "" {
		t.Error("IdentityFile should not be empty")
	}
}

func TestParse_MultipleAliases(t *testing.T) {
	content := `
Host prod web1
    HostName 10.0.0.1
    User deploy
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(cfg.Hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(cfg.Hosts))
	}

	aliases := []string{cfg.Hosts[0].Alias, cfg.Hosts[1].Alias}
	expected := []string{"prod", "web1"}
	if !reflect.DeepEqual(aliases, expected) {
		t.Errorf("aliases = %v, want %v", aliases, expected)
	}

	for _, h := range cfg.Hosts {
		if h.HostName != "10.0.0.1" {
			t.Errorf("HostName = %q, want 10.0.0.1", h.HostName)
		}
		if h.User != "deploy" {
			t.Errorf("User = %q, want deploy", h.User)
		}
	}
}

func TestParse_WildcardIgnored(t *testing.T) {
	content := `
Host *
    User root
    Port 22

Host specific
    HostName 1.2.3.4
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(cfg.Hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(cfg.Hosts))
	}
	if cfg.Hosts[0].Alias != "specific" {
		t.Errorf("Alias = %q, want specific", cfg.Hosts[0].Alias)
	}
}

func TestParse_CommentsAndEmptyLines(t *testing.T) {
	content := `
# This is a comment
Host dev

    HostName dev.local
    User ubuntu

# Another comment
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(cfg.Hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(cfg.Hosts))
	}
	if cfg.Hosts[0].Alias != "dev" {
		t.Errorf("Alias = %q, want dev", cfg.Hosts[0].Alias)
	}
}

func TestParse_Indentation(t *testing.T) {
	content := "Host test\n\tHostName test.example.com\n\tUser testuser\n\tPort 2022\n"
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(cfg.Hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(cfg.Hosts))
	}
	h := cfg.Hosts[0]
	if h.HostName != "test.example.com" {
		t.Errorf("HostName = %q, want test.example.com", h.HostName)
	}
	if h.User != "testuser" {
		t.Errorf("User = %q, want testuser", h.User)
	}
	if h.Port != 2022 {
		t.Errorf("Port = %d, want 2022", h.Port)
	}
}

func TestParse_Include(t *testing.T) {
	tmpDir := t.TempDir()

	// main config
	mainContent := `
Host main
    HostName main.example.com

Include ` + tmpDir + `/included_config
`
	mainPath := filepath.Join(tmpDir, "config")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// included config
	includedContent := `
Host included
    HostName included.example.com
    User includeduser
`
	includedPath := filepath.Join(tmpDir, "included_config")
	if err := os.WriteFile(includedPath, []byte(includedContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Parse(mainPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(cfg.Hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(cfg.Hosts))
	}

	aliases := make(map[string]bool)
	for _, h := range cfg.Hosts {
		aliases[h.Alias] = true
	}
	if !aliases["main"] || !aliases["included"] {
		t.Errorf("expected aliases main and included, got %v", aliases)
	}
}

func TestParse_FileNotExists(t *testing.T) {
	cfg, err := Parse("/nonexistent/path/config")
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if len(cfg.Hosts) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(cfg.Hosts))
	}
}

func TestConfig_GetHost(t *testing.T) {
	cfg := &Config{
		Hosts: []Host{
			{Alias: "prod", HostName: "1.1.1.1", User: "admin", Port: 22},
		},
	}

	h := cfg.GetHost("prod")
	if h == nil {
		t.Fatal("expected to find prod")
	}
	if h.HostName != "1.1.1.1" {
		t.Errorf("HostName = %q, want 1.1.1.1", h.HostName)
	}

	h = cfg.GetHost("missing")
	if h != nil {
		t.Error("expected nil for missing host")
	}
}

func TestConfig_GetHost_FallbackAlias(t *testing.T) {
	cfg := &Config{
		Hosts: []Host{
			{Alias: "prod", HostName: "", User: "admin", Port: 22},
		},
	}

	h := cfg.GetHost("prod")
	if h == nil {
		t.Fatal("expected to find prod")
	}
	if h.HostName != "prod" {
		t.Errorf("HostName fallback = %q, want prod", h.HostName)
	}
}

func TestConfig_ListAliases(t *testing.T) {
	cfg := &Config{
		Hosts: []Host{
			{Alias: "a"},
			{Alias: "b"},
			{Alias: "c"},
		},
	}

	got := cfg.ListAliases()
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ListAliases = %v, want %v", got, want)
	}
}

func TestDefaultPath(t *testing.T) {
	path := DefaultPath()
	if path == "" {
		t.Error("DefaultPath should not be empty")
	}
	if !filepath.IsAbs(path) {
		t.Error("DefaultPath should be absolute")
	}
}
