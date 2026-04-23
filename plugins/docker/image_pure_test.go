package docker

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================
// parseRepoTag 扩展边界用例
// ============================

func TestParseRepoTag_WithRegistryAndPort(t *testing.T) {
	// 带端口的仓库地址镜像 — 注意 SplitN 只按第一个冒号分割
	repo, tag := parseRepoTag([]string{"myregistry.io:5000/myapp:v2"})
	assert.Equal(t, "myregistry.io", repo)
	assert.Equal(t, "5000/myapp:v2", tag)
}

func TestParseRepoTag_MultipleTags(t *testing.T) {
	// 多标签时只取第一个
	repo, tag := parseRepoTag([]string{"nginx:latest", "nginx:stable", "nginx:1.25"})
	assert.Equal(t, "nginx", repo)
	assert.Equal(t, "latest", tag)
}

func TestParseRepoTag_ColonInTag(t *testing.T) {
	// 标签中包含冒号（端口）
	repo, tag := parseRepoTag([]string{"myapp:v1:extra"})
	assert.Equal(t, "myapp", repo)
	assert.Equal(t, "v1:extra", tag)
}

// ============================
// formatImageID 扩展用例
// ============================

func TestFormatImageID_Exactly12Chars(t *testing.T) {
	// 正好 12 字符不截断
	id := formatImageID("abcdef123456")
	assert.Equal(t, "abcdef123456", id)
}

func TestFormatImageID_13Chars(t *testing.T) {
	// 13 字符应截断为 12
	id := formatImageID("abcdef1234567")
	assert.Equal(t, "abcdef123456", id)
}

func TestFormatImageID_Sha256Exactly12Chars(t *testing.T) {
	// sha256: + 正好12字符
	id := formatImageID("sha256:1234567890ab")
	assert.Equal(t, "1234567890ab", id)
}

func TestFormatImageID_Sha256Short(t *testing.T) {
	// sha256: + 短 ID
	id := formatImageID("sha256:abc")
	assert.Equal(t, "abc", id)
}

// ============================
// formatImageCreated 扩展用例
// ============================

func TestFormatImageCreated_MonthsAgo(t *testing.T) {
	// 几个月前（60天）
	created := time.Now().Unix() - 60*86400
	result := formatImageCreated(created)
	assert.Equal(t, "2月前", result)
}

func TestFormatImageCreated_YearsAgo(t *testing.T) {
	// 几年前（400天）
	created := time.Now().Unix() - 400*86400
	result := formatImageCreated(created)
	assert.Equal(t, "1年前", result)
}

func TestFormatImageCreated_ManyYearsAgo(t *testing.T) {
	// 很多年前（1000天）
	created := time.Now().Unix() - 1000*86400
	result := formatImageCreated(created)
	assert.Equal(t, "2年前", result)
}

func TestFormatImageCreated_JustUnderOneMinute(t *testing.T) {
	// 59 秒前，仍然属于"刚刚"
	created := time.Now().Unix() - 59
	result := formatImageCreated(created)
	assert.Equal(t, "刚刚", result)
}

func TestFormatImageCreated_ExactlyOneHour(t *testing.T) {
	// 正好 1 小时前
	created := time.Now().Unix() - 3600
	result := formatImageCreated(created)
	assert.Equal(t, "1小时前", result)
}

func TestFormatImageCreated_ExactlyOneDay(t *testing.T) {
	// 正好 1 天前
	created := time.Now().Unix() - 86400
	result := formatImageCreated(created)
	assert.Equal(t, "1天前", result)
}

func TestFormatImageCreated_JustUnder30Days(t *testing.T) {
	// 29 天前，仍然属于"天"
	created := time.Now().Unix() - 29*86400
	result := formatImageCreated(created)
	assert.Equal(t, "29天前", result)
}

func TestFormatImageCreated_JustOver30Days(t *testing.T) {
	// 31 天前，属于"月"
	created := time.Now().Unix() - 31*86400
	result := formatImageCreated(created)
	assert.Equal(t, "1月前", result)
}

func TestFormatImageCreated_JustUnder365Days(t *testing.T) {
	// 360 天前，仍然属于"月"
	created := time.Now().Unix() - 360*86400
	result := formatImageCreated(created)
	assert.Equal(t, "12月前", result)
}

// ============================
// toInt64 扩展用例
// ============================

func TestToInt64_JsonNumber(t *testing.T) {
	// json.Number 类型
	n := json.Number("12345")
	result := toInt64(n)
	assert.Equal(t, int64(12345), result)
}

func TestToInt64_JsonNumberInvalid(t *testing.T) {
	// 无效 json.Number
	n := json.Number("not-a-number")
	result := toInt64(n)
	assert.Equal(t, int64(0), result)
}

func TestToInt64_NegativeFloat64(t *testing.T) {
	// 负数 float64
	result := toInt64(float64(-3.9))
	assert.Equal(t, int64(-3), result)
}

func TestToInt64_NegativeInt(t *testing.T) {
	// 负数 int
	result := toInt64(-42)
	assert.Equal(t, int64(-42), result)
}

func TestToInt64_LargeFloat64(t *testing.T) {
	// 大数 float64
	result := toInt64(float64(9999999999))
	assert.Equal(t, int64(9999999999), result)
}

func TestToInt64_StringNegative(t *testing.T) {
	// 字符串负数
	result := toInt64("-100")
	assert.Equal(t, int64(-100), result)
}

func TestToInt64_StructType(t *testing.T) {
	// 不支持的 struct 类型
	result := toInt64(struct{ A int }{A: 1})
	assert.Equal(t, int64(0), result)
}

func TestToInt64_SliceType(t *testing.T) {
	// 不支持的 slice 类型
	result := toInt64([]int{1, 2, 3})
	assert.Equal(t, int64(0), result)
}

// ============================
// getStr 扩展用例
// ============================

func TestGetStr_EmptyStringValue(t *testing.T) {
	// 空字符串值（存在但为空）
	m := map[string]interface{}{"key": ""}
	result := getStr(m, "key")
	assert.Equal(t, "", result)
}

func TestGetStr_BoolValue(t *testing.T) {
	// bool 值不是字符串
	m := map[string]interface{}{"flag": true}
	result := getStr(m, "flag")
	assert.Equal(t, "", result)
}

func TestGetStr_SliceValue(t *testing.T) {
	// slice 值不是字符串
	m := map[string]interface{}{"items": []string{"a", "b"}}
	result := getStr(m, "items")
	assert.Equal(t, "", result)
}

func TestGetStr_FloatValue(t *testing.T) {
	// float 值不是字符串
	m := map[string]interface{}{"price": float64(9.99)}
	result := getStr(m, "price")
	assert.Equal(t, "", result)
}

func TestGetStr_UnicodeValue(t *testing.T) {
	// Unicode 字符串
	m := map[string]interface{}{"name": "中文镜像"}
	result := getStr(m, "name")
	assert.Equal(t, "中文镜像", result)
}

// ============================
// parseImageInspect 扩展用例
// ============================

func TestParseImageInspect_WithAuthor(t *testing.T) {
	// 带作者的镜像
	raw := map[string]interface{}{
		"Id":           "sha256:abc123",
		"Author":       "test@example.com",
		"Architecture": "arm64",
		"Os":           "linux",
		"Created":      "2024-04-25T12:00:00Z",
	}
	result := parseImageInspect(raw)
	assert.Equal(t, "test@example.com", result.Author)
	assert.Equal(t, "arm64", result.Architecture)
	assert.Equal(t, "linux", result.OS)
	assert.False(t, result.Created.IsZero(), "Created 应被成功解析")
}

func TestParseImageInspect_CreatedRFC3339Nano(t *testing.T) {
	// RFC3339Nano 格式时间
	raw := map[string]interface{}{
		"Id":      "sha256:abc",
		"Created": "2024-04-25T12:00:00.123456789Z",
	}
	result := parseImageInspect(raw)
	assert.False(t, result.Created.IsZero(), "RFC3339Nano 应被正确解析")
	assert.Equal(t, 2024, result.Created.Year())
}

func TestParseImageInspect_CreatedInvalid(t *testing.T) {
	// 无效的 Created 字符串
	raw := map[string]interface{}{
		"Id":      "sha256:abc",
		"Created": "invalid-date",
	}
	result := parseImageInspect(raw)
	assert.True(t, result.Created.IsZero(), "无效时间应保持零值")
}

func TestParseImageInspect_EmptyCreated(t *testing.T) {
	// Created 为空字符串
	raw := map[string]interface{}{
		"Id":      "sha256:abc",
		"Created": "",
	}
	result := parseImageInspect(raw)
	assert.True(t, result.Created.IsZero())
}

func TestParseImageInspect_SizeAsInt(t *testing.T) {
	// Size 为 int 类型而非 float64
	raw := map[string]interface{}{
		"Id":          "sha256:abc",
		"Size":        int(2048),
		"VirtualSize": int64(4096),
	}
	result := parseImageInspect(raw)
	assert.Equal(t, int64(2048), result.Size)
	assert.Equal(t, int64(4096), result.VirtualSize)
}

func TestParseImageInspect_SizeAsString(t *testing.T) {
	// Size 为 string 类型
	raw := map[string]interface{}{
		"Id":          "sha256:abc",
		"Size":        "1024",
		"VirtualSize": "2048",
	}
	result := parseImageInspect(raw)
	assert.Equal(t, int64(1024), result.Size)
	assert.Equal(t, int64(2048), result.VirtualSize)
}

func TestParseImageInspect_SizeNil(t *testing.T) {
	// Size 为 nil
	raw := map[string]interface{}{
		"Id": "sha256:abc",
	}
	result := parseImageInspect(raw)
	assert.Equal(t, int64(0), result.Size)
	assert.Equal(t, int64(0), result.VirtualSize)
}

func TestParseImageInspect_RepoTagsWithNonString(t *testing.T) {
	// RepoTags 包含非字符串元素
	raw := map[string]interface{}{
		"Id":       "sha256:abc",
		"RepoTags": []interface{}{"nginx:latest", 42, "redis:alpine"},
	}
	result := parseImageInspect(raw)
	assert.Equal(t, []string{"nginx:latest", "redis:alpine"}, result.RepoTags)
}

func TestParseImageInspect_RepoDigestsWithNonString(t *testing.T) {
	// RepoDigests 包含非字符串元素
	raw := map[string]interface{}{
		"Id":          "sha256:abc",
		"RepoDigests": []interface{}{"nginx@sha256:abc", true},
	}
	result := parseImageInspect(raw)
	assert.Equal(t, []string{"nginx@sha256:abc"}, result.RepoDigests)
}

func TestParseImageInspect_LabelsWithNonStringValues(t *testing.T) {
	// Labels 包含非字符串值
	raw := map[string]interface{}{
		"Id": "sha256:abc",
		"Config": map[string]interface{}{
			"Labels": map[string]interface{}{
				"version": "1.0",
				"count":   42,
			},
		},
	}
	result := parseImageInspect(raw)
	assert.Equal(t, "1.0", result.Labels["version"])
	_, exists := result.Labels["count"]
	assert.False(t, exists, "非字符串标签值应被忽略")
}

func TestParseImageInspect_ConfigWithNonMapLabels(t *testing.T) {
	// Config.Labels 不是 map
	raw := map[string]interface{}{
		"Id": "sha256:abc",
		"Config": map[string]interface{}{
			"Labels": "not-a-map",
		},
	}
	result := parseImageInspect(raw)
	assert.Empty(t, result.Labels)
}

func TestParseImageInspect_ExposedPorts(t *testing.T) {
	// 多个暴露端口
	raw := map[string]interface{}{
		"Id": "sha256:abc",
		"Config": map[string]interface{}{
			"ExposedPorts": map[string]interface{}{
				"80/tcp":  map[string]interface{}{},
				"443/tcp": map[string]interface{}{},
			},
		},
	}
	result := parseImageInspect(raw)
	assert.Len(t, result.ExposedPorts, 2)
	assert.Contains(t, result.ExposedPorts, "80/tcp")
	assert.Contains(t, result.ExposedPorts, "443/tcp")
}

func TestParseImageInspect_EnvWithNonString(t *testing.T) {
	// Env 包含非字符串元素
	raw := map[string]interface{}{
		"Id": "sha256:abc",
		"Config": map[string]interface{}{
			"Env": []interface{}{"PATH=/usr/bin", 123, "HOME=/root"},
		},
	}
	result := parseImageInspect(raw)
	assert.Equal(t, []string{"PATH=/usr/bin", "HOME=/root"}, result.Env)
}

func TestParseImageInspect_EntrypointWithNonString(t *testing.T) {
	// Entrypoint 包含非字符串元素
	raw := map[string]interface{}{
		"Id": "sha256:abc",
		"Config": map[string]interface{}{
			"Entrypoint": []interface{}{"/entrypoint.sh", 42},
		},
	}
	result := parseImageInspect(raw)
	assert.Equal(t, []string{"/entrypoint.sh"}, result.Entrypoint)
}

func TestParseImageInspect_CmdWithNonString(t *testing.T) {
	// Cmd 包含非字符串元素
	raw := map[string]interface{}{
		"Id": "sha256:abc",
		"Config": map[string]interface{}{
			"Cmd": []interface{}{"run", true, "server"},
		},
	}
	result := parseImageInspect(raw)
	assert.Equal(t, []string{"run", "server"}, result.Cmd)
}

func TestParseImageInspect_WorkingDir(t *testing.T) {
	// WorkingDir 设置
	raw := map[string]interface{}{
		"Id": "sha256:abc",
		"Config": map[string]interface{}{
			"WorkingDir": "/app/src",
		},
	}
	result := parseImageInspect(raw)
	assert.Equal(t, "/app/src", result.WorkingDir)
}

func TestParseImageInspect_ConfigNotMap(t *testing.T) {
	// Config 不是 map 类型
	raw := map[string]interface{}{
		"Id":     "sha256:abc",
		"Config": "invalid",
	}
	result := parseImageInspect(raw)
	assert.Empty(t, result.Env)
	assert.Empty(t, result.Cmd)
	assert.Empty(t, result.ExposedPorts)
	assert.Equal(t, "", result.WorkingDir)
}

func TestParseImageInspect_FullRoundTrip(t *testing.T) {
	// 完整的 JSON 反序列化 + parseImageInspect 端到端
	jsonStr := `{
		"Id": "sha256:abcd12345678",
		"RepoTags": ["myapp:v1.0"],
		"RepoDigests": ["myapp@sha256:abcdef"],
		"Created": "2024-06-15T10:30:00Z",
		"Author": "dev@example.com",
		"Architecture": "amd64",
		"Os": "linux",
		"Size": 100000000,
		"VirtualSize": 150000000,
		"Config": {
			"User": "appuser",
			"WorkingDir": "/home/app",
			"Labels": {"com.example.version": "1.0"},
			"Env": ["APP_ENV=prod", "PORT=8080"],
			"Entrypoint": ["/start.sh"],
			"Cmd": ["--mode", "server"],
			"ExposedPorts": {"8080/tcp": {}}
		}
	}`
	var raw map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &raw)
	require.NoError(t, err)

	result := parseImageInspect(raw)
	assert.Equal(t, "sha256:abcd12345678", result.ID)
	assert.Equal(t, []string{"myapp:v1.0"}, result.RepoTags)
	assert.Equal(t, []string{"myapp@sha256:abcdef"}, result.RepoDigests)
	assert.Equal(t, "dev@example.com", result.Author)
	assert.Equal(t, "amd64", result.Architecture)
	assert.Equal(t, "linux", result.OS)
	assert.Equal(t, int64(100000000), result.Size)
	assert.Equal(t, int64(150000000), result.VirtualSize)
	assert.Equal(t, "1.0", result.Labels["com.example.version"])
	assert.Equal(t, []string{"APP_ENV=prod", "PORT=8080"}, result.Env)
	assert.Equal(t, []string{"/start.sh"}, result.Entrypoint)
	assert.Equal(t, []string{"--mode", "server"}, result.Cmd)
	assert.Equal(t, "/home/app", result.WorkingDir)
	assert.Contains(t, result.ExposedPorts, "8080/tcp")
	assert.Equal(t, "appuser", result.ContainerConfig.User)
	assert.Equal(t, "/home/app", result.ContainerConfig.WorkingDir)
}

// ============================
// parseContainerConfig 扩展用例
// ============================

func TestParseContainerConfig_NilConfig(t *testing.T) {
	// nil map 传入（Go 中 nil map 可安全读取）
	result := parseContainerConfig(nil)
	assert.Empty(t, result.User)
	assert.Empty(t, result.Env)
	assert.Empty(t, result.Cmd)
}

func TestParseContainerConfig_EnvWithNonString(t *testing.T) {
	// Env 包含非字符串元素
	config := map[string]interface{}{
		"Env": []interface{}{"KEY=VAL", 123, "OTHER=thing"},
	}
	result := parseContainerConfig(config)
	assert.Equal(t, []string{"KEY=VAL", "OTHER=thing"}, result.Env)
}

func TestParseContainerConfig_EntrypointWithNonString(t *testing.T) {
	// Entrypoint 包含非字符串元素
	config := map[string]interface{}{
		"Entrypoint": []interface{}{"/bin/sh", nil},
	}
	result := parseContainerConfig(config)
	assert.Equal(t, []string{"/bin/sh"}, result.Entrypoint)
}

func TestParseContainerConfig_CmdWithNonString(t *testing.T) {
	// Cmd 包含非字符串元素
	config := map[string]interface{}{
		"Cmd": []interface{}{"serve", 42, "--verbose"},
	}
	result := parseContainerConfig(config)
	assert.Equal(t, []string{"serve", "--verbose"}, result.Cmd)
}

func TestParseContainerConfig_ExposedPortsWrongType(t *testing.T) {
	// ExposedPorts 不是 map
	config := map[string]interface{}{
		"ExposedPorts": "80/tcp",
	}
	result := parseContainerConfig(config)
	assert.Nil(t, result.ExposedPorts)
}

func TestParseContainerConfig_EnvWrongType(t *testing.T) {
	// Env 不是 []interface{}
	config := map[string]interface{}{
		"Env": "PATH=/usr/bin",
	}
	result := parseContainerConfig(config)
	assert.Empty(t, result.Env)
}

func TestParseContainerConfig_FullConfig(t *testing.T) {
	// 完整配置
	config := map[string]interface{}{
		"User":       "root",
		"WorkingDir": "/data",
		"Env":        []interface{}{"A=1", "B=2"},
		"Cmd":        []interface{}{"./app"},
		"Entrypoint": []interface{}{"/init"},
		"ExposedPorts": map[string]interface{}{
			"9090/tcp": map[string]interface{}{},
			"53/udp":   map[string]interface{}{},
		},
	}
	result := parseContainerConfig(config)
	assert.Equal(t, "root", result.User)
	assert.Equal(t, "/data", result.WorkingDir)
	assert.Equal(t, []string{"A=1", "B=2"}, result.Env)
	assert.Equal(t, []string{"./app"}, result.Cmd)
	assert.Equal(t, []string{"/init"}, result.Entrypoint)
	assert.NotNil(t, result.ExposedPorts)
	assert.Len(t, result.ExposedPorts, 2)
}

// ============================
// ImageInfo 结构体边界用例
// ============================

func TestImageInfo_EmptyJSON(t *testing.T) {
	// 空的 JSON 对象
	var img ImageInfo
	err := json.Unmarshal([]byte("{}"), &img)
	assert.NoError(t, err)
	assert.Equal(t, "", img.ID)
	assert.Nil(t, img.RepoTags)
	assert.Equal(t, int64(0), img.Created)
	assert.Equal(t, int64(0), img.Size)
}

func TestImageInfo_InvalidJSON(t *testing.T) {
	// 无效 JSON
	var img ImageInfo
	err := json.Unmarshal([]byte("not-json"), &img)
	assert.Error(t, err)
}

// ============================
// formatImageCreated 边界
// ============================

func TestFormatImageCreated_FutureTime(t *testing.T) {
	// 未来时间（应在1分钟内，显示"刚刚"）
	created := time.Now().Unix() + 30
	result := formatImageCreated(created)
	assert.Equal(t, "刚刚", result)
}

func TestFormatImageCreated_Exactly30Days(t *testing.T) {
	// 正好 30 天（属于"月"范畴）
	created := time.Now().Unix() - 30*86400
	result := formatImageCreated(created)
	assert.Equal(t, "1月前", result)
}

func TestFormatImageCreated_Exactly365Days(t *testing.T) {
	// 正好 365 天（属于"年"范畴）
	created := time.Now().Unix() - 365*86400
	result := formatImageCreated(created)
	assert.Equal(t, "1年前", result)
}

// ============================
// json.Number toInt64 边界
// ============================

func TestToInt64_JsonNumberLarge(t *testing.T) {
	// 大数 json.Number
	n := json.Number("9999999999999")
	result := toInt64(n)
	assert.Equal(t, int64(9999999999999), result)
}

func TestToInt64_JsonNumberZero(t *testing.T) {
	// 零值 json.Number
	n := json.Number("0")
	result := toInt64(n)
	assert.Equal(t, int64(0), result)
}

func TestToInt64_JsonNumberNegative(t *testing.T) {
	// 负数 json.Number
	n := json.Number("-500")
	result := toInt64(n)
	assert.Equal(t, int64(-500), result)
}

func TestToInt64_JsonNumberFloatString(t *testing.T) {
	// json.Number 包含浮点字符串（应解析失败）
	n := json.Number("3.14")
	result := toInt64(n)
	assert.Equal(t, int64(0), result)
}

// ============================
// PrintImageInspect 不会 panic 的验证
// ============================

func TestPrintImageInspect_MinimalResult(t *testing.T) {
	// 确保 PrintImageInspect 不会 panic（仅验证不崩溃）
	result := &ImageInspectResult{
		ID:       "sha256:abc123",
		RepoTags: []string{"test:v1"},
		Created:  time.Now(),
	}
	// 不验证输出内容（stdout），只确保不 panic
	assert.NotPanics(t, func() {
		PrintImageInspect(result)
	})
}

func TestPrintImageInspect_WithAllFields(t *testing.T) {
	// 所有字段都有值
	result := &ImageInspectResult{
		ID:           "sha256:abc123",
		RepoTags:     []string{"test:v1"},
		RepoDigests:  []string{"test@sha256:abc"},
		Created:      time.Now(),
		Author:       "dev@example.com",
		Architecture: "amd64",
		OS:           "linux",
		Size:         1024000,
		VirtualSize:  2048000,
		Labels:       map[string]string{"version": "1.0"},
		Entrypoint:   []string{"/entry.sh"},
		Cmd:          []string{"--serve"},
		ExposedPorts: []string{"8080/tcp"},
		Env:          []string{"ENV=prod"},
		WorkingDir:   "/app",
	}
	assert.NotPanics(t, func() {
		PrintImageInspect(result)
	})
}
