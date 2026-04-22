package builtin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== ParseDdArgs 测试 ====================

func TestParseDdArgs_FullOptions(t *testing.T) {
	args := []string{
		"if=/dev/zero", "of=/dev/null", "bs=4K", "count=100",
		"skip=10", "seek=5", "conv=notrunc", "status=none",
		"iflag=direct", "oflag=direct",
	}
	opts, err := ParseDdArgs(args)
	assert.NoError(t, err)
	assert.Equal(t, "/dev/zero", opts.If)
	assert.Equal(t, "/dev/null", opts.Of)
	assert.Equal(t, int64(4096), opts.Bs)
	assert.Equal(t, int64(100), opts.Count)
	assert.Equal(t, int64(10), opts.Skip)
	assert.Equal(t, int64(5), opts.Seek)
	assert.Equal(t, "notrunc", opts.Conv)
	assert.Equal(t, "none", opts.Status)
	assert.Equal(t, "direct", opts.IFlag)
	assert.Equal(t, "direct", opts.OFlag)
}

func TestParseDdArgs_EmptyArgs(t *testing.T) {
	opts, err := ParseDdArgs([]string{})
	assert.NoError(t, err)
	assert.Equal(t, DdOptions{}, opts)
}

func TestParseDdArgs_BasicIfOf(t *testing.T) {
	args := []string{"if=input.dat", "of=output.dat"}
	opts, err := ParseDdArgs(args)
	assert.NoError(t, err)
	assert.Equal(t, "input.dat", opts.If)
	assert.Equal(t, "output.dat", opts.Of)
}

func TestParseDdArgs_BsNumeric(t *testing.T) {
	args := []string{"bs=512"}
	opts, err := ParseDdArgs(args)
	assert.NoError(t, err)
	assert.Equal(t, int64(512), opts.Bs)
}

func TestParseDdArgs_StatusProgress(t *testing.T) {
	args := []string{"status=progress"}
	opts, err := ParseDdArgs(args)
	assert.NoError(t, err)
	assert.Equal(t, "progress", opts.Status)
}

func TestParseDdArgs_StatusNoxfer(t *testing.T) {
	args := []string{"status=noxfer"}
	opts, err := ParseDdArgs(args)
	assert.NoError(t, err)
	assert.Equal(t, "noxfer", opts.Status)
}

func TestParseDdArgs_NoEquals(t *testing.T) {
	args := []string{"noequals"}
	_, err := ParseDdArgs(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无效的参数格式")
	assert.Contains(t, err.Error(), "noequals")
}

func TestParseDdArgs_InvalidBs(t *testing.T) {
	args := []string{"bs=abc"}
	_, err := ParseDdArgs(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无效的 bs 值")
}

func TestParseDdArgs_InvalidCount(t *testing.T) {
	args := []string{"count=abc"}
	_, err := ParseDdArgs(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无效的 count 值")
}

func TestParseDdArgs_InvalidSkip(t *testing.T) {
	args := []string{"skip=abc"}
	_, err := ParseDdArgs(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无效的 skip 值")
}

func TestParseDdArgs_InvalidSeek(t *testing.T) {
	args := []string{"seek=abc"}
	_, err := ParseDdArgs(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无效的 seek 值")
}

func TestParseDdArgs_InvalidStatus(t *testing.T) {
	args := []string{"status=invalid"}
	_, err := ParseDdArgs(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无效的 status 值")
	assert.Contains(t, err.Error(), "invalid")
}

func TestParseDdArgs_UnknownParam(t *testing.T) {
	args := []string{"foo=bar"}
	_, err := ParseDdArgs(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未知参数")
	assert.Contains(t, err.Error(), "foo")
}

func TestParseDdArgs_CaseInsensitiveKeys(t *testing.T) {
	args := []string{"IF=/dev/zero", "OF=/dev/null", "BS=1M", "COUNT=1"}
	opts, err := ParseDdArgs(args)
	assert.NoError(t, err)
	assert.Equal(t, "/dev/zero", opts.If)
	assert.Equal(t, "/dev/null", opts.Of)
	assert.Equal(t, int64(1048576), opts.Bs)
	assert.Equal(t, int64(1), opts.Count)
}

func TestParseDdArgs_BsWithSuffix(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantBs   int64
	}{
		{"bs_1K", []string{"bs=1K"}, 1024},
		{"bs_1M", []string{"bs=1M"}, 1048576},
		{"bs_1G", []string{"bs=1G"}, 1073741824},
		{"bs_4K", []string{"bs=4K"}, 4096},
		{"bs_4KB", []string{"bs=4KB"}, 4096},
		{"bs_1MB", []string{"bs=1MB"}, 1048576},
		{"bs_1GB", []string{"bs=1GB"}, 1073741824},
		{"bs_512", []string{"bs=512"}, 512},
		{"bs_1B", []string{"bs=1B"}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseDdArgs(tt.args)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantBs, opts.Bs)
		})
	}
}

func TestParseDdArgs_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		check   func(t *testing.T, opts DdOptions)
	}{
		{
			name:    "value_with_equals",
			args:    []string{"conv=a=b"},
			wantErr: false,
			check: func(t *testing.T, opts DdOptions) {
				assert.Equal(t, "a=b", opts.Conv)
			},
		},
		{
			name:    "empty_value",
			args:    []string{"if="},
			wantErr: false,
			check: func(t *testing.T, opts DdOptions) {
				assert.Equal(t, "", opts.If)
			},
		},
		{
			name:    "zero_count",
			args:    []string{"count=0"},
			wantErr: false,
			check: func(t *testing.T, opts DdOptions) {
				assert.Equal(t, int64(0), opts.Count)
			},
		},
		{
			name:    "large_count",
			args:    []string{"count=999999999"},
			wantErr: false,
			check: func(t *testing.T, opts DdOptions) {
				assert.Equal(t, int64(999999999), opts.Count)
			},
		},
		{
			name:    "negative_skip",
			args:    []string{"skip=-1"},
			wantErr: false,
			check: func(t *testing.T, opts DdOptions) {
				assert.Equal(t, int64(-1), opts.Skip)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseDdArgs(tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(t, opts)
				}
			}
		})
	}
}

// ==================== parseSize 测试 ====================

func TestParseSize_PlainNumber(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"0", 0},
		{"1", 1},
		{"512", 512},
		{"1024", 1024},
		{"999999", 999999},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseSize(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseSize_SuffixK(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"1K", 1024},
		{"4K", 4096},
		{"512K", 524288},
		{"1k", 1024},
		{"4k", 4096},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseSize(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseSize_SuffixM(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"1M", 1048576},
		{"10M", 10485760},
		{"1m", 1048576},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseSize(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseSize_SuffixG(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"1G", 1073741824},
		{"2G", 2147483648},
		{"1g", 1073741824},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseSize(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseSize_SuffixB(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"1B", 1},
		{"512B", 512},
		{"4KB", 4096},
		{"1MB", 1048576},
		{"1GB", 1073741824},
		{"1b", 1},
		{"4kb", 4096},
		{"1mb", 1048576},
		{"1gb", 1073741824},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseSize(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseSize_InvalidInput(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"abc"},
		{"K"},
		{"M"},
		{"G"},
		{""},
		{"12.5"},
		{"1KBx"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parseSize(tt.input)
			assert.Error(t, err)
		})
	}
}

// ==================== formatBytes 测试 ====================

func TestFormatBytes_Zero(t *testing.T) {
	assert.Equal(t, "0 B", formatBytes(0))
}

func TestFormatBytes_Bytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{1, "1 B"},
		{512, "512 B"},
		{100, "100 B"},
		{1023, "1023 B"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, formatBytes(tt.input))
		})
	}
}

func TestFormatBytes_KB(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{10240, "10.0 KB"},
		{512 * 1024, "512.0 KB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, formatBytes(tt.input))
		})
	}
}

func TestFormatBytes_MB(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{1024 * 1024, "1.0 MB"},
		{1536 * 1024, "1.5 MB"},
		{10 * 1024 * 1024, "10.0 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, formatBytes(tt.input))
		})
	}
}

func TestFormatBytes_GB(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{1024 * 1024 * 1024, "1.0 GB"},
		{2 * 1024 * 1024 * 1024, "2.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, formatBytes(tt.input))
		})
	}
}

func TestFormatBytes_TB(t *testing.T) {
	input := int64(1024) * 1024 * 1024 * 1024 // 1 TB
	assert.Equal(t, "1.0 TB", formatBytes(input))
}

func TestFormatBytes_LargeValues(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  string
	}{
		{"100_bytes", 100, "100 B"},
		{"2048_bytes", 2048, "2.0 KB"},
		{"1048576_bytes", 1048576, "1.0 MB"},
		{"1073741824_bytes", 1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatBytes(tt.input))
		})
	}
}
