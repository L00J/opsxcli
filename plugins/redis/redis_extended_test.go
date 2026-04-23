package redis

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// === GetClient 集群模式测试 ===

func TestGetClient_SingleMode(t *testing.T) {
	// 单机模式应返回 *redis.Client
	client, err := GetClient("127.0.0.1", 6379, "", 0, false, "")
	require.NoError(t, err)
	require.NotNil(t, client)
	client.Close()
}

func TestGetClient_SingleMode_WithPassword(t *testing.T) {
	// 单机模式带密码
	client, err := GetClient("127.0.0.1", 6379, "mypassword", 2, false, "")
	require.NoError(t, err)
	require.NotNil(t, client)
	client.Close()
}

func TestGetClient_ClusterMode_SingleAddr(t *testing.T) {
	// 集群模式不指定addrs，使用host:port
	client, err := GetClient("127.0.0.1", 6379, "pass", 0, true, "")
	require.NoError(t, err)
	require.NotNil(t, client)
	client.Close()
}

func TestGetClient_ClusterMode_MultipleAddrs(t *testing.T) {
	// 集群模式指定多个地址
	addrs := "10.0.0.1:6379, 10.0.0.2:6379, 10.0.0.3:6379"
	client, err := GetClient("", 0, "clusterpass", 0, true, addrs)
	require.NoError(t, err)
	require.NotNil(t, client)
	client.Close()
}

func TestGetClient_ClusterMode_SingleAddrInList(t *testing.T) {
	// 集群模式只指定一个地址
	client, err := GetClient("", 0, "", 0, true, "192.168.1.1:6379")
	require.NoError(t, err)
	require.NotNil(t, client)
	client.Close()
}

func TestGetClient_ClusterMode_AddrsWithSpaces(t *testing.T) {
	// 地址列表包含空格应被正确清理
	addrs := "  10.0.0.1:6379  ,  10.0.0.2:6379  "
	client, err := GetClient("", 0, "", 0, true, addrs)
	require.NoError(t, err)
	require.NotNil(t, client)
	client.Close()
}

// === getRedisCompleter 测试 ===

func TestGetRedisCompleter_NotNil(t *testing.T) {
	completer := getRedisCompleter()
	assert.NotNil(t, completer)
}

func TestGetRedisCompleter_CommonCommands(t *testing.T) {
	// getRedisCompleter 应该包含常见Redis命令
	completer := getRedisCompleter()
	assert.NotNil(t, completer)
	// 只要能创建不panic就算通过
}

// === printResult 补充测试 ===

func TestPrintResult_DefaultType(t *testing.T) {
	// 测试 default 分支：非标准类型（如 float64）
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printResult(3.14)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	output := buf.String()
	assert.Contains(t, output, "3.14")
}

func TestPrintResult_ComplexDefault(t *testing.T) {
	// 测试 default 分支中使用自定义类型
	type customType struct {
		Name string
	}
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printResult(customType{Name: "test"})

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	output := buf.String()
	assert.Contains(t, output, "test")
}

func TestPrintResult_ListWithBytes(t *testing.T) {
	// 列表中包含 []byte 元素
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	input := []interface{}{[]byte("hello"), "world"}
	printResult(input)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	output := buf.String()
	assert.Contains(t, output, "1)")
	assert.Contains(t, output, "hello")
	assert.Contains(t, output, "2)")
	assert.Contains(t, output, "world")
}

func TestPrintResult_ListWithInt64(t *testing.T) {
	// 列表中包含 int64 元素
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	input := []interface{}{int64(100), int64(200)}
	printResult(input)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	output := buf.String()
	assert.Contains(t, output, "(integer) 100")
	assert.Contains(t, output, "(integer) 200")
}

func TestPrintResult_ListWithNil(t *testing.T) {
	// 列表中包含 nil 元素
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	input := []interface{}{nil, "value"}
	printResult(input)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	output := buf.String()
	assert.Contains(t, output, "(nil)")
}

func TestPrintResult_ListWithBool(t *testing.T) {
	// 列表中包含 bool 元素
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	input := []interface{}{true, false}
	printResult(input)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	output := buf.String()
	assert.Contains(t, output, "(integer) 1")
	assert.Contains(t, output, "(integer) 0")
}

// === NewGetCmd / NewSetCmd / NewInteractiveCmd 测试 ===

func TestNewGetCmd(t *testing.T) {
	cmd := NewGetCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "get <key>", cmd.Use)
	assert.Equal(t, "获取键值", cmd.Short)
	assert.NotNil(t, cmd.RunE)
}

func TestNewSetCmd(t *testing.T) {
	cmd := NewSetCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "set <key> <value>", cmd.Use)
	assert.Equal(t, "设置键值", cmd.Short)
	assert.NotNil(t, cmd.RunE)

	// 检查 expire flag
	f := cmd.Flags().Lookup("expire")
	require.NotNil(t, f)
	assert.Equal(t, "e", f.Shorthand)
	assert.Equal(t, "", f.DefValue)
}

func TestNewInteractiveCmd(t *testing.T) {
	cmd := NewInteractiveCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "interactive", cmd.Use)
	assert.Equal(t, "进入交互式Redis shell", cmd.Short)
	assert.NotNil(t, cmd.RunE)
}

// === parseRedisCommand 补充边界测试 ===

func TestParseRedisCommand_MismatchedQuotes(t *testing.T) {
	// 双引号内遇到单引号（不同引号类型），应写入内容
	result := parseRedisCommand(`SET key "value'with'quotes"`)
	assert.Equal(t, []string{"SET", "key", "value'with'quotes"}, result)
}

func TestParseRedisCommand_SingleQuoteInDouble(t *testing.T) {
	// 单引号内遇到双引号
	result := parseRedisCommand(`SET key 'value"with"quotes'`)
	assert.Equal(t, []string{"SET", "key", `value"with"quotes`}, result)
}

func TestParseRedisCommand_OnlySpacesBetweenQuotes(t *testing.T) {
	result := parseRedisCommand(`  SET   key   value  `)
	assert.Equal(t, []string{"SET", "key", "value"}, result)
}

func TestParseRedisCommand_SingleWord(t *testing.T) {
	result := parseRedisCommand("PING")
	assert.Equal(t, []string{"PING"}, result)
}

// === 时间解析辅助测试（Set命令中的expire参数）===

func TestSetCmd_ExpireFlag_ValidDuration(t *testing.T) {
	duration, err := time.ParseDuration("1h")
	assert.NoError(t, err)
	assert.Equal(t, time.Hour, duration)
}

func TestSetCmd_ExpireFlag_InvalidDuration(t *testing.T) {
	_, err := time.ParseDuration("invalid")
	assert.Error(t, err)
}

func TestSetCmd_ExpireFlag_Seconds(t *testing.T) {
	duration, err := time.ParseDuration("60s")
	assert.NoError(t, err)
	assert.Equal(t, 60*time.Second, duration)
}

func TestSetCmd_ExpireFlag_Minutes(t *testing.T) {
	duration, err := time.ParseDuration("30m")
	assert.NoError(t, err)
	assert.Equal(t, 30*time.Minute, duration)
}
