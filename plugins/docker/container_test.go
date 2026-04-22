package docker

import (
	"strings"
	"testing"
)

// =============================================================================
// buildRunArgs 纯函数测试
// =============================================================================

func TestBuildRunArgs_BasicImageOnly(t *testing.T) {
	opts := &RunOptions{Image: "nginx:latest"}
	args := buildRunArgs(opts)

	expected := []string{"run", "nginx:latest"}
	if len(args) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, args)
	}
	for i, v := range expected {
		if args[i] != v {
			t.Errorf("args[%d] = %q, want %q", i, args[i], v)
		}
	}
}

func TestBuildRunArgs_DetachMode(t *testing.T) {
	opts := &RunOptions{Image: "alpine", Detach: true}
	args := buildRunArgs(opts)

	if !sliceContains(args, "-d") {
		t.Error("expected -d flag in args")
	}
}

func TestBuildRunArgs_InteractiveTTY(t *testing.T) {
	opts := &RunOptions{Image: "alpine", Interactive: true, TTY: true, Remove: true}
	args := buildRunArgs(opts)

	if !sliceContains(args, "-i") {
		t.Error("expected -i flag")
	}
	if !sliceContains(args, "-t") {
		t.Error("expected -t flag")
	}
	if !sliceContains(args, "--rm") {
		t.Error("expected --rm flag")
	}
}

func TestBuildRunArgs_ContainerName(t *testing.T) {
	opts := &RunOptions{Image: "nginx", Name: "my-web"}
	args := buildRunArgs(opts)

	idx := sliceIndexOf(args, "--name")
	if idx < 0 {
		t.Fatal("expected --name flag")
	}
	if args[idx+1] != "my-web" {
		t.Errorf("name = %q, want %q", args[idx+1], "my-web")
	}
}

func TestBuildRunArgs_Hostname(t *testing.T) {
	opts := &RunOptions{Image: "alpine", Hostname: "myhost"}
	args := buildRunArgs(opts)

	idx := sliceIndexOf(args, "-h")
	if idx < 0 {
		t.Fatal("expected -h flag")
	}
	if args[idx+1] != "myhost" {
		t.Errorf("hostname = %q, want %q", args[idx+1], "myhost")
	}
}

func TestBuildRunArgs_User(t *testing.T) {
	opts := &RunOptions{Image: "alpine", User: "root"}
	args := buildRunArgs(opts)

	idx := sliceIndexOf(args, "-u")
	if idx < 0 {
		t.Fatal("expected -u flag")
	}
	if args[idx+1] != "root" {
		t.Errorf("user = %q, want %q", args[idx+1], "root")
	}
}

func TestBuildRunArgs_Workdir(t *testing.T) {
	opts := &RunOptions{Image: "alpine", Workdir: "/app"}
	args := buildRunArgs(opts)

	idx := sliceIndexOf(args, "-w")
	if idx < 0 {
		t.Fatal("expected -w flag")
	}
	if args[idx+1] != "/app" {
		t.Errorf("workdir = %q, want %q", args[idx+1], "/app")
	}
}

func TestBuildRunArgs_Network(t *testing.T) {
	opts := &RunOptions{Image: "nginx", Network: "my-net"}
	args := buildRunArgs(opts)

	idx := sliceIndexOf(args, "--network")
	if idx < 0 {
		t.Fatal("expected --network flag")
	}
	if args[idx+1] != "my-net" {
		t.Errorf("network = %q, want %q", args[idx+1], "my-net")
	}
}

func TestBuildRunArgs_DNS(t *testing.T) {
	opts := &RunOptions{Image: "alpine", DNS: []string{"8.8.8.8", "8.8.4.4"}}
	args := buildRunArgs(opts)

	count := countFlag(args, "--dns")
	if count != 2 {
		t.Errorf("expected 2 --dns flags, got %d", count)
	}
}

func TestBuildRunArgs_ExtraHosts(t *testing.T) {
	opts := &RunOptions{Image: "alpine", ExtraHosts: []string{"host1:192.168.1.1"}}
	args := buildRunArgs(opts)

	idx := sliceIndexOf(args, "--add-host")
	if idx < 0 {
		t.Fatal("expected --add-host flag")
	}
	if args[idx+1] != "host1:192.168.1.1" {
		t.Errorf("add-host = %q, want %q", args[idx+1], "host1:192.168.1.1")
	}
}

func TestBuildRunArgs_RestartPolicy(t *testing.T) {
	opts := &RunOptions{Image: "nginx", Restart: "always"}
	args := buildRunArgs(opts)

	idx := sliceIndexOf(args, "--restart")
	if idx < 0 {
		t.Fatal("expected --restart flag")
	}
	if args[idx+1] != "always" {
		t.Errorf("restart = %q, want %q", args[idx+1], "always")
	}
}

func TestBuildRunArgs_Memory(t *testing.T) {
	opts := &RunOptions{Image: "nginx", Memory: "512m"}
	args := buildRunArgs(opts)

	idx := sliceIndexOf(args, "-m")
	if idx < 0 {
		t.Fatal("expected -m flag")
	}
	if args[idx+1] != "512m" {
		t.Errorf("memory = %q, want %q", args[idx+1], "512m")
	}
}

func TestBuildRunArgs_CPUs(t *testing.T) {
	opts := &RunOptions{Image: "nginx", CPUs: "1.5"}
	args := buildRunArgs(opts)

	idx := sliceIndexOf(args, "--cpus")
	if idx < 0 {
		t.Fatal("expected --cpus flag")
	}
	if args[idx+1] != "1.5" {
		t.Errorf("cpus = %q, want %q", args[idx+1], "1.5")
	}
}

func TestBuildRunArgs_Env(t *testing.T) {
	opts := &RunOptions{
		Image: "mysql",
		Env:   []string{"MYSQL_ROOT_PASSWORD=123456", "MYSQL_DATABASE=testdb"},
	}
	args := buildRunArgs(opts)

	count := countFlag(args, "-e")
	if count != 2 {
		t.Errorf("expected 2 -e flags, got %d", count)
	}

	// Verify specific values
	values := getFlagValues(args, "-e")
	if !sliceContains(values, "MYSQL_ROOT_PASSWORD=123456") {
		t.Error("expected MYSQL_ROOT_PASSWORD env var")
	}
	if !sliceContains(values, "MYSQL_DATABASE=testdb") {
		t.Error("expected MYSQL_DATABASE env var")
	}
}

func TestBuildRunArgs_EnvFile(t *testing.T) {
	opts := &RunOptions{Image: "app", EnvFile: []string{".env", "prod.env"}}
	args := buildRunArgs(opts)

	count := countFlag(args, "--env-file")
	if count != 2 {
		t.Errorf("expected 2 --env-file flags, got %d", count)
	}
}

func TestBuildRunArgs_Label(t *testing.T) {
	opts := &RunOptions{Image: "nginx", Label: []string{"version=1.0", "env=prod"}}
	args := buildRunArgs(opts)

	count := countFlag(args, "--label")
	if count != 2 {
		t.Errorf("expected 2 --label flags, got %d", count)
	}
}

func TestBuildRunArgs_Publish(t *testing.T) {
	opts := &RunOptions{Image: "nginx", Publish: []string{"80:80", "443:443"}}
	args := buildRunArgs(opts)

	count := countFlag(args, "-p")
	if count != 2 {
		t.Errorf("expected 2 -p flags, got %d", count)
	}
}

func TestBuildRunArgs_Expose(t *testing.T) {
	opts := &RunOptions{Image: "nginx", Expose: []string{"8080", "9090"}}
	args := buildRunArgs(opts)

	count := countFlag(args, "--expose")
	if count != 2 {
		t.Errorf("expected 2 --expose flags, got %d", count)
	}
}

func TestBuildRunArgs_Volume(t *testing.T) {
	opts := &RunOptions{
		Image:  "nginx",
		Volume: []string{"/host/data:/container/data", "/host/config:/etc/nginx:ro"},
	}
	args := buildRunArgs(opts)

	count := countFlag(args, "-v")
	if count != 2 {
		t.Errorf("expected 2 -v flags, got %d", count)
	}
}

func TestBuildRunArgs_Privileged(t *testing.T) {
	opts := &RunOptions{Image: "alpine", Privileged: true}
	args := buildRunArgs(opts)

	if !sliceContains(args, "--privileged") {
		t.Error("expected --privileged flag")
	}
}

func TestBuildRunArgs_Init(t *testing.T) {
	opts := &RunOptions{Image: "alpine", Init: true}
	args := buildRunArgs(opts)

	if !sliceContains(args, "--init") {
		t.Error("expected --init flag")
	}
}

func TestBuildRunArgs_Command(t *testing.T) {
	opts := &RunOptions{Image: "alpine", Command: []string{"sh", "-c", "echo hello"}}
	args := buildRunArgs(opts)

	// Last args should be the command
	idx := sliceIndexOf(args, "alpine")
	if idx < 0 {
		t.Fatal("expected image name in args")
	}
	rem := args[idx+1:]
	if len(rem) != 3 {
		t.Fatalf("expected 3 command args, got %d: %v", len(rem), rem)
	}
	if rem[0] != "sh" || rem[1] != "-c" || rem[2] != "echo hello" {
		t.Errorf("command = %v, want [sh -c echo hello]", rem)
	}
}

// Full integration test: all options combined
func TestBuildRunArgs_AllOptions(t *testing.T) {
	opts := &RunOptions{
		Image:       "nginx:latest",
		Command:     []string{"nginx", "-g", "daemon off;"},
		Name:        "web-server",
		Detach:      true,
		Interactive: false,
		TTY:         false,
		Remove:      false,
		Env:         []string{"FOO=bar"},
		Publish:     []string{"80:80"},
		Expose:      []string{"443"},
		Volume:      []string{"/data:/data"},
		Network:     "bridge",
		Restart:     "always",
		Memory:      "1g",
		CPUs:        "2",
		User:        "nginx",
		Workdir:     "/app",
		Hostname:    "web",
		Privileged:  false,
		Init:        true,
		EnvFile:     []string{".env"},
		Label:       []string{"app=web"},
		DNS:         []string{"8.8.8.8"},
		ExtraHosts:  []string{"db:10.0.0.1"},
	}
	args := buildRunArgs(opts)

	// Verify key elements present
	argStr := strings.Join(args, " ")

	checks := []string{
		"run",
		"-d",
		"--init",
		"--name web-server",
		"-h web",
		"-u nginx",
		"-w /app",
		"--network bridge",
		"--dns 8.8.8.8",
		"--add-host db:10.0.0.1",
		"--restart always",
		"-m 1g",
		"--cpus 2",
		"-e FOO=bar",
		"--env-file .env",
		"--label app=web",
		"-p 80:80",
		"--expose 443",
		"-v /data:/data",
		"nginx:latest",
		"nginx -g daemon off;",
	}

	for _, check := range checks {
		if !strings.Contains(argStr, check) {
			t.Errorf("expected args to contain %q, got: %s", check, argStr)
		}
	}

	// Verify order: "run" first, image before command
	runIdx := sliceIndexOf(args, "run")
	imageIdx := sliceIndexOf(args, "nginx:latest")
	if runIdx != 0 {
		t.Errorf("expected 'run' at index 0, got %d", runIdx)
	}
	if imageIdx <= 0 {
		t.Error("expected image to appear after flags")
	}
	// Command should come after image
	cmdStart := imageIdx + 1
	if args[cmdStart] != "nginx" {
		t.Errorf("expected command to start after image, got %q", args[cmdStart])
	}
}

// Test that empty strings are properly omitted
func TestBuildRunArgs_EmptyStrings(t *testing.T) {
	opts := &RunOptions{
		Image:    "alpine",
		Name:     "",
		Network:  "",
		Restart:  "",
		Memory:   "",
		CPUs:     "",
		User:     "",
		Workdir:  "",
		Hostname: "",
	}
	args := buildRunArgs(opts)

	// Should only have "run" and "alpine"
	if len(args) != 2 {
		t.Errorf("expected 2 args for empty options, got %d: %v", len(args), args)
	}
}

// Test that nil slices are properly handled
func TestBuildRunArgs_NilSlices(t *testing.T) {
	opts := &RunOptions{
		Image:   "alpine",
		Env:     nil,
		Publish: nil,
		Volume:  nil,
		DNS:     nil,
	}
	args := buildRunArgs(opts)

	if len(args) != 2 {
		t.Errorf("expected 2 args with nil slices, got %d: %v", len(args), args)
	}
}

// =============================================================================
// Run 验证测试（不依赖 Docker）
// =============================================================================

func TestRun_NilOptions(t *testing.T) {
	err := Run(nil)
	if err == nil {
		t.Error("expected error for nil options")
	}
	if err.Error() != "请指定要运行的镜像" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRun_EmptyImage(t *testing.T) {
	err := Run(&RunOptions{Image: ""})
	if err == nil {
		t.Error("expected error for empty image")
	}
	if err.Error() != "请指定要运行的镜像" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// =============================================================================
// 辅助函数
// =============================================================================

func sliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func sliceIndexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

func countFlag(slice []string, flag string) int {
	count := 0
	for _, s := range slice {
		if s == flag {
			count++
		}
	}
	return count
}

func getFlagValues(slice []string, flag string) []string {
	var values []string
	for i, s := range slice {
		if s == flag && i+1 < len(slice) {
			values = append(values, slice[i+1])
		}
	}
	return values
}
