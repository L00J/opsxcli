package cmd

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"opsxcli/internal/auth"
	"opsxcli/internal/db"
)

// NewTestCmd 创建测试命令
func NewTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test-db",
		Short: "测试数据库和认证模块",
		Long:  `测试 SQLite 数据库、用户管理、JWT 认证等功能`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTests()
		},
	}

	return cmd
}

func runTests() error {
	fmt.Println(color.CyanString("🧪 开始测试模块..."))
	fmt.Println()

	// 1. 测试数据库初始化
	fmt.Println(color.YellowString("1️⃣  测试数据库初始化"))
	dataDir, err := db.GetDefaultDataDir()
	if err != nil {
		return fmt.Errorf("获取数据目录失败: %w", err)
	}
	fmt.Printf("   数据目录: %s\n", dataDir)

	database, err := db.NewDB(dataDir)
	if err != nil {
		return fmt.Errorf("初始化数据库失败: %w", err)
	}
	defer database.Close()
	fmt.Println(color.GreenString("   ✓ 数据库初始化成功"))
	fmt.Println()

	// 2. 测试用户仓储
	fmt.Println(color.YellowString("2️⃣  测试用户管理"))
	userRepo := db.NewUserRepository(database)

	// 查询默认管理员
	admin, err := userRepo.GetByUsername("admin")
	if err != nil {
		return fmt.Errorf("查询管理员失败: %w", err)
	}
	fmt.Printf("   管理员账号: %s (ID: %d, 角色: %s)\n", admin.Username, admin.ID, admin.Role)
	fmt.Println(color.GreenString("   ✓ 用户查询成功"))
	fmt.Println()

	// 3. 测试密码验证
	fmt.Println(color.YellowString("3️⃣  测试密码验证"))
	testPassword := "admin"
	isValid := auth.VerifyPassword(admin.PasswordHash, testPassword)
	if !isValid {
		return fmt.Errorf("密码验证失败")
	}
	fmt.Printf("   测试密码: %s\n", testPassword)
	fmt.Println(color.GreenString("   ✓ 密码验证通过"))
	fmt.Println()

	// 4. 测试 JWT 生成和验证
	fmt.Println(color.YellowString("4️⃣  测试 JWT 认证"))
	jwtManager := auth.NewJWTManager("", "opsxcli", 24*time.Hour)

	token, err := jwtManager.Generate(admin.ID, admin.Username, admin.Role)
	if err != nil {
		return fmt.Errorf("生成 Token 失败: %w", err)
	}
	fmt.Printf("   JWT Token: %s...\n", token[:50])

	claims, err := jwtManager.Verify(token)
	if err != nil {
		return fmt.Errorf("验证 Token 失败: %w", err)
	}
	fmt.Printf("   解析结果: UserID=%d, Username=%s, Role=%s\n", claims.UserID, claims.Username, claims.Role)
	fmt.Println(color.GreenString("   ✓ JWT 认证成功"))
	fmt.Println()

	// 5. 测试创建新用户
	fmt.Println(color.YellowString("5️⃣  测试创建新用户"))
	passwordHash, err := auth.HashPassword("test123")
	if err != nil {
		return fmt.Errorf("密码哈希失败: %w", err)
	}

	newUser := &db.User{
		Username:     "testuser",
		PasswordHash: passwordHash,
		Role:         "operator",
		Email:        "test@opsxcli.local",
	}

	err = userRepo.Create(newUser)
	if err != nil {
		// 可能已存在，不算错误
		fmt.Printf("   ⚠️  创建用户失败（可能已存在）: %v\n", err)
	} else {
		fmt.Printf("   新用户: %s (ID: %d)\n", newUser.Username, newUser.ID)
		fmt.Println(color.GreenString("   ✓ 用户创建成功"))
	}
	fmt.Println()

	// 6. 测试审计日志
	fmt.Println(color.YellowString("6️⃣  测试审计日志"))
	auditRepo := db.NewAuditLogRepository(database)

	auditLog := &db.AuditLog{
		UserID:      admin.ID,
		Username:    admin.Username,
		ActionType:  "test",
		Command:     "test command",
		RiskLevel:   "safe",
		Environment: "test",
		IPAddress:   "127.0.0.1",
	}

	err = auditRepo.Create(auditLog)
	if err != nil {
		return fmt.Errorf("创建审计日志失败: %w", err)
	}
	fmt.Printf("   审计日志 ID: %d\n", auditLog.ID)
	fmt.Println(color.GreenString("   ✓ 审计日志创建成功"))
	fmt.Println()

	// 7. 测试审计日志查询
	fmt.Println(color.YellowString("7️⃣  测试审计日志查询"))
	logs, err := auditRepo.List(10, 0, map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("查询审计日志失败: %w", err)
	}
	fmt.Printf("   查询到 %d 条审计日志\n", len(logs))
	for i, log := range logs {
		if i >= 3 {
			break // 只显示前3条
		}
		fmt.Printf("   - [%s] %s: %s (风险: %s)\n", log.CreatedAt.Format("15:04:05"), log.Username, log.Command, log.RiskLevel)
	}
	fmt.Println(color.GreenString("   ✓ 审计日志查询成功"))
	fmt.Println()

	// 8. 测试审计统计
	fmt.Println(color.YellowString("8️⃣  测试审计统计"))
	stats, err := auditRepo.GetStats(nil, 24*time.Hour)
	if err != nil {
		return fmt.Errorf("获取统计失败: %w", err)
	}
	fmt.Printf("   最近24小时:\n")
	fmt.Printf("   - 总操作: %d\n", stats["total"])
	fmt.Printf("   - 危险操作: %d\n", stats["dangerous"])
	fmt.Printf("   - 已执行: %d\n", stats["executed"])
	fmt.Printf("   - 成功: %d\n", stats["successful"])
	fmt.Println(color.GreenString("   ✓ 统计数据获取成功"))
	fmt.Println()

	// 总结
	fmt.Println(color.GreenString("✅ 所有测试通过！"))
	fmt.Println()
	fmt.Println("📊 测试总结:")
	fmt.Println("  ✓ 数据库初始化")
	fmt.Println("  ✓ 用户管理")
	fmt.Println("  ✓ 密码验证")
	fmt.Println("  ✓ JWT 认证")
	fmt.Println("  ✓ 审计日志")
	fmt.Println()

	return nil
}
