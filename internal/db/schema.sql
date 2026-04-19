-- opsxcli 数据库 Schema

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT DEFAULT 'operator' CHECK(role IN ('admin', 'operator', 'viewer')),
    email TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP
);

-- 会话表
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title TEXT,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 消息表
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('user', 'assistant', 'system', 'tool')),
    content TEXT NOT NULL,
    tool_calls TEXT,
    metadata TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- 审计日志表
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT,
    user_id INTEGER NOT NULL,
    username TEXT NOT NULL,
    action_type TEXT NOT NULL,
    command TEXT NOT NULL,
    arguments TEXT,
    risk_level TEXT NOT NULL CHECK(risk_level IN ('safe', 'low', 'medium', 'high', 'critical')),
    approved BOOLEAN DEFAULT 0,
    approved_by TEXT,
    approved_at TIMESTAMP,
    executed BOOLEAN DEFAULT 0,
    executed_at TIMESTAMP,
    success BOOLEAN,
    output TEXT,
    error TEXT,
    environment TEXT DEFAULT 'unknown',
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE SET NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 备份表
CREATE TABLE IF NOT EXISTS backups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    audit_log_id INTEGER NOT NULL,
    original_path TEXT NOT NULL,
    backup_path TEXT NOT NULL,
    file_size INTEGER,
    checksum TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    restored BOOLEAN DEFAULT 0,
    FOREIGN KEY (audit_log_id) REFERENCES audit_logs(id) ON DELETE CASCADE
);

-- 配置表
CREATE TABLE IF NOT EXISTS configs (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    category TEXT,
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 记忆表
CREATE TABLE IF NOT EXISTS memories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL CHECK(type IN ('short_term', 'long_term')),
    session_id TEXT,
    category TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    metadata TEXT,
    access_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    UNIQUE(type, category, key),
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_updated ON sessions(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_time ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_risk ON audit_logs(risk_level);
CREATE INDEX IF NOT EXISTS idx_audit_executed ON audit_logs(executed);
CREATE INDEX IF NOT EXISTS idx_backups_audit ON backups(audit_log_id);
CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);
CREATE INDEX IF NOT EXISTS idx_memories_category ON memories(category);
CREATE INDEX IF NOT EXISTS idx_memories_session ON memories(session_id);
CREATE INDEX IF NOT EXISTS idx_memories_expires ON memories(expires_at);

-- 插入默认管理员账号 (密码: admin，需要首次登录后修改)
-- bcrypt hash of "admin"
INSERT OR IGNORE INTO users (id, username, password_hash, role, email)
VALUES (1, 'admin', '$2a$10$TJyxkASjPLx2f3e0UkeOeOEJieh4wL2HKEDGvmIp8GQa6Ol7HtLOy', 'admin', 'admin@opsxcli.local');

-- 插入默认配置
INSERT OR IGNORE INTO configs (key, value, category, description) VALUES
('safety.mode', 'strict', 'safety', '安全模式: strict, balanced, permissive'),
('safety.auto_backup', 'true', 'safety', '自动备份危险操作涉及的文件'),
('safety.backup_retention_days', '7', 'safety', '备份文件保留天数'),
('llm.default_provider', 'ollama', 'llm', '默认 LLM 提供商'),
	('llm.default_model', 'qwen3:14b', 'llm', '默认模型'),
('server.jwt_secret', '', 'server', 'JWT 密钥 (自动生成)'),
('server.session_duration', '24h', 'server', 'JWT Token 有效期');
