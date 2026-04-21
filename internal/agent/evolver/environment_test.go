package evolver

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// === parseHost 测试 ===

func TestParseHost_PlainHost(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	user, hostname, port := em.parseHost("192.168.1.1")
	assert.Equal(t, "root", user)
	assert.Equal(t, "192.168.1.1", hostname)
	assert.Equal(t, "22", port)
}

func TestParseHost_UserAndHost(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	user, hostname, port := em.parseHost("admin@10.0.0.1")
	assert.Equal(t, "admin", user)
	assert.Equal(t, "10.0.0.1", hostname)
	assert.Equal(t, "22", port)
}

func TestParseHost_UserHostPort(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	user, hostname, port := em.parseHost("deploy@web01:2222")
	assert.Equal(t, "deploy", user)
	assert.Equal(t, "web01", hostname)
	assert.Equal(t, "2222", port)
}

func TestParseHost_HostWithPort(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	user, hostname, port := em.parseHost("server.com:443")
	assert.Equal(t, "root", user)
	assert.Equal(t, "server.com", hostname)
	assert.Equal(t, "443", port)
}

func TestParseHost_EmptyString(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	user, hostname, port := em.parseHost("")
	assert.Equal(t, "root", user)
	assert.Equal(t, "", hostname)
	assert.Equal(t, "22", port)
}

// === appendUnique 测试 ===

func TestAppendUnique_NewItem(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	slice := []string{"a", "b"}
	result := em.appendUnique(slice, "c")
	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func TestAppendUnique_DuplicateItem(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	slice := []string{"a", "b", "c"}
	result := em.appendUnique(slice, "b")
	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func TestAppendUnique_EmptySlice(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	slice := []string{}
	result := em.appendUnique(slice, "x")
	assert.Equal(t, []string{"x"}, result)
}

// === querySimilarity 测试 (Jaccard) ===

func TestQuerySimilarity_Identical(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	score := em.querySimilarity("查看磁盘空间", "查看磁盘空间")
	assert.Equal(t, 1.0, score)
}

func TestQuerySimilarity_CompletelyDifferent(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	score := em.querySimilarity("查看磁盘空间", "重启nginx服务")
	assert.Equal(t, 0.0, score)
}

func TestQuerySimilarity_PartialOverlap(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	score := em.querySimilarity("查看磁盘空间", "查看内存使用")
	assert.True(t, score > 0 && score < 1.0)
}

func TestQuerySimilarity_EmptyStrings(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	score := em.querySimilarity("", "")
	assert.Equal(t, 0.0, score)
}

func TestQuerySimilarity_CaseInsensitive(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	score := em.querySimilarity("Check Disk", "check disk")
	assert.Equal(t, 1.0, score)
}

// === NewEnvironmentMemory 测试 ===

func TestNewEnvironmentMemory_Defaults(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	assert.NotNil(t, em)
	assert.Empty(t, em.KnownServers)
	assert.Empty(t, em.LastQueries)
	assert.Empty(t, em.CustomHints)
	assert.Equal(t, "balanced", em.UserPrefs.SafetyMode)
	assert.Equal(t, "vim", em.UserPrefs.Editor)
	assert.Equal(t, "bash", em.UserPrefs.Shell)
	assert.Equal(t, 60, em.UserPrefs.TimeoutSeconds)
	assert.False(t, em.UserPrefs.AutoApproveLow)
	assert.False(t, em.UserPrefs.PreferSudo)
}

// === RecordServer / GetServer / ServerCount / RemoveServer 测试 ===

func TestRecordServer_NewServer(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.RecordServer("admin@web01:2222", map[string]string{"os": "linux"})

	assert.Equal(t, 1, em.ServerCount())
	server := em.GetServer("web01")
	assert.NotNil(t, server)
	assert.Equal(t, "web01", server.Host)
	assert.Equal(t, "admin", server.DefaultUser)
	assert.Equal(t, "linux", server.OS)
	assert.Equal(t, 1, server.UseCount)
}

func TestRecordServer_UpdateExisting(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.RecordServer("root@db01", map[string]string{"os": "ubuntu"})
	em.RecordServer("root@db01", map[string]string{"os": "centos", "env": "production"})

	assert.Equal(t, 1, em.ServerCount())
	server := em.GetServer("db01")
	assert.NotNil(t, server)
	assert.Equal(t, "centos", server.OS)
	assert.Equal(t, "production", server.CustomInfo["env"])
	assert.Equal(t, 2, server.UseCount)
}

func TestRecordServer_AppendCommonPaths(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.RecordServer("root@app01", map[string]string{"path": "/var/log"})
	em.RecordServer("root@app01", map[string]string{"path": "/opt/data"})

	server := em.GetServer("app01")
	assert.NotNil(t, server)
	assert.Contains(t, server.CommonPaths, "/var/log")
	assert.Contains(t, server.CommonPaths, "/opt/data")
}

func TestRecordServer_NonDefaultPort(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.RecordServer("root@host1:3333", map[string]string{})

	server := em.GetServer("host1")
	assert.NotNil(t, server)
	assert.Equal(t, "3333", server.CustomInfo["port"])
}

func TestRecordServer_DefaultPortNotStored(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.RecordServer("root@host1:22", map[string]string{})

	server := em.GetServer("host1")
	assert.NotNil(t, server)
	_, hasPort := server.CustomInfo["port"]
	assert.False(t, hasPort)
}

func TestGetServer_NotFound(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	server := em.GetServer("nonexistent")
	assert.Nil(t, server)
}

func TestRemoveServer_Success(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.RecordServer("root@srv1", map[string]string{})
	assert.Equal(t, 1, em.ServerCount())

	result := em.RemoveServer("srv1")
	assert.True(t, result)
	assert.Equal(t, 0, em.ServerCount())
}

func TestRemoveServer_NotFound(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	result := em.RemoveServer("nonexistent")
	assert.False(t, result)
}

func TestServerCount_Empty(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	assert.Equal(t, 0, em.ServerCount())
}

func TestServerCount_Multiple(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.RecordServer("root@srv1", map[string]string{})
	em.RecordServer("root@srv2", map[string]string{})
	em.RecordServer("admin@srv3", map[string]string{})
	assert.Equal(t, 3, em.ServerCount())
}

// === GetRecentServers 测试 ===

func TestGetRecentServers_SortedByLastUsed(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.RecordServer("root@old", map[string]string{})
	// Small sleep to ensure different timestamps
	em.RecordServer("root@new", map[string]string{})

	servers := em.GetRecentServers(10)
	assert.Len(t, servers, 2)
	// Most recent should be first
	assert.Equal(t, "new", servers[0].Host)
	assert.Equal(t, "old", servers[1].Host)
}

func TestGetRecentServers_Limit(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.RecordServer("root@srv1", map[string]string{})
	em.RecordServer("root@srv2", map[string]string{})
	em.RecordServer("root@srv3", map[string]string{})

	servers := em.GetRecentServers(2)
	assert.Len(t, servers, 2)
}

func TestGetRecentServers_ZeroOrNegative(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.RecordServer("root@srv1", map[string]string{})

	servers := em.GetRecentServers(0)
	assert.Len(t, servers, 1)
}

// === GetServerSuggestions 测试 ===

func TestGetServerSuggestions_ByFullPrefix(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.RecordServer("root@web01", map[string]string{})
	em.RecordServer("root@db01", map[string]string{})
	em.RecordServer("admin@api01", map[string]string{})

	suggestions := em.GetServerSuggestions("root@")
	sort.Strings(suggestions)
	assert.Equal(t, []string{"root@db01", "root@web01"}, suggestions)
}

func TestGetServerSuggestions_ByHostPrefix(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.RecordServer("root@web01", map[string]string{})
	em.RecordServer("root@web02", map[string]string{})
	em.RecordServer("root@db01", map[string]string{})

	suggestions := em.GetServerSuggestions("web")
	sort.Strings(suggestions)
	assert.Equal(t, []string{"root@web01", "root@web02"}, suggestions)
}

func TestGetServerSuggestions_NoMatch(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.RecordServer("root@web01", map[string]string{})

	suggestions := em.GetServerSuggestions("db")
	assert.Empty(t, suggestions)
}

// === UpdateLastQueries / GetSimilarQueries 测试 ===

func TestUpdateLastQueries_AddNew(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.UpdateLastQueries("查看磁盘")
	em.UpdateLastQueries("查看内存")

	assert.Len(t, em.LastQueries, 2)
}

func TestUpdateLastQueries_DedupMovesToEnd(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.UpdateLastQueries("查询A")
	em.UpdateLastQueries("查询B")
	em.UpdateLastQueries("查询A") // 重复，应移到末尾

	assert.Len(t, em.LastQueries, 2)
	assert.Equal(t, "查询B", em.LastQueries[0])
	assert.Equal(t, "查询A", em.LastQueries[1])
}

func TestUpdateLastQueries_Max50(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	for i := 0; i < 55; i++ {
		em.UpdateLastQueries("查询" + string(rune('0'+i%10)))
	}
	assert.True(t, len(em.LastQueries) <= 50)
}

func TestGetSimilarQueries_WithMatch(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.UpdateLastQueries("查看磁盘使用情况")
	em.UpdateLastQueries("查看内存使用情况")
	em.UpdateLastQueries("重启nginx")

	results := em.GetSimilarQueries("查看磁盘空间", 5)
	assert.NotEmpty(t, results)
}

func TestGetSimilarQueries_NoMatch(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.UpdateLastQueries("查看磁盘空间")

	results := em.GetSimilarQueries("完全不同的查询", 5)
	// "完全不同的查询" 和 "查看磁盘空间" 没有共享词
	assert.Empty(t, results)
}

func TestGetSimilarQueries_ExcludesSelf(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.UpdateLastQueries("查看磁盘空间")

	results := em.GetSimilarQueries("查看磁盘空间", 5)
	// Should not include itself
	for _, r := range results {
		assert.NotEqual(t, "查看磁盘空间", r)
	}
}

// === UserPreference 测试 ===

func TestSetGetUserPreference_SafetyMode(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.SetUserPreference("safety_mode", "strict")
	assert.Equal(t, "strict", em.GetUserPreference("safety_mode"))
}

func TestSetGetUserPreference_AutoApproveLow(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.SetUserPreference("auto_approve_low", "true")
	assert.Equal(t, "true", em.GetUserPreference("auto_approve_low"))

	em.SetUserPreference("auto_approve_low", "false")
	assert.Equal(t, "false", em.GetUserPreference("auto_approve_low"))
}

func TestSetGetUserPreference_PreferSudo(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.SetUserPreference("prefer_sudo", "true")
	assert.Equal(t, "true", em.GetUserPreference("prefer_sudo"))
}

func TestSetGetUserPreference_Timeout(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.SetUserPreference("timeout", "120")
	assert.Equal(t, "120", em.GetUserPreference("timeout"))
}

func TestSetGetUserPreference_TimeoutInvalid(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	// Invalid timeout should not change the default
	em.SetUserPreference("timeout", "notanumber")
	assert.Equal(t, "60", em.GetUserPreference("timeout"))
}

func TestSetGetUserPreference_Editor(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.SetUserPreference("editor", "nano")
	assert.Equal(t, "nano", em.GetUserPreference("editor"))
}

func TestSetGetUserPreference_Shell(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.SetUserPreference("shell", "zsh")
	assert.Equal(t, "zsh", em.GetUserPreference("shell"))
}

func TestGetUserPreference_UnknownKey(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	assert.Equal(t, "", em.GetUserPreference("nonexistent_key"))
}

// === CustomHints 测试 ===

func TestCustomHints_SetAndGet(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.AddCustomHint("磁盘分析", "先检查 /var/log 大小")
	assert.Equal(t, "先检查 /var/log 大小", em.GetCustomHint("磁盘分析"))
}

func TestCustomHints_NotSet(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	assert.Equal(t, "", em.GetCustomHint("nonexistent"))
}

func TestCustomHints_Overwrite(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.AddCustomHint("test", "hint1")
	em.AddCustomHint("test", "hint2")
	assert.Equal(t, "hint2", em.GetCustomHint("test"))
}

// === Save/Load 测试 ===

func TestEnvironmentMemory_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	em := NewEnvironmentMemory(dir)
	em.RecordServer("root@web01", map[string]string{"os": "linux"})
	em.SetUserPreference("editor", "nano")
	em.AddCustomHint("test", "hint value")
	em.UpdateLastQueries("测试查询")

	// Save
	err := em.Save()
	assert.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(filepath.Join(dir, "environment.json"))
	assert.NoError(t, err)

	// Load
	loaded, err := LoadEnvironmentMemory(dir)
	assert.NoError(t, assert.NotEmpty(t, loaded))

	assert.Equal(t, 1, loaded.ServerCount())
	assert.Equal(t, "nano", loaded.GetUserPreference("editor"))
	assert.Equal(t, "hint value", loaded.GetCustomHint("test"))
}

func TestLoadEnvironmentMemory_NoFile(t *testing.T) {
	dir := t.TempDir()
	em, err := LoadEnvironmentMemory(dir)
	assert.NoError(t, err)
	assert.NotNil(t, em)
	assert.Equal(t, 0, em.ServerCount())
}

func TestLoadEnvironmentMemory_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "environment.json"), []byte("{invalid json"), 0600)
	assert.NoError(t, err)

	em, err := LoadEnvironmentMemory(dir)
	assert.Error(t, err)
	assert.Nil(t, em)
}

func TestEnvironmentMemory_Save_NilFields(t *testing.T) {
	dir := t.TempDir()
	em := NewEnvironmentMemory(dir)
	// Simulate loading from a file with nil fields
	em.LastQueries = nil
	em.CustomHints = nil
	em.KnownServers = nil

	loaded, err := LoadEnvironmentMemory(dir)
	// Should still create new memory since no file exists
	_ = loaded
	// The save itself shouldn't panic
	err = em.Save()
	assert.NoError(t, err)
}

// === 并发安全测试 ===

func TestEnvironmentMemory_ConcurrentAccess(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(n int) {
			em.RecordServer("root@host"+string(rune('A'+n)), map[string]string{"os": "linux"})
			em.UpdateLastQueries("查询" + string(rune('A'+n)))
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			_ = em.ServerCount()
			_ = em.GetServer("host0")
			_ = em.GetRecentServers(5)
			_ = em.GetServerSuggestions("root@")
			_ = em.GetUserPreference("editor")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	assert.Equal(t, 10, em.ServerCount())
}

// === GetServerSuggestions 空 KnownServers ===

func TestGetServerSuggestions_Empty(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	suggestions := em.GetServerSuggestions("root@")
	assert.Empty(t, suggestions)
}

// === RecordServer with empty info ===

func TestRecordServer_EmptyInfo(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())
	em.RecordServer("simple-host", nil)
	assert.Equal(t, 1, em.ServerCount())
	server := em.GetServer("simple-host")
	assert.NotNil(t, server)
	assert.Equal(t, 1, server.UseCount)
}

// === 时间排序精确验证 ===

func TestGetRecentServers_PreciseOrder(t *testing.T) {
	em := NewEnvironmentMemory(t.TempDir())

	em.RecordServer("root@first", map[string]string{})
	time.Sleep(10 * time.Millisecond)
	em.RecordServer("root@second", map[string]string{})
	time.Sleep(10 * time.Millisecond)
	em.RecordServer("root@third", map[string]string{})

	servers := em.GetRecentServers(3)
	assert.Equal(t, "third", servers[0].Host)
	assert.Equal(t, "second", servers[1].Host)
	assert.Equal(t, "first", servers[2].Host)
}
