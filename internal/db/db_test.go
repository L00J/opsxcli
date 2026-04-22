package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDB 辅助函数：创建临时测试数据库
func testDB(t *testing.T) *DB {
	t.Helper()
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err, "创建测试数据库不应失败")
	t.Cleanup(func() { db.Close() })
	return db
}

// --- DB 创建与连接 ---

func TestNewDB_创建成功(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// 验证数据库文件存在
	dbPath := filepath.Join(tmpDir, "opsxcli.db")
	_, err = os.Stat(dbPath)
	assert.NoError(t, err, "数据库文件应存在")
}

func TestNewDB_自动创建目录(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "deep", "nested", "dir")
	db, err := NewDB(nestedDir)
	require.NoError(t, err)
	defer db.Close()

	// 验证嵌套目录已创建
	_, err = os.Stat(nestedDir)
	assert.NoError(t, err, "嵌套数据目录应自动创建")
}

func TestNewDB_Schema初始化(t *testing.T) {
	db := testDB(t)

	// 验证关键表已创建
	tables := []string{"users", "sessions", "messages", "audit_logs", "configs", "memories", "backups"}
	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		assert.NoError(t, err, "表 %s 应存在且可查询", table)
	}
}

func TestNewDB_默认数据(t *testing.T) {
	db := testDB(t)

	// 验证默认管理员已插入
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "应有默认admin用户")

	// 验证默认配置已插入
	err = db.QueryRow("SELECT COUNT(*) FROM configs").Scan(&count)
	assert.NoError(t, err)
	assert.Greater(t, count, 0, "应有默认配置项")
}

func TestNewDB_连接可Ping(t *testing.T) {
	db := testDB(t)
	err := db.Ping()
	assert.NoError(t, err, "数据库应可Ping")
}

func TestGetDefaultDataDir(t *testing.T) {
	dir, err := GetDefaultDataDir()
	require.NoError(t, err)
	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".opsxcli", "data")
	assert.Equal(t, expected, dir)
}

// --- Session CRUD ---

func TestSessionRepository_CreateAndGet(t *testing.T) {
	db := testDB(t)
	repo := NewSessionRepository(db)

	// 先创建用户（外键约束）
	userRepo := NewUserRepository(db)
	user := &User{Username: "testuser", PasswordHash: "hash", Role: "operator"}
	require.NoError(t, userRepo.Create(user))

	session := &Session{
		UserID:   user.ID,
		Title:    "测试会话",
		Provider: "ollama",
		Model:    "qwen3:14b",
	}
	err := repo.Create(session)
	require.NoError(t, err)
	assert.NotEmpty(t, session.ID, "会话ID应自动生成")
	assert.False(t, session.CreatedAt.IsZero(), "CreatedAt应被设置")

	// 通过ID获取
	got, err := repo.GetByID(session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.Title, got.Title)
	assert.Equal(t, session.Provider, got.Provider)
	assert.Equal(t, session.Model, got.Model)
}

func TestSessionRepository_Create指定ID(t *testing.T) {
	db := testDB(t)
	repo := NewSessionRepository(db)

	userRepo := NewUserRepository(db)
	user := &User{Username: "uid_test", PasswordHash: "hash", Role: "operator"}
	require.NoError(t, userRepo.Create(user))

	session := &Session{
		ID:       "custom_id_123",
		UserID:   user.ID,
		Title:    "自定义ID",
		Provider: "glm",
		Model:    "glm-4",
	}
	err := repo.Create(session)
	require.NoError(t, err)
	assert.Equal(t, "custom_id_123", session.ID)
}

func TestSessionRepository_GetByID不存在(t *testing.T) {
	db := testDB(t)
	repo := NewSessionRepository(db)

	_, err := repo.GetByID("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "会话不存在")
}

func TestSessionRepository_GetByUser(t *testing.T) {
	db := testDB(t)
	repo := NewSessionRepository(db)
	userRepo := NewUserRepository(db)

	user := &User{Username: "listuser", PasswordHash: "hash", Role: "operator"}
	require.NoError(t, userRepo.Create(user))

	// 创建3个会话
	titles := []string{"会话A", "会话B", "会话C"}
	for _, title := range titles {
		session := &Session{
			UserID:   user.ID,
			Title:    title,
			Provider: "ollama",
			Model:    "qwen3:14b",
		}
		require.NoError(t, repo.Create(session))
	}

	// 获取用户会话
	sessions, err := repo.GetByUser(user.ID, 10)
	require.NoError(t, err)
	assert.Len(t, sessions, 3)

	// 限制数量
	sessions, err = repo.GetByUser(user.ID, 2)
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
}

func TestSessionRepository_Update(t *testing.T) {
	db := testDB(t)
	repo := NewSessionRepository(db)
	userRepo := NewUserRepository(db)

	user := &User{Username: "upduser", PasswordHash: "hash", Role: "operator"}
	require.NoError(t, userRepo.Create(user))

	session := &Session{UserID: user.ID, Title: "旧标题", Provider: "ollama", Model: "qwen3:14b"}
	require.NoError(t, repo.Create(session))

	session.Title = "新标题"
	err := repo.Update(session)
	require.NoError(t, err)

	got, err := repo.GetByID(session.ID)
	require.NoError(t, err)
	assert.Equal(t, "新标题", got.Title)
}

func TestSessionRepository_Delete(t *testing.T) {
	db := testDB(t)
	repo := NewSessionRepository(db)
	userRepo := NewUserRepository(db)

	user := &User{Username: "deluser", PasswordHash: "hash", Role: "operator"}
	require.NoError(t, userRepo.Create(user))

	session := &Session{UserID: user.ID, Title: "待删除", Provider: "ollama", Model: "qwen3:14b"}
	require.NoError(t, repo.Create(session))

	err := repo.Delete(session.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(session.ID)
	assert.Error(t, err, "删除后应无法获取")
}

// --- Message CRUD ---

func TestMessageRepository_CreateAndGet(t *testing.T) {
	db := testDB(t)
	sessRepo := NewSessionRepository(db)
	msgRepo := NewMessageRepository(db)
	userRepo := NewUserRepository(db)

	user := &User{Username: "msguser", PasswordHash: "hash", Role: "operator"}
	require.NoError(t, userRepo.Create(user))

	session := &Session{UserID: user.ID, Title: "消息测试", Provider: "ollama", Model: "qwen3:14b"}
	require.NoError(t, sessRepo.Create(session))

	msg := &Message{
		SessionID: session.ID,
		Role:      "user",
		Content:   "你好，请帮我检查服务器状态",
	}
	err := msgRepo.Create(msg)
	require.NoError(t, err)
	assert.Greater(t, msg.ID, int64(0), "消息ID应被设置")
	assert.False(t, msg.CreatedAt.IsZero(), "CreatedAt应被设置")
}

func TestMessageRepository_GetBySession(t *testing.T) {
	db := testDB(t)
	sessRepo := NewSessionRepository(db)
	msgRepo := NewMessageRepository(db)
	userRepo := NewUserRepository(db)

	user := &User{Username: "msgsuser", PasswordHash: "hash", Role: "operator"}
	require.NoError(t, userRepo.Create(user))

	session := &Session{UserID: user.ID, Title: "多消息测试", Provider: "ollama", Model: "qwen3:14b"}
	require.NoError(t, sessRepo.Create(session))

	// 创建多条消息
	roles := []string{"user", "assistant", "tool"}
	for i, role := range roles {
		msg := &Message{
			SessionID: session.ID,
			Role:      role,
			Content:   "消息内容",
		}
		require.NoError(t, msgRepo.Create(msg), "第%d条消息创建失败", i+1)
	}

	messages, err := msgRepo.GetBySession(session.ID)
	require.NoError(t, err)
	assert.Len(t, messages, 3)

	// 验证按时间升序
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "assistant", messages[1].Role)
	assert.Equal(t, "tool", messages[2].Role)
}

func TestMessageRepository_ToolCalls和Metadata(t *testing.T) {
	db := testDB(t)
	sessRepo := NewSessionRepository(db)
	msgRepo := NewMessageRepository(db)
	userRepo := NewUserRepository(db)

	user := &User{Username: "tooluser", PasswordHash: "hash", Role: "operator"}
	require.NoError(t, userRepo.Create(user))

	session := &Session{UserID: user.ID, Title: "工具测试", Provider: "ollama", Model: "qwen3:14b"}
	require.NoError(t, sessRepo.Create(session))

	msg := &Message{
		SessionID: session.ID,
		Role:      "assistant",
		Content:   "我来执行命令",
		ToolCalls: `[{"name":"local_bash","arguments":{"command":"ls -la"}}]`,
		Metadata:  `{"tokens":150}`,
	}
	require.NoError(t, msgRepo.Create(msg))

	messages, err := msgRepo.GetBySession(session.ID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, msg.ToolCalls, messages[0].ToolCalls)
	assert.Equal(t, msg.Metadata, messages[0].Metadata)
}

func TestMessageRepository_GetBySession空结果(t *testing.T) {
	db := testDB(t)
	msgRepo := NewMessageRepository(db)

	messages, err := msgRepo.GetBySession("nonexistent_session")
	require.NoError(t, err)
	assert.Empty(t, messages)
}

// --- AuditLog CRUD ---

func TestAuditLogRepository_Create(t *testing.T) {
	db := testDB(t)
	repo := NewAuditLogRepository(db)

	log := &AuditLog{
		UserID:      1,
		Username:    "admin",
		ActionType:  "command",
		Command:     "rm -rf /tmp/test",
		RiskLevel:   "high",
		Environment: "production",
	}
	err := repo.Create(log)
	require.NoError(t, err)
	assert.Greater(t, log.ID, int64(0), "审计日志ID应被设置")
}

func TestAuditLogRepository_MarkApproved(t *testing.T) {
	db := testDB(t)
	repo := NewAuditLogRepository(db)

	log := &AuditLog{
		UserID:      1,
		Username:    "admin",
		ActionType:  "command",
		Command:     "shutdown -r now",
		RiskLevel:   "critical",
		Environment: "production",
	}
	require.NoError(t, repo.Create(log))

	err := repo.MarkApproved(log.ID, "supervisor")
	require.NoError(t, err)

	// 验证已标记为已审批
	logs, err := repo.List(10, 0, nil)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.True(t, logs[0].Approved)
	assert.Equal(t, "supervisor", *logs[0].ApprovedBy)
}

func TestAuditLogRepository_MarkExecuted(t *testing.T) {
	db := testDB(t)
	repo := NewAuditLogRepository(db)

	log := &AuditLog{
		UserID:      1,
		Username:    "admin",
		ActionType:  "command",
		Command:     "ls -la",
		RiskLevel:   "safe",
		Environment: "dev",
	}
	require.NoError(t, repo.Create(log))

	err := repo.MarkExecuted(log.ID, true, "total 32\ndrwxr-xr-x", "")
	require.NoError(t, err)

	logs, err := repo.List(10, 0, nil)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.True(t, logs[0].Executed)
	assert.True(t, *logs[0].Success)
	assert.Equal(t, "total 32\ndrwxr-xr-x", logs[0].Output)
}

func TestAuditLogRepository_List过滤(t *testing.T) {
	db := testDB(t)
	repo := NewAuditLogRepository(db)

	// 创建不同风险级别的日志
	riskLevels := []string{"safe", "low", "medium", "high", "critical"}
	for _, risk := range riskLevels {
		log := &AuditLog{
			UserID:      1,
			Username:    "admin",
			ActionType:  "command",
			Command:     "test_" + risk,
			RiskLevel:   risk,
			Environment: "dev",
		}
		require.NoError(t, repo.Create(log))
	}

	// 不带过滤
	all, err := repo.List(10, 0, nil)
	require.NoError(t, err)
	assert.Len(t, all, 5)

	// 按风险级别过滤
	highOnly, err := repo.List(10, 0, map[string]interface{}{"risk_level": "high"})
	require.NoError(t, err)
	assert.Len(t, highOnly, 1)
	assert.Equal(t, "high", highOnly[0].RiskLevel)

	// 分页
	page1, err := repo.List(2, 0, nil)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	page2, err := repo.List(2, 2, nil)
	require.NoError(t, err)
	assert.Len(t, page2, 2)
}

func TestAuditLogRepository_GetStats(t *testing.T) {
	db := testDB(t)
	repo := NewAuditLogRepository(db)

	// 创建多条审计日志
	for i := 0; i < 5; i++ {
		risk := "safe"
		if i%2 == 0 {
			risk = "high"
		}
		log := &AuditLog{
			UserID:      1,
			Username:    "admin",
			ActionType:  "command",
			Command:     "test_cmd",
			RiskLevel:   risk,
			Environment: "dev",
		}
		require.NoError(t, repo.Create(log))

		// 标记部分为已执行
		if i < 3 {
			require.NoError(t, repo.MarkExecuted(log.ID, i < 2, "output", ""))
		}
	}

	stats, err := repo.GetStats(nil, 999999*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(5), stats["total"], "总计应为5条")
	// high: i=0,2,4 => 3条
	assert.Equal(t, int64(3), stats["dangerous"], "高危应为3条")
	// executed: 前3条标记了 => 3条
	assert.Equal(t, int64(3), stats["executed"], "已执行应为3条")
}

// --- User CRUD ---

func TestUserRepository_CreateAndGet(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)

	user := &User{
		Username:     "newuser",
		PasswordHash: "hashed_password",
		Role:         "operator",
		Email:        "test@example.com",
	}
	err := repo.Create(user)
	require.NoError(t, err)
	assert.Greater(t, user.ID, int64(0), "用户ID应被设置")

	// 通过用户名获取
	got, err := repo.GetByUsername("newuser")
	require.NoError(t, err)
	assert.Equal(t, "newuser", got.Username)
	assert.Equal(t, "operator", got.Role)
	assert.Equal(t, "test@example.com", got.Email)

	// 通过ID获取
	gotByID, err := repo.GetByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.Username, gotByID.Username)
}

func TestUserRepository_GetByUsername不存在(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)

	_, err := repo.GetByUsername("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "用户不存在")
}

func TestUserRepository_GetByID不存在(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)

	_, err := repo.GetByID(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "用户不存在")
}

func TestUserRepository_UpdateLastLogin(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)

	user := &User{Username: "loginuser", PasswordHash: "hash", Role: "operator"}
	require.NoError(t, repo.Create(user))

	err := repo.UpdateLastLogin(user.ID)
	require.NoError(t, err)

	got, err := repo.GetByID(user.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.LastLogin, "最后登录时间应已更新")
}

func TestUserRepository_List(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)

	// 已有默认admin，再添加两个
	for _, name := range []string{"user1", "user2"} {
		user := &User{Username: name, PasswordHash: "hash", Role: "operator"}
		require.NoError(t, repo.Create(user))
	}

	users, err := repo.List()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 3, "应有至少3个用户(1 admin + 2 新)")
}

func TestUserRepository_用户名唯一约束(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)

	user1 := &User{Username: "dupuser", PasswordHash: "hash1", Role: "operator"}
	require.NoError(t, repo.Create(user1))

	user2 := &User{Username: "dupuser", PasswordHash: "hash2", Role: "admin"}
	err := repo.Create(user2)
	assert.Error(t, err, "重复用户名应报错")
}

// --- ApplyDefaults ---

func TestApplyDefaults_AllProvided(t *testing.T) {
	host, port, user := ApplyDefaults("myhost", 3306, "root", "mysql")
	assert.Equal(t, "myhost", host)
	assert.Equal(t, 3306, port)
	assert.Equal(t, "root", user)
}

func TestApplyDefaults_MySQLDefaults(t *testing.T) {
	host, port, user := ApplyDefaults("", 0, "", "mysql")
	// Falls back to config defaults for mysql
	assert.NotEmpty(t, host, "should have a default mysql host")
	assert.NotZero(t, port, "should have a default mysql port")
	_ = user
}

func TestApplyDefaults_PostgresDefaults(t *testing.T) {
	host, port, user := ApplyDefaults("", 0, "", "postgres")
	assert.NotEmpty(t, host)
	assert.NotZero(t, port)
	_ = user
}

func TestApplyDefaults_EmptyDBType(t *testing.T) {
	host, port, user := ApplyDefaults("host", 5432, "", "")
	assert.Equal(t, "host", host)
	assert.Equal(t, 5432, port)
	assert.Equal(t, "", user)
}
