package redis

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/go-redis/redis/v8"

	"opsxcli/internal/db"
	"opsxcli/internal/logger"
)

var ctx = context.Background()

// GetClient 获取Redis客户端
func GetClient(host string, port int, password string, dbNum int, cluster bool, addrs string) (redis.UniversalClient, error) {
	// 应用默认配置
	host, port, _ = db.ApplyDefaults(host, port, "", "redis")

	if cluster {
		// 集群模式
		if addrs != "" {
			// 使用指定的地址列表
			addrList := strings.Split(addrs, ",")
			for i, addr := range addrList {
				addrList[i] = strings.TrimSpace(addr)
			}
			return redis.NewClusterClient(&redis.ClusterOptions{
				Addrs:        addrList,
				Password:     password,
				DialTimeout:  5 * time.Second,
				ReadTimeout:  3 * time.Second,
				WriteTimeout: 3 * time.Second,
			}), nil
		}
		// 使用单个地址
		return redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        []string{fmt.Sprintf("%s:%d", host, port)},
			Password:     password,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		}), nil
	}

	// 单机模式
	return redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           dbNum,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}), nil
}

// Interactive 交互式Redis shell
func Interactive(host string, port int, password string, db int, cluster bool, addrs string) error {
	client, err := GetClient(host, port, password, db, cluster, addrs)
	if err != nil {
		logger.Error("创建Redis客户端失败: %v", err)
		return fmt.Errorf("创建Redis客户端失败: %v", err)
	}
	defer client.Close()

	// 先验证连接（设置超时）
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = client.Ping(pingCtx).Err()
	if err != nil {
		// 如果是 NOAUTH 错误，允许进入交互式模式，用户可以手动执行 AUTH 命令
		if strings.Contains(err.Error(), "NOAUTH") {
			color.Yellow("⚠ Redis 需要密码认证")
			fmt.Println("提示: 请使用 AUTH 命令进行认证，例如: AUTH <password>")
		} else {
			// 其他错误则直接返回
			return fmt.Errorf("连接Redis失败: %v (请检查网络连接和服务器状态)", err)
		}
	} else {
		// 连接成功
		logger.Success("已连接到 Redis (%s:%d)", host, port)
	}

	if cluster {
		fmt.Println("集群模式")
	}

	// 创建readline实例（带命令补全）
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          fmt.Sprintf("%s:%d> ", host, port),
		HistoryFile:     os.Getenv("HOME") + "/.opsxcli/redis_history",
		AutoComplete:    getRedisCompleter(),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return err
	}
	defer rl.Close()

	// 交互式循环
	for {
		line, err := rl.Readline()
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 处理特殊命令
		if line == "exit" || line == "quit" || line == "\\q" {
			break
		}
		if line == "help" || line == "\\h" {
			printHelp()
			continue
		}

		// 解析命令（支持引号）
		parts := parseRedisCommand(line)
		if len(parts) == 0 {
			continue
		}

		// 使用 Do 方法执行任意 Redis 命令
		// Do 方法的签名: Do(ctx, ...interface{})
		cmdArgs := make([]interface{}, len(parts))
		for i, part := range parts {
			cmdArgs[i] = part
		}

		// 执行命令（使用 Do 方法支持所有 Redis 命令，设置超时）
		cmdCtx, cmdCancel := context.WithTimeout(ctx, 10*time.Second)
		result := client.Do(cmdCtx, cmdArgs...)
		cmdCancel()

		if err := result.Err(); err != nil {
			if err == redis.Nil {
				fmt.Println("(nil)")
			} else if err == context.DeadlineExceeded {
				logger.Error("(error) 命令执行超时")
			} else {
				logger.Error("(error) %v", err)
			}
			continue
		}

		// 格式化输出结果
		printResult(result.Val())
	}

	fmt.Println("Bye")
	return nil
}

// Get 获取键值
func Get(host string, port int, password string, db int, cluster bool, addrs string, key string) error {
	client, err := GetClient(host, port, password, db, cluster, addrs)
	if err != nil {
		return fmt.Errorf("创建Redis客户端失败: %v", err)
	}
	defer client.Close()

	// 验证连接
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		if strings.Contains(err.Error(), "NOAUTH") {
			return fmt.Errorf("连接Redis失败: %v\n提示: 该Redis实例需要密码认证，请使用 -a 参数提供密码", err)
		}
		return fmt.Errorf("连接Redis失败: %v", err)
	}

	cmdCtx, cmdCancel := context.WithTimeout(ctx, 10*time.Second)
	defer cmdCancel()
	result := client.Get(cmdCtx, key)
	if err := result.Err(); err != nil {
		return err
	}

	fmt.Println(result.Val())
	return nil
}

// Set 设置键值
func Set(host string, port int, password string, db int, cluster bool, addrs string, key, value string, expire time.Duration) error {
	client, err := GetClient(host, port, password, db, cluster, addrs)
	if err != nil {
		return fmt.Errorf("创建Redis客户端失败: %v", err)
	}
	defer client.Close()

	// 验证连接
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		if strings.Contains(err.Error(), "NOAUTH") {
			return fmt.Errorf("连接Redis失败: %v\n提示: 该Redis实例需要密码认证，请使用 -a 参数提供密码", err)
		}
		return fmt.Errorf("连接Redis失败: %v", err)
	}

	cmdCtx, cmdCancel := context.WithTimeout(ctx, 10*time.Second)
	defer cmdCancel()

	var setErr error
	if expire > 0 {
		setErr = client.Set(cmdCtx, key, value, expire).Err()
	} else {
		setErr = client.Set(cmdCtx, key, value, 0).Err()
	}

	if setErr != nil {
		return setErr
	}

	fmt.Println("OK")
	return nil
}

// parseRedisCommand 解析Redis命令（支持引号）
func parseRedisCommand(line string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false
	quoteChar := byte(0)

	for i := 0; i < len(line); i++ {
		char := line[i]

		if char == '"' || char == '\'' {
			if !inQuotes {
				inQuotes = true
				quoteChar = char
			} else if char == quoteChar {
				inQuotes = false
				quoteChar = 0
			} else {
				current.WriteByte(char)
			}
		} else if char == ' ' && !inQuotes {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(char)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// getRedisCompleter 创建Redis命令补全器
func getRedisCompleter() *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("GET"),
		readline.PcItem("SET"),
		readline.PcItem("DEL"),
		readline.PcItem("KEYS"),
		readline.PcItem("EXISTS"),
		readline.PcItem("EXPIRE"),
		readline.PcItem("TTL"),
		readline.PcItem("TYPE"),
		readline.PcItem("INCR"),
		readline.PcItem("DECR"),
		readline.PcItem("INCRBY"),
		readline.PcItem("DECRBY"),
		readline.PcItem("APPEND"),
		readline.PcItem("STRLEN"),
		readline.PcItem("HSET"),
		readline.PcItem("HGET"),
		readline.PcItem("HGETALL"),
		readline.PcItem("HDEL"),
		readline.PcItem("HKEYS"),
		readline.PcItem("HVALS"),
		readline.PcItem("LPUSH"),
		readline.PcItem("RPUSH"),
		readline.PcItem("LPOP"),
		readline.PcItem("RPOP"),
		readline.PcItem("LLEN"),
		readline.PcItem("LRANGE"),
		readline.PcItem("SADD"),
		readline.PcItem("SMEMBERS"),
		readline.PcItem("SREM"),
		readline.PcItem("SCARD"),
		readline.PcItem("ZADD"),
		readline.PcItem("ZRANGE"),
		readline.PcItem("ZREM"),
		readline.PcItem("ZCARD"),
		readline.PcItem("CLUSTER"),
		readline.PcItem("CLUSTER", readline.PcItem("NODES")),
		readline.PcItem("CLUSTER", readline.PcItem("INFO")),
		readline.PcItem("CLUSTER", readline.PcItem("SLOTS")),
		readline.PcItem("INFO"),
		readline.PcItem("PING"),
		readline.PcItem("DBSIZE"),
		readline.PcItem("FLUSHDB"),
		readline.PcItem("FLUSHALL"),
		readline.PcItem("SELECT"),
		readline.PcItem("AUTH"),
		readline.PcItem("QUIT"),
		readline.PcItem("EXIT"),
	)
}

// 辅助函数
func printResult(val interface{}) {
	switch v := val.(type) {
	case string:
		fmt.Printf("\"%s\"\n", v)
	case []interface{}:
		if len(v) == 0 {
			fmt.Println("(empty list or set)")
		} else {
			for i, item := range v {
				fmt.Printf("%d) ", i+1)
				printResult(item)
			}
		}
	case []byte:
		fmt.Printf("\"%s\"\n", string(v))
	case int64:
		fmt.Printf("(integer) %d\n", v)
	case nil:
		fmt.Println("(nil)")
	case bool:
		if v {
			fmt.Println("(integer) 1")
		} else {
			fmt.Println("(integer) 0")
		}
	default:
		// 尝试转换为字符串
		if str, ok := v.(string); ok {
			fmt.Printf("\"%s\"\n", str)
		} else {
			fmt.Printf("%v\n", v)
		}
	}
}

func printHelp() {
	fmt.Println("Redis命令帮助:")
	fmt.Println("  输入任意Redis命令，如: GET key, SET key value, CLUSTER NODES")
	fmt.Println("  支持命令补全（按 TAB 键）")
	fmt.Println("  help 或 \\h  - 显示帮助")
	fmt.Println("  exit 或 quit - 退出")
}
