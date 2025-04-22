package genid

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// lockKey 格式 "project|feature|key"
func Lock(ctx context.Context, rdb *redis.Client, appId string, project string, lockKey string) (int64, string, error) {
   s := strings.Split(lockKey, "|")
   if len(s) < 3 {
         return -1, "", fmt.Errorf("lockKey format error,use format project|feature|key")
   }

	lockValue := uuid.New().String() // 可以使用 UUID 或其他唯一值
	expireTime := 10 * time.Second

	// 获取锁
	lockScript := `
        local lock_key = KEYS[1]
        local lock_value = ARGV[1]
        local expire_time = ARGV[2]

        if redis.call("SETNX", lock_key, lock_value) == 1 then
            redis.call("EXPIRE", lock_key, expire_time)
            return 1
        else
            return 0
        end
    `

	result, err := rdb.Eval(ctx, lockScript, []string{lockKey}, lockValue, int(expireTime.Seconds())).Result()
	if err != nil || result == nil {
		fmt.Println("Error:", err)
		return -1, "", err
	}

	v, ok := result.(int64)
	if ok && v == 1 {
		fmt.Println("Lock acquired")
		// 执行需要加锁的操作

	} else {
		err = fmt.Errorf("Failed to acquire lock")
		return -1, "", err
	}
	return v, lockValue, nil
}
// lockKey 格式 "project|feature|key"
func Unlock(ctx context.Context,rdb *redis.Client, appId string, project string, lockKey string, lockValue string) error {
	// 释放锁
       s := strings.Split(lockKey, "|")
   if len(s) < 3 {
         return  fmt.Errorf("lockKey format error,use format project|feature|key")
   }
	unlockScript := `
            local lock_key = KEYS[1]
            local lock_value = ARGV[1]

            if redis.call("GET", lock_key) == lock_value then
                return redis.call("DEL", lock_key)
            else
                return 0
            end
        `
	_, err := rdb.Eval(ctx, unlockScript, []string{lockKey}, lockValue).Result()
	if err != nil {
		err = fmt.Errorf("Error releasing lock: %v", err)
		return err
	} else {
		fmt.Println("Lock released")
	}
	return nil
}
