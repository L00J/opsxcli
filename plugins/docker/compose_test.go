package docker

import (
	"strings"
	"testing"
)

// =============================================================================
// buildComposeUpArgs 纯函数测试
// =============================================================================

func TestBuildComposeUpArgs_Minimal(t *testing.T) {
	opts := &ComposeUpOptions{}
	args := buildComposeUpArgs(opts)

	if args[0] != "compose" {
		t.Errorf("first arg should be 'compose', got %q", args[0])
	}
	if args[1] != "up" {
		t.Errorf("second arg should be 'up', got %q", args[1])
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d: %v", len(args), args)
	}
}

func TestBuildComposeUpArgs_Detach(t *testing.T) {
	opts := &ComposeUpOptions{Detach: true}
	args := buildComposeUpArgs(opts)

	if !sliceContains(args, "-d") {
		t.Error("expected -d flag")
	}
}

func TestBuildComposeUpArgs_Build(t *testing.T) {
	opts := &ComposeUpOptions{Build: true}
	args := buildComposeUpArgs(opts)

	if !sliceContains(args, "--build") {
		t.Error("expected --build flag")
	}
}

func TestBuildComposeUpArgs_ForceRecreate(t *testing.T) {
	opts := &ComposeUpOptions{Force: true}
	args := buildComposeUpArgs(opts)

	if !sliceContains(args, "--force-recreate") {
		t.Error("expected --force-recreate flag")
	}
}

func TestBuildComposeUpArgs_FileAndProject(t *testing.T) {
	opts := &ComposeUpOptions{
		File:    "docker-compose.prod.yml",
		Project: "myapp",
	}
	args := buildComposeUpArgs(opts)

	argStr := strings.Join(args, " ")
	if !strings.Contains(argStr, "-f docker-compose.prod.yml") {
		t.Error("expected -f flag with file path")
	}
	if !strings.Contains(argStr, "-p myapp") {
		t.Error("expected -p flag with project name")
	}
}

func TestBuildComposeUpArgs_Services(t *testing.T) {
	opts := &ComposeUpOptions{Services: []string{"web", "db"}}
	args := buildComposeUpArgs(opts)

	// Services should be at the end
	lastTwo := args[len(args)-2:]
	if lastTwo[0] != "web" || lastTwo[1] != "db" {
		t.Errorf("expected last args to be [web db], got %v", lastTwo)
	}
}

func TestBuildComposeUpArgs_AllFlags(t *testing.T) {
	opts := &ComposeUpOptions{
		File:    "docker-compose.yml",
		Project: "test",
		Detach:  true,
		Build:   true,
		Force:   true,
		Remove:  true,
	}
	args := buildComposeUpArgs(opts)

	argStr := strings.Join(args, " ")
	checks := []string{"-f docker-compose.yml", "-p test", "up", "-d", "--build", "--force-recreate", "--remove-orphans"}
	for _, check := range checks {
		if !strings.Contains(argStr, check) {
			t.Errorf("expected args to contain %q, got: %s", check, argStr)
		}
	}
}

// =============================================================================
// buildComposeDownArgs 纯函数测试
// =============================================================================

func TestBuildComposeDownArgs_Minimal(t *testing.T) {
	opts := &ComposeDownOptions{}
	args := buildComposeDownArgs(opts)

	if args[0] != "compose" || args[1] != "down" {
		t.Errorf("expected [compose down], got %v", args[:2])
	}
}

func TestBuildComposeDownArgs_Volumes(t *testing.T) {
	opts := &ComposeDownOptions{Volumes: true}
	args := buildComposeDownArgs(opts)

	if !sliceContains(args, "--volumes") {
		t.Error("expected --volumes flag")
	}
}

func TestBuildComposeDownArgs_RemoveOrphans(t *testing.T) {
	opts := &ComposeDownOptions{RemoveOrphans: true}
	args := buildComposeDownArgs(opts)

	if !sliceContains(args, "--remove-orphans") {
		t.Error("expected --remove-orphans flag")
	}
}

func TestBuildComposeDownArgs_RMI(t *testing.T) {
	opts := &ComposeDownOptions{Images: "all"}
	args := buildComposeDownArgs(opts)

	idx := sliceIndexOf(args, "--rmi")
	if idx < 0 {
		t.Fatal("expected --rmi flag")
	}
	if args[idx+1] != "all" {
		t.Errorf("rmi = %q, want %q", args[idx+1], "all")
	}
}

func TestBuildComposeDownArgs_Timeout(t *testing.T) {
	opts := &ComposeDownOptions{Timeout: 30}
	args := buildComposeDownArgs(opts)

	idx := sliceIndexOf(args, "--timeout")
	if idx < 0 {
		t.Fatal("expected --timeout flag")
	}
	if args[idx+1] != "30" {
		t.Errorf("timeout = %q, want %q", args[idx+1], "30")
	}
}

func TestBuildComposeDownArgs_All(t *testing.T) {
	opts := &ComposeDownOptions{
		File:          "compose.yml",
		Project:       "myapp",
		RemoveOrphans: true,
		Volumes:       true,
		Images:        "local",
		Timeout:       60,
	}
	args := buildComposeDownArgs(opts)

	argStr := strings.Join(args, " ")
	checks := []string{"-f compose.yml", "-p myapp", "down", "--remove-orphans", "--volumes", "--rmi local", "--timeout 60"}
	for _, check := range checks {
		if !strings.Contains(argStr, check) {
			t.Errorf("expected args to contain %q, got: %s", check, argStr)
		}
	}
}

// =============================================================================
// appendComposeFile / appendComposeProject 测试
// =============================================================================

func TestAppendComposeFile_Empty(t *testing.T) {
	args := []string{"compose"}
	result := appendComposeFile(args, "")
	if len(result) != 1 {
		t.Errorf("expected 1 arg with empty file, got %d: %v", len(result), result)
	}
}

func TestAppendComposeFile_NonEmpty(t *testing.T) {
	args := []string{"compose"}
	result := appendComposeFile(args, "docker-compose.yml")
	if len(result) != 3 {
		t.Errorf("expected 3 args, got %d: %v", len(result), result)
	}
	if result[1] != "-f" || result[2] != "docker-compose.yml" {
		t.Errorf("expected [-f docker-compose.yml], got %v", result[1:])
	}
}

func TestAppendComposeProject_Empty(t *testing.T) {
	args := []string{"compose"}
	result := appendComposeProject(args, "")
	if len(result) != 1 {
		t.Errorf("expected 1 arg with empty project, got %d: %v", len(result), result)
	}
}

func TestAppendComposeProject_NonEmpty(t *testing.T) {
	args := []string{"compose"}
	result := appendComposeProject(args, "myproject")
	if len(result) != 3 {
		t.Errorf("expected 3 args, got %d: %v", len(result), result)
	}
	if result[1] != "-p" || result[2] != "myproject" {
		t.Errorf("expected [-p myproject], got %v", result[1:])
	}
}

// =============================================================================
// ComposeUp/ComposeDown 输入验证测试（不依赖 Docker）
// =============================================================================

func TestComposeUp_NilOptions(t *testing.T) {
	err := ComposeUp(nil)
	if err == nil {
		t.Error("expected error for nil options")
	}
}

func TestComposeDown_NilOptions(t *testing.T) {
	err := ComposeDown(nil)
	if err == nil {
		t.Error("expected error for nil options")
	}
}

func TestComposePS_NilOptions(t *testing.T) {
	err := ComposePS(nil)
	if err == nil {
		t.Error("expected error for nil options")
	}
}

func TestComposeLogs_NilOptions(t *testing.T) {
	err := ComposeLogs(nil)
	if err == nil {
		t.Error("expected error for nil options")
	}
}

func TestComposeBuild_NilOptions(t *testing.T) {
	err := ComposeBuild(nil)
	if err == nil {
		t.Error("expected error for nil options")
	}
}

func TestComposePull_NilOptions(t *testing.T) {
	err := ComposePull(nil)
	if err == nil {
		t.Error("expected error for nil options")
	}
}

func TestComposeRestart_NilOptions(t *testing.T) {
	err := ComposeRestart(nil)
	if err == nil {
		t.Error("expected error for nil options")
	}
}

func TestComposeStop_NilOptions(t *testing.T) {
	err := ComposeStop(nil)
	if err == nil {
		t.Error("expected error for nil options")
	}
}
