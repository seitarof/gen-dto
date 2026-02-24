package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seitarof/gen-dto/internal/generator"
	"github.com/seitarof/gen-dto/internal/matcher"
	"github.com/seitarof/gen-dto/internal/parser"
	"github.com/seitarof/gen-dto/internal/resolver"
)

func TestRunner_Run_GeneratesBidirectionalConverters(t *testing.T) {
	out := filepath.Join(t.TempDir(), "bidi_gen.go")

	runner := NewRunner(
		parser.New(),
		matcher.NewStructMatcher(),
		matcher.NewFieldMatcher(),
		resolver.New(resolver.DefaultRules()...),
		generator.New(generator.NewGoimportsFormatter(), generator.NewFileWriter()),
	)

	cfg := &Config{
		SrcType:  "User",
		SrcPath:  "github.com/seitarof/gen-dto/testdata/bidi/source",
		DstType:  "UserResponse",
		DstPath:  "github.com/seitarof/gen-dto/testdata/bidi/dest",
		Filename: out,
	}

	if err := runner.Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	content, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(content)

	checks := []string{
		"func ConvertUserToUserResponse",
		"func ConvertUserResponseToUser",
		"func ConvertSourceAddressToDestAddress",
		"func ConvertDestAddressToSourceAddress",
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Fatalf("generated code does not contain %q\n%s", check, got)
		}
	}
}

func TestRunner_Run_SupportsTypeAliasField(t *testing.T) {
	out := filepath.Join(t.TempDir(), "alias_gen.go")

	runner := NewRunner(
		parser.New(),
		matcher.NewStructMatcher(),
		matcher.NewFieldMatcher(),
		resolver.New(resolver.DefaultRules()...),
		generator.New(generator.NewGoimportsFormatter(), generator.NewFileWriter()),
	)

	cfg := &Config{
		SrcType:  "Patient",
		SrcPath:  "github.com/seitarof/gen-dto/testdata/aliassrc",
		DstType:  "PatientDTO",
		DstPath:  "github.com/seitarof/gen-dto/testdata/aliasdst",
		Filename: out,
	}

	if err := runner.Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	content, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(content)

	checks := []string{
		"func ConvertPatientToPatientDTO",
		"func ConvertPatientDTOToPatient",
		"dst.ProviderType = src.ProviderType",
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Fatalf("generated code does not contain %q\n%s", check, got)
		}
	}
}

func TestRunner_Run_SupportsNestedAliasStructFieldWithCast(t *testing.T) {
	out := filepath.Join(t.TempDir(), "alias_nested_gen.go")

	runner := NewRunner(
		parser.New(),
		matcher.NewStructMatcher(),
		matcher.NewFieldMatcher(),
		resolver.New(resolver.DefaultRules()...),
		generator.New(generator.NewGoimportsFormatter(), generator.NewFileWriter()),
	)

	cfg := &Config{
		SrcType:  "Patient",
		SrcPath:  "github.com/seitarof/gen-dto/testdata/aliasnested/src",
		DstType:  "PatientDTO",
		DstPath:  "github.com/seitarof/gen-dto/testdata/aliasnested/dst",
		Filename: out,
	}

	if err := runner.Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	content, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(content)

	checks := []string{
		"package src",
		"func ConvertPatientToPatientDTO",
		"dst.Provider = (dst.PatientProviderType)(src.Provider)",
		"func ConvertPatientDTOToPatient",
		"dst.Provider = (ProviderAlias)(src.Provider)",
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Fatalf("generated code does not contain %q\n%s", check, got)
		}
	}
}

func TestRunner_Run_RecursesNestedStructsAcrossPackagesInSameModule(t *testing.T) {
	out := filepath.Join(t.TempDir(), "crosspkg_gen.go")

	runner := NewRunner(
		parser.New(),
		matcher.NewStructMatcher(),
		matcher.NewFieldMatcher(),
		resolver.New(resolver.DefaultRules()...),
		generator.New(generator.NewGoimportsFormatter(), generator.NewFileWriter()),
	)

	cfg := &Config{
		SrcType:  "Notification",
		SrcPath:  "github.com/seitarof/gen-dto/testdata/crosspkg/srcroot",
		DstType:  "Notification",
		DstPath:  "github.com/seitarof/gen-dto/testdata/crosspkg/dstroot",
		Filename: out,
	}

	if err := runner.Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	content, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(content)

	checks := []string{
		"func ConvertSrcrootNotificationToDstrootNotification",
		"dst.Recipient = ConvertSrcnestedRecipientToDstnestedRecipient(src.Recipient)",
		"func ConvertSrcnestedRecipientToDstnestedRecipient",
		"dst.ID = src.ID",
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Fatalf("generated code does not contain %q\n%s", check, got)
		}
	}

	if strings.Contains(got, "skipped conversion Recipient") {
		t.Fatalf("recipient conversion should not be skipped\n%s", got)
	}
}
