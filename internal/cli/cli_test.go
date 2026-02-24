package cli

import "testing"

func TestParseArgs_Success(t *testing.T) {
	cfg, err := ParseArgs([]string{
		"--src-type", "User",
		"--src-path", "./src",
		"--dst-type", "UserDTO",
		"--dst-path", "./dst",
		"--filename", "user_gen.go",
		"--ignore-fields", "Password, Secret",
	})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if cfg.SrcType != "User" || cfg.DstType != "UserDTO" {
		t.Fatalf("unexpected types: %#v", cfg)
	}
	if len(cfg.IgnoreFields) != 2 {
		t.Fatalf("expected 2 ignore fields, got %d", len(cfg.IgnoreFields))
	}
}

func TestParseArgs_RequiresFields(t *testing.T) {
	_, err := ParseArgs([]string{
		"--src-type", "User",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
