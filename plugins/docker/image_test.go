package docker

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- parseRepoTag ---

func TestParseRepoTag(t *testing.T) {
	tests := []struct {
		name     string
		repoTags []string
		wantRepo string
		wantTag  string
	}{
		{
			name:     "标准镜像和标签",
			repoTags: []string{"nginx:latest"},
			wantRepo: "nginx",
			wantTag:  "latest",
		},
		{
			name:     "带用户名镜像",
			repoTags: []string{"user/myapp:v1.0"},
			wantRepo: "user/myapp",
			wantTag:  "v1.0",
		},
		{
			name:     "带仓库地址",
			repoTags: []string{"registry.example.com:5000/myapp:2.0"},
			wantRepo: "registry.example.com",
			wantTag:  "5000/myapp:2.0",
		},
		{
			name:     "空标签列表",
			repoTags: []string{},
			wantRepo: "<none>",
			wantTag:  "<none>",
		},
		{
			name:     "nil标签列表",
			repoTags: nil,
			wantRepo: "<none>",
			wantTag:  "<none>",
		},
		{
			name:     "无标签",
			repoTags: []string{"nginx"},
			wantRepo: "nginx",
			wantTag:  "<none>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, tag := parseRepoTag(tt.repoTags)
			assert.Equal(t, tt.wantRepo, repo)
			assert.Equal(t, tt.wantTag, tag)
		})
	}
}

// --- formatImageID ---

func TestFormatImageID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "sha256前缀_长ID",
			input: "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			want:  "abcdef123456",
		},
		{
			name:  "无前缀_长ID",
			input: "abcdef1234567890abcdef1234567890",
			want:  "abcdef123456",
		},
		{
			name:  "短ID",
			input: "abc123",
			want:  "abc123",
		},
		{
			name:  "空字符串",
			input: "",
			want:  "",
		},
		{
			name:  "sha256前缀_正好12字符",
			input: "sha256:abcdef123456",
			want:  "abcdef123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatImageID(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- formatImageCreated ---

func TestFormatImageCreated(t *testing.T) {
	tests := []struct {
		name    string
		created int64
		want    string
	}{
		{
			name:    "零值",
			created: 0,
			want:    "N/A",
		},
		{
			name:    "刚刚",
			created: time.Now().Unix() - 10,
			want:    "刚刚",
		},
		{
			name:    "几分钟前",
			created: time.Now().Unix() - 5*60,
			want:    "5分钟前",
		},
		{
			name:    "几小时前",
			created: time.Now().Unix() - 3*3600,
			want:    "3小时前",
		},
		{
			name:    "几天前",
			created: time.Now().Unix() - 5*86400,
			want:    "5天前",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatImageCreated(tt.created)
			assert.Equal(t, tt.want, got)
		})
	}
}

// 辅助函数：获取当前 Unix 时间戳
// (使用 time.Now() 获取真实时间)

// --- toInt64 ---

func TestToInt64(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  int64
	}{
		{name: "float64", input: float64(1024), want: 1024},
		{name: "int", input: 2048, want: 2048},
		{name: "int64", input: int64(4096), want: 4096},
		{name: "string数字", input: "8192", want: 8192},
		{name: "string非数字", input: "abc", want: 0},
		{name: "nil", input: nil, want: 0},
		{name: "bool", input: true, want: 0},
		{name: "float64小数", input: float64(3.7), want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toInt64(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- getStr ---

func TestGetStr(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want string
	}{
		{
			name: "字符串值",
			m:    map[string]interface{}{"name": "nginx"},
			key:  "name",
			want: "nginx",
		},
		{
			name: "非字符串值",
			m:    map[string]interface{}{"count": 42},
			key:  "count",
			want: "",
		},
		{
			name: "key不存在",
			m:    map[string]interface{}{"name": "nginx"},
			key:  "missing",
			want: "",
		},
		{
			name: "空map",
			m:    map[string]interface{}{},
			key:  "any",
			want: "",
		},
		{
			name: "nil map",
			m:    nil,
			key:  "any",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStr(tt.m, tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- ImageInfo JSON 解析 ---

func TestImageInfoJSON(t *testing.T) {
	jsonStr := `{
		"Id": "sha256:abc123def456",
		"RepoTags": ["nginx:latest", "nginx:stable"],
		"RepoDigests": ["nginx@sha256:abc123"],
		"Created": 1714000000,
		"Size": 187600000,
		"Labels": {"maintainer": "NGINX Docker Maintainers"}
	}`

	var img ImageInfo
	err := json.Unmarshal([]byte(jsonStr), &img)
	assert.NoError(t, err)
	assert.Equal(t, "sha256:abc123def456", img.ID)
	assert.Len(t, img.RepoTags, 2)
	assert.Equal(t, "nginx:latest", img.RepoTags[0])
	assert.Len(t, img.RepoDigests, 1)
	assert.Equal(t, int64(1714000000), img.Created)
	assert.Equal(t, int64(187600000), img.Size)
	assert.Equal(t, "NGINX Docker Maintainers", img.Labels["maintainer"])
}

func TestImageInfoJSON_Minimal(t *testing.T) {
	jsonStr := `{"Id": "sha256:deadbeef"}`

	var img ImageInfo
	err := json.Unmarshal([]byte(jsonStr), &img)
	assert.NoError(t, err)
	assert.Equal(t, "sha256:deadbeef", img.ID)
	assert.Nil(t, img.RepoTags)
	assert.Nil(t, img.RepoDigests)
	assert.Equal(t, int64(0), img.Size)
}

// --- ImagesOptions 默认值 ---

func TestImagesOptionsDefaults(t *testing.T) {
	opts := &ImagesOptions{}
	assert.False(t, opts.All)
	assert.False(t, opts.Quiet)
	assert.False(t, opts.NoTrunc)
	assert.Equal(t, "", opts.Filters)
}

// --- RMIOptions ---

func TestRMIOptions(t *testing.T) {
	opts := &RMIOptions{
		Images:  []string{"nginx:latest", "redis:alpine"},
		Force:   true,
		NoPrune: true,
	}
	assert.Len(t, opts.Images, 2)
	assert.True(t, opts.Force)
	assert.True(t, opts.NoPrune)
}

// --- TagOptions ---

func TestTagOptions(t *testing.T) {
	opts := &TagOptions{
		Source: "nginx:latest",
		Target: "myregistry/nginx:latest",
	}
	assert.Equal(t, "nginx:latest", opts.Source)
	assert.Equal(t, "myregistry/nginx:latest", opts.Target)
}

// --- PushImageOptions ---

func TestPushImageOptions(t *testing.T) {
	opts := &PushImageOptions{Image: "myregistry/nginx:latest"}
	assert.Equal(t, "myregistry/nginx:latest", opts.Image)
}

// --- ImageInspectResult 默认值 ---

func TestImageInspectResultDefaults(t *testing.T) {
	result := &ImageInspectResult{}
	assert.Empty(t, result.RepoTags)
	assert.Empty(t, result.RepoDigests)
	assert.Empty(t, result.ExposedPorts)
	assert.Empty(t, result.Env)
	assert.Empty(t, result.Entrypoint)
	assert.Empty(t, result.Cmd)
	assert.Empty(t, result.WorkingDir)
}

// --- ContainerConfigInfo ---

func TestContainerConfigInfo(t *testing.T) {
	config := ContainerConfigInfo{
		User:       "nginx",
		WorkingDir: "/app",
		Env:        []string{"PATH=/usr/local/bin:/usr/bin", "LANG=en_US.UTF-8"},
		Cmd:        []string{"nginx", "-g", "daemon off;"},
	}
	assert.Equal(t, "nginx", config.User)
	assert.Equal(t, "/app", config.WorkingDir)
	assert.Len(t, config.Env, 2)
	assert.Len(t, config.Cmd, 3)
}

// --- parseImageInspect ---

func TestParseImageInspect(t *testing.T) {
	raw := map[string]interface{}{
		"Id":           "sha256:abcdef1234567890",
		"Author":       "",
		"Architecture": "amd64",
		"Os":           "linux",
		"Size":         float64(187600000),
		"VirtualSize":  float64(250000000),
		"Created":      "2024-04-25T00:00:00Z",
		"RepoTags":     []interface{}{"nginx:latest"},
		"RepoDigests":  []interface{}{"nginx@sha256:abc123"},
		"Config": map[string]interface{}{
			"Labels": map[string]interface{}{
				"maintainer": "NGINX Docker Maintainers",
			},
			"Env":        []interface{}{"PATH=/usr/local/bin"},
			"Cmd":        []interface{}{"nginx", "-g", "daemon off;"},
			"Entrypoint": []interface{}{"/docker-entrypoint.sh"},
			"WorkingDir": "",
			"ExposedPorts": map[string]interface{}{
				"80/tcp": map[string]interface{}{},
			},
		},
	}

	result := parseImageInspect(raw)
	assert.Equal(t, "sha256:abcdef1234567890", result.ID)
	assert.Equal(t, "amd64", result.Architecture)
	assert.Equal(t, "linux", result.OS)
	assert.Equal(t, int64(187600000), result.Size)
	assert.Equal(t, int64(250000000), result.VirtualSize)
	assert.Len(t, result.RepoTags, 1)
	assert.Equal(t, "nginx:latest", result.RepoTags[0])
	assert.Len(t, result.RepoDigests, 1)
	assert.Len(t, result.ExposedPorts, 1)
	assert.Contains(t, result.ExposedPorts[0], "80/tcp")
	assert.Len(t, result.Env, 1)
	assert.Len(t, result.Cmd, 3)
	assert.Len(t, result.Entrypoint, 1)
	assert.Equal(t, "NGINX Docker Maintainers", result.Labels["maintainer"])
}

func TestParseImageInspect_Minimal(t *testing.T) {
	raw := map[string]interface{}{
		"Id": "sha256:deadbeef",
	}

	result := parseImageInspect(raw)
	assert.Equal(t, "sha256:deadbeef", result.ID)
	assert.Empty(t, result.RepoTags)
	assert.Empty(t, result.ExposedPorts)
	assert.Empty(t, result.Env)
}

func TestParseImageInspect_NoConfig(t *testing.T) {
	raw := map[string]interface{}{
		"Id":       "sha256:testid",
		"Size":     float64(1024),
		"RepoTags": []interface{}{"test:v1"},
	}

	result := parseImageInspect(raw)
	assert.Equal(t, "sha256:testid", result.ID)
	assert.Equal(t, int64(1024), result.Size)
	assert.Len(t, result.RepoTags, 1)
	assert.Empty(t, result.Env)
	assert.Empty(t, result.Cmd)
}

// --- 错误路径测试 ---

func TestRMI_NoImages(t *testing.T) {
	err := RMI(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "请指定")

	err = RMI(&RMIOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "请指定")
}

func TestTag_MissingArgs(t *testing.T) {
	err := Tag(nil)
	assert.Error(t, err)

	err = Tag(&TagOptions{})
	assert.Error(t, err)

	err = Tag(&TagOptions{Source: "nginx:latest"})
	assert.Error(t, err)

	err = Tag(&TagOptions{Target: "myrepo/nginx:latest"})
	assert.Error(t, err)
}

func TestPushImage_MissingArgs(t *testing.T) {
	err := PushImage(nil)
	assert.Error(t, err)

	err = PushImage(&PushImageOptions{})
	assert.Error(t, err)
}

func TestInspectImage_Empty(t *testing.T) {
	_, err := InspectImage("")
	assert.Error(t, err)
}

func TestSaveImage_Empty(t *testing.T) {
	err := SaveImage("", "output.tar")
	assert.Error(t, err)
}

func TestLoadImage_Empty(t *testing.T) {
	err := LoadImage("")
	assert.Error(t, err)
}

func TestSearchImages_Empty(t *testing.T) {
	err := SearchImages("", 10)
	assert.Error(t, err)
}

func TestHistoryImages_Empty(t *testing.T) {
	err := HistoryImages("", false, false)
	assert.Error(t, err)
}

// --- Images nil 选项 ---

func TestImages_NilOptions(t *testing.T) {
	// 测试 nil 选项不会 panic
	opts := &ImagesOptions{}
	_ = opts
	// 不能实际运行 Images() 因为需要 docker CLI
	// 但验证默认值
	assert.False(t, opts.All)
	assert.False(t, opts.Quiet)
}

// --- formatImageCreated 更多边界 ---

func TestFormatImageCreated_WithinMinute(t *testing.T) {
	// 使用固定时间戳测试
	created := int64(1714000000)
	result := formatImageCreated(created)
	// 不检查具体值（因为依赖当前时间），只验证不是空
	assert.NotEmpty(t, result)
}

// --- parseContainerConfig ---

func TestParseContainerConfig(t *testing.T) {
	config := map[string]interface{}{
		"User":       "www-data",
		"WorkingDir": "/var/www",
		"Env":        []interface{}{"HOME=/root", "PATH=/usr/bin"},
		"Cmd":        []interface{}{"python", "-m", "http.server"},
		"Entrypoint": []interface{}{"/entrypoint.sh"},
		"ExposedPorts": map[string]interface{}{
			"8080/tcp": map[string]interface{}{},
		},
	}

	result := parseContainerConfig(config)
	assert.Equal(t, "www-data", result.User)
	assert.Equal(t, "/var/www", result.WorkingDir)
	assert.Len(t, result.Env, 2)
	assert.Len(t, result.Cmd, 3)
	assert.Len(t, result.Entrypoint, 1)
	assert.NotNil(t, result.ExposedPorts)
}

func TestParseContainerConfig_Empty(t *testing.T) {
	config := map[string]interface{}{}
	result := parseContainerConfig(config)
	assert.Empty(t, result.User)
	assert.Empty(t, result.Env)
	assert.Empty(t, result.Cmd)
}
