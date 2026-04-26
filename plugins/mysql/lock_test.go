package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== InnoDBLockWait 解析测试 ====================

func TestParseInnoDBLockWaits(t *testing.T) {
	t.Run("空输入", func(t *testing.T) {
		got := ParseInnoDBLockWaits("")
		assert.Equal(t, 0, len(got))
	})

	t.Run("无锁等待内容", func(t *testing.T) {
		input := `=====================================
150101 12:00:00 INNODB MONITOR OUTPUT
=====================================
Per second averages calculated from the last 10 seconds
----------
SEMAPHORES
----------
OS WAIT ARRAY INFO: reservation count 10, signal count 10
----------
BUFFER POOL AND MEMORY
----------
Total memory allocated 137363456; in additional pool allocated 0
Dictionary memory allocated 81920
Buffer pool size   8191
Free buffers       7770
Database pages     420
Old database pages 0
END OF INNODB MONITOR OUTPUT`
		got := ParseInnoDBLockWaits(input)
		assert.Equal(t, 0, len(got))
	})

	t.Run("单条锁等待", func(t *testing.T) {
		input := `---TRANSACTION 12345, ACTIVE 30 sec starting index read
mysql tables in use 1, locked 1
LOCK WAIT 2 lock struct(s), heap size 360, 1 row lock(s)
MySQL thread id 10, OS thread handle 0x1234, query id 100 localhost root updating
UPDATE users SET name='test' WHERE id=1
------- TRX HAS BEEN WAITING 5 SEC FOR THIS LOCK TO BE GRANTED:
RECORD LOCKS space id 58 page no 4 n bits 72 index ` + "`PRIMARY`" + ` of table ` + "`test`.`users`" + ` trx id 12345 lock_mode X locks rec but not gap waiting
Record lock, heap no 2 PHYSICAL RECORD: n_fields 3; compact format; info bits 0
 0: len 4; hex 80000001; asc     ;;
 1: len 6; hex 000000000918; asc       ;;
 2: len 7; hex 81000001100110; asc        ;;

---TRANSACTION 12340, ACTIVE 40 sec
2 lock struct(s), heap size 360, 1 row lock(s), undo log entries 1
MySQL thread id 9, OS thread handle 0x5678, query id 99 localhost root updating
UPDATE users SET email='new@test.com' WHERE id=1`
		got := ParseInnoDBLockWaits(input)
		assert.Equal(t, 1, len(got))

		lw := got[0]
		assert.Equal(t, int64(12345), lw.TransactionID)
		assert.Equal(t, int64(30), lw.ActiveSec)
		assert.Equal(t, int64(10), lw.ThreadID)
		assert.Equal(t, "localhost", lw.Host)
		assert.Equal(t, "root", lw.User)
		assert.Equal(t, int64(5), lw.WaitSec)
		assert.Contains(t, lw.WaitingQuery, "UPDATE users SET name='test'")
		assert.Contains(t, lw.LockMode, "X")
		assert.Contains(t, lw.LockTable, "test")
	})

	t.Run("多条锁等待", func(t *testing.T) {
		input := `---TRANSACTION 100, ACTIVE 10 sec starting index read
mysql tables in use 1, locked 1
LOCK WAIT 2 lock struct(s), heap size 360
MySQL thread id 5, OS thread handle 0xaaa, query id 50 10.0.0.1 app_user updating
DELETE FROM orders WHERE status='expired'
------- TRX HAS BEEN WAITING 3 SEC FOR THIS LOCK TO BE GRANTED:
RECORD LOCKS space id 10 page no 5 n bits 72 index idx_status of table ` + "`shop`.`orders`" + ` trx id 100 lock_mode X waiting

---TRANSACTION 200, ACTIVE 20 sec fetching rows
mysql tables in use 1, locked 1
LOCK WAIT 3 lock struct(s), heap size 360
MySQL thread id 8, OS thread handle 0xbbb, query id 80 192.168.1.5 app_user Searching
SELECT * FROM products WHERE price > 100 FOR UPDATE
------- TRX HAS BEEN WAITING 15 SEC FOR THIS LOCK TO BE GRANTED:
RECORD LOCKS space id 20 page no 10 n bits 72 index idx_price of table ` + "`shop`.`products`" + ` trx id 200 lock_mode X locks rec but not gap waiting`
		got := ParseInnoDBLockWaits(input)
		assert.Equal(t, 2, len(got))

		assert.Equal(t, int64(100), got[0].TransactionID)
		assert.Equal(t, int64(5), got[0].ThreadID)
		assert.Equal(t, "10.0.0.1", got[0].Host)
		assert.Equal(t, int64(3), got[0].WaitSec)
		assert.Contains(t, got[0].WaitingQuery, "DELETE FROM orders")

		assert.Equal(t, int64(200), got[1].TransactionID)
		assert.Equal(t, int64(8), got[1].ThreadID)
		assert.Equal(t, int64(15), got[1].WaitSec)
		assert.Contains(t, got[1].WaitingQuery, "SELECT * FROM products")
	})

	t.Run("死锁检测", func(t *testing.T) {
		input := `------------------------
LATEST DETECTED DEADLOCK
------------------------
150101 12:00:00
*** (1) TRANSACTION:
TRANSACTION 1000, ACTIVE 5 sec starting index read
mysql tables in use 1, locked 1
LOCK WAIT 2 lock struct(s), heap size 360, 1 row lock(s)
MySQL thread id 10, OS thread handle 0x1111, query id 200 localhost root updating
UPDATE accounts SET balance=100 WHERE id=1
*** (1) WAITING FOR THIS LOCK TO BE GRANTED:
RECORD LOCKS space id 58 page no 4 n bits 72 index ` + "`PRIMARY`" + ` of table ` + "`bank`.`accounts`" + ` trx id 1000 lock_mode X locks rec but not gap waiting

*** (2) TRANSACTION:
TRANSACTION 2000, ACTIVE 3 sec starting index read
mysql tables in use 1, locked 1
3 lock struct(s), heap size 360, 2 row lock(s), undo log entries 1
MySQL thread id 11, OS thread handle 0x2222, query id 201 localhost root updating
UPDATE accounts SET balance=200 WHERE id=2
*** (2) HOLDS THE LOCK(S):
RECORD LOCKS space id 58 page no 4 n bits 72 index ` + "`PRIMARY`" + ` of table ` + "`bank`.`accounts`" + ` trx id 2000 lock_mode X locks rec but not gap
*** (2) WAITING FOR THIS LOCK TO BE GRANTED:
RECORD LOCKS space id 58 page no 5 n bits 72 index ` + "`PRIMARY`" + ` of table ` + "`bank`.`accounts`" + ` trx id 2000 lock_mode X locks rec but not gap waiting
*** WE ROLL BACK TRANSACTION (2)`
		got := ParseInnoDBLockWaits(input)
		assert.Equal(t, 2, len(got))

		assert.Equal(t, int64(1000), got[0].TransactionID)
		assert.Contains(t, got[0].WaitingQuery, "UPDATE accounts SET balance=100")
		assert.Contains(t, got[0].LockMode, "X")

		assert.Equal(t, int64(2000), got[1].TransactionID)
		assert.Contains(t, got[1].WaitingQuery, "UPDATE accounts SET balance=200")
	})
}

// ==================== DetectDeadlocks 测试 ====================

func TestDetectDeadlocks(t *testing.T) {
	t.Run("无死锁", func(t *testing.T) {
		input := "No deadlock content here"
		got := DetectDeadlocks(input)
		assert.False(t, got.Detected)
		assert.Equal(t, 0, len(got.Victims))
	})

	t.Run("有死锁", func(t *testing.T) {
		input := `------------------------
LATEST DETECTED DEADLOCK
------------------------
150101 12:00:00
*** (1) TRANSACTION:
TRANSACTION 1000, ACTIVE 5 sec starting index read
MySQL thread id 10, OS thread handle 0x1111, query id 200 localhost root updating
UPDATE accounts SET balance=100 WHERE id=1
*** (1) WAITING FOR THIS LOCK TO BE GRANTED:
RECORD LOCKS space id 58 page no 4 of table ` + "`bank`.`accounts`" + ` trx id 1000 lock_mode X waiting
*** (2) TRANSACTION:
TRANSACTION 2000, ACTIVE 3 sec starting index read
MySQL thread id 11, OS thread handle 0x2222, query id 201 localhost root updating
UPDATE accounts SET balance=200 WHERE id=2
*** (2) HOLDS THE LOCK(S):
RECORD LOCKS space id 58 page no 4 of table ` + "`bank`.`accounts`" + ` trx id 2000 lock_mode X
*** WE ROLL BACK TRANSACTION (2)`
		got := DetectDeadlocks(input)
		assert.True(t, got.Detected)
		assert.Equal(t, 2, len(got.Participants))
		assert.Contains(t, got.Victims, int64(2000))
	})

	t.Run("死锁标题但不完整", func(t *testing.T) {
		// 只有标题没有完整内容
		input := "LATEST DETECTED DEADLOCK\nsome noise"
		got := DetectDeadlocks(input)
		// 标题存在但无法解析出完整事务
		assert.True(t, got.Detected)
		assert.Equal(t, 0, len(got.Participants))
	})
}

// ==================== FormatLockWaitReport 测试 ====================

func TestFormatLockWaitReport(t *testing.T) {
	t.Run("空列表", func(t *testing.T) {
		got := FormatLockWaitReport(nil, DeadlockInfo{})
		assert.Equal(t, "未检测到锁等待", got)
	})

	t.Run("有锁等待无死锁", func(t *testing.T) {
		waits := []InnoDBLockWait{
			{
				TransactionID: 100,
				ActiveSec:     30,
				ThreadID:      5,
				Host:          "10.0.0.1",
				User:          "app_user",
				WaitSec:       10,
				WaitingQuery:  "UPDATE orders SET status='done' WHERE id=1",
				LockMode:      "X",
				LockTable:     "shop.orders",
				LockIndex:     "PRIMARY",
			},
		}
		got := FormatLockWaitReport(waits, DeadlockInfo{})
		assert.Contains(t, got, "锁等待分析报告")
		assert.Contains(t, got, "1 条")
		assert.Contains(t, got, "30s")
		assert.Contains(t, got, "10s")
		assert.Contains(t, got, "app_user")
		assert.Contains(t, got, "UPDATE orders")
		assert.Contains(t, got, "X")
	})

	t.Run("有锁等待且有死锁", func(t *testing.T) {
		waits := []InnoDBLockWait{
			{TransactionID: 100, ThreadID: 5, WaitSec: 3, WaitingQuery: "DELETE FROM t"},
		}
		dl := DeadlockInfo{
			Detected: true,
			Victims:  []int64{200},
			Participants: []DeadlockParticipant{
				{TransactionID: 100, ThreadID: 5, Query: "UPDATE t SET x=1"},
				{TransactionID: 200, ThreadID: 6, Query: "UPDATE t SET y=2"},
			},
		}
		got := FormatLockWaitReport(waits, dl)
		assert.Contains(t, got, "锁等待分析报告")
		assert.Contains(t, got, "⚠️ 检测到死锁")
		assert.Contains(t, got, "被回滚事务")
		assert.Contains(t, got, "200")
	})
}

// ==================== CategorizeLockMode 测试 ====================

func TestCategorizeLockMode(t *testing.T) {
	tests := []struct {
		name    string
		lockStr string
		want    string
	}{
		{"排他锁", "lock_mode X locks rec but not gap", "排他锁(X)"},
		{"共享锁", "lock_mode S locks rec but not gap", "共享锁(S)"},
		{"意向排他锁", "lock_mode IX", "意向排他锁(IX)"},
		{"意向共享锁", "lock_mode IS", "意向共享锁(IS)"},
		{"排他间隙锁", "lock_mode X locks gap before rec", "排他间隙锁(X)"},
		{"自动增量锁", "lock_mode AUTO_INC", "自动增量锁"},
		{"未知类型", "lock_mode Z", "其他(Z)"},
		{"空字符串", "", "未知"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CategorizeLockMode(tt.lockStr)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== AnalyzeLockSeverity 测试 ====================

func TestAnalyzeLockSeverity(t *testing.T) {
	tests := []struct {
		name     string
		waits    []InnoDBLockWait
		deadlock DeadlockInfo
		want     string
	}{
		{
			name:     "无锁等待",
			waits:    nil,
			deadlock: DeadlockInfo{},
			want:     "正常",
		},
		{
			name: "低风险-短等待",
			waits: []InnoDBLockWait{
				{WaitSec: 2, ActiveSec: 5},
			},
			deadlock: DeadlockInfo{},
			want:     "低风险",
		},
		{
			name: "中风险-中等等待",
			waits: []InnoDBLockWait{
				{WaitSec: 10, ActiveSec: 30},
			},
			deadlock: DeadlockInfo{},
			want:     "中风险",
		},
		{
			name: "高风险-长等待",
			waits: []InnoDBLockWait{
				{WaitSec: 60, ActiveSec: 120},
			},
			deadlock: DeadlockInfo{},
			want:     "高风险",
		},
		{
			name:  "有死锁-始终严重",
			waits: nil,
			deadlock: DeadlockInfo{
				Detected: true,
				Victims:  []int64{100},
			},
			want: "严重",
		},
		{
			name: "多条等待-取最大等待",
			waits: []InnoDBLockWait{
				{WaitSec: 2, ActiveSec: 5},
				{WaitSec: 4, ActiveSec: 10},
				{WaitSec: 3, ActiveSec: 8},
			},
			deadlock: DeadlockInfo{},
			want:     "低风险",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AnalyzeLockSeverity(tt.waits, tt.deadlock)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== GenerateLockWaitSuggestions 测试 ====================

func TestGenerateLockWaitSuggestions(t *testing.T) {
	t.Run("无锁等待-无建议", func(t *testing.T) {
		got := GenerateLockWaitSuggestions(nil, DeadlockInfo{})
		assert.Equal(t, 0, len(got))
	})

	t.Run("单条短等待-无建议", func(t *testing.T) {
		waits := []InnoDBLockWait{
			{WaitSec: 2, ActiveSec: 5, LockMode: "X", WaitingQuery: "UPDATE t SET x=1"},
		}
		got := GenerateLockWaitSuggestions(waits, DeadlockInfo{})
		assert.Equal(t, 0, len(got))
	})

	t.Run("长等待-有建议", func(t *testing.T) {
		waits := []InnoDBLockWait{
			{WaitSec: 30, ActiveSec: 60, LockMode: "X", WaitingQuery: "UPDATE big_table SET col=val"},
		}
		got := GenerateLockWaitSuggestions(waits, DeadlockInfo{})
		assert.True(t, len(got) > 0)
		// 应该包含优化建议
		found := false
		for _, s := range got {
			if s.Level == "warning" || s.Level == "critical" {
				found = true
				break
			}
		}
		assert.True(t, found, "应该包含 warning 或 critical 级别建议")
	})

	t.Run("死锁-有建议", func(t *testing.T) {
		dl := DeadlockInfo{
			Detected: true,
			Victims:  []int64{100},
			Participants: []DeadlockParticipant{
				{TransactionID: 100, Query: "UPDATE t1"},
				{TransactionID: 200, Query: "UPDATE t2"},
			},
		}
		got := GenerateLockWaitSuggestions(nil, dl)
		assert.True(t, len(got) > 0)
		assert.Contains(t, got[0].Level, "critical")
	})
}
