package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	// Capture stdout since the version command uses fmt.Printf
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := newRootCmd()
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		os.Stdout = oldStdout
		t.Fatalf("version command failed: %v", err)
	}

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = oldStdout

	output := buf.String()
	if !strings.Contains(output, "gar-cleanup") {
		t.Errorf("version output should contain 'gar-cleanup', got: %s", output)
	}
}

func TestRunCommand_MissingProject(t *testing.T) {
	// Ensure env var is not set
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")

	cmd := newRootCmd()
	cmd.SetArgs([]string{"run", "--project", "", "--policy", ""})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when project is empty (GAR client creation should fail)")
	}
}

func TestRunCommand_InvalidPolicyFile(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"run", "--project", "test-project", "--policy", "/nonexistent/policy.yaml"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent policy file")
	}
}

func TestRunCommand_ValidPolicyFile(t *testing.T) {
	content := `
keep: 5
max_age: 30d
protect_tags:
  - "^v\\d+\\.\\d+\\.\\d+$"
delete_untagged: true
dry_run: true
`
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"run", "--project", "test-project", "--policy", path})

	// This will succeed because GARClient stubs return nil repos.
	if err := cmd.Execute(); err != nil {
		t.Fatalf("run command failed: %v", err)
	}
}

func TestRootCommand_Help(t *testing.T) {
	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("help command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "gar-cleanup") {
		t.Errorf("help output should contain 'gar-cleanup', got: %s", output)
	}
}

func TestRunCommand_DefaultFlags(t *testing.T) {
	cmd := newRootCmd()
	runCmd, _, err := cmd.Find([]string{"run"})
	if err != nil {
		t.Fatalf("could not find 'run' subcommand: %v", err)
	}

	// Check default flag values
	policyFlag := runCmd.Flags().Lookup("policy")
	if policyFlag == nil {
		t.Fatal("--policy flag not found")
	}
	if policyFlag.DefValue != "policy.yaml" {
		t.Errorf("expected --policy default 'policy.yaml', got %q", policyFlag.DefValue)
	}

	liveFlag := runCmd.Flags().Lookup("live")
	if liveFlag == nil {
		t.Fatal("--live flag not found")
	}
	if liveFlag.DefValue != "false" {
		t.Errorf("expected --live default 'false', got %q", liveFlag.DefValue)
	}

	locationFlag := runCmd.Flags().Lookup("location")
	if locationFlag == nil {
		t.Fatal("--location flag not found")
	}
	if locationFlag.DefValue != "us-central1" {
		t.Errorf("expected --location default 'us-central1', got %q", locationFlag.DefValue)
	}
}
