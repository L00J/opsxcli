package db

import (
	"database/sql"
	"fmt"
	"time"
)

// AuditLog 审计日志模型
type AuditLog struct {
	ID          int64      `json:"id"`
	SessionID   *string    `json:"session_id,omitempty"`
	UserID      int64      `json:"user_id"`
	Username    string     `json:"username"`
	ActionType  string     `json:"action_type"`
	Command     string     `json:"command"`
	Arguments   string     `json:"arguments,omitempty"`
	RiskLevel   string     `json:"risk_level"`
	Approved    bool       `json:"approved"`
	ApprovedBy  *string    `json:"approved_by,omitempty"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	Executed    bool       `json:"executed"`
	ExecutedAt  *time.Time `json:"executed_at,omitempty"`
	Success     *bool      `json:"success,omitempty"`
	Output      string     `json:"output,omitempty"`
	Error       string     `json:"error,omitempty"`
	Environment string     `json:"environment"`
	IPAddress   string     `json:"ip_address,omitempty"`
	UserAgent   string     `json:"user_agent,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// AuditLogRepository 审计日志仓储
type AuditLogRepository struct {
	db *DB
}

// NewAuditLogRepository 创建审计日志仓储
func NewAuditLogRepository(db *DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create 创建审计日志
func (r *AuditLogRepository) Create(log *AuditLog) error {
	result, err := r.db.Exec(`
		INSERT INTO audit_logs (
			session_id, user_id, username, action_type, command, arguments,
			risk_level, environment, ip_address, user_agent
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		log.SessionID, log.UserID, log.Username, log.ActionType, log.Command, log.Arguments,
		log.RiskLevel, log.Environment, log.IPAddress, log.UserAgent,
	)

	if err != nil {
		return fmt.Errorf("创建审计日志失败: %w", err)
	}

	id, _ := result.LastInsertId()
	log.ID = id
	return nil
}

// MarkApproved 标记为已批准
func (r *AuditLogRepository) MarkApproved(id int64, approvedBy string) error {
	_, err := r.db.Exec(`
		UPDATE audit_logs
		SET approved = 1, approved_by = ?, approved_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, approvedBy, id)
	return err
}

// MarkExecuted 标记为已执行
func (r *AuditLogRepository) MarkExecuted(id int64, success bool, output, errMsg string) error {
	_, err := r.db.Exec(`
		UPDATE audit_logs
		SET executed = 1, executed_at = CURRENT_TIMESTAMP, success = ?, output = ?, error = ?
		WHERE id = ?
	`, success, output, errMsg, id)
	return err
}

// List 列出审计日志
func (r *AuditLogRepository) List(limit, offset int, filters map[string]interface{}) ([]*AuditLog, error) {
	query := `
		SELECT id, session_id, user_id, username, action_type, command,
		       COALESCE(arguments, ''), risk_level, approved,
		       approved_by, approved_at, executed, executed_at,
		       success, COALESCE(output, ''), COALESCE(error, ''),
		       environment, COALESCE(ip_address, ''), COALESCE(user_agent, ''), created_at
		FROM audit_logs
		WHERE 1=1
	`

	args := []interface{}{}

	// 添加过滤条件
	if userID, ok := filters["user_id"].(int64); ok {
		query += " AND user_id = ?"
		args = append(args, userID)
	}
	if riskLevel, ok := filters["risk_level"].(string); ok {
		query += " AND risk_level = ?"
		args = append(args, riskLevel)
	}
	if executed, ok := filters["executed"].(bool); ok {
		query += " AND executed = ?"
		args = append(args, executed)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		log := &AuditLog{}
		err := rows.Scan(
			&log.ID, &log.SessionID, &log.UserID, &log.Username, &log.ActionType,
			&log.Command, &log.Arguments, &log.RiskLevel, &log.Approved,
			&log.ApprovedBy, &log.ApprovedAt, &log.Executed, &log.ExecutedAt,
			&log.Success, &log.Output, &log.Error, &log.Environment,
			&log.IPAddress, &log.UserAgent, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// GetStats 获取统计信息
func (r *AuditLogRepository) GetStats(userID *int64, timeRange time.Duration) (map[string]int64, error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN risk_level = 'high' OR risk_level = 'critical' THEN 1 ELSE 0 END) as dangerous,
			SUM(CASE WHEN executed = 1 THEN 1 ELSE 0 END) as executed,
			SUM(CASE WHEN executed = 1 AND success = 1 THEN 1 ELSE 0 END) as successful
		FROM audit_logs
		WHERE created_at >= datetime('now', ?)
	`

	args := []interface{}{fmt.Sprintf("-%d seconds", int(timeRange.Seconds()))}

	if userID != nil {
		query += " AND user_id = ?"
		args = append(args, *userID)
	}

	stats := make(map[string]int64)
	var total, dangerous, executed, successful sql.NullInt64

	err := r.db.QueryRow(query, args...).Scan(&total, &dangerous, &executed, &successful)
	if err != nil {
		return nil, err
	}

	stats["total"] = total.Int64
	stats["dangerous"] = dangerous.Int64
	stats["executed"] = executed.Int64
	stats["successful"] = successful.Int64

	return stats, nil
}
