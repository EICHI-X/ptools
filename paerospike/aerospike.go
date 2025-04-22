package paerospike

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"runtime/debug"

	"github.com/EICHI-X/ptools/env"
	"github.com/EICHI-X/ptools/logs"
	"github.com/EICHI-X/ptools/putils"
	aerospike "github.com/aerospike/aerospike-client-go/v6"
	"github.com/pkg/errors"
)

const DefaultBin = "value"
const MaxBatchSize = 100

type Client struct {
	Psm       string
	Host      string
	Port      int
	Client    *aerospike.Client
	Namespace string
	Policy    *aerospike.ClientPolicy
	Set       string
}

const keyFormatMsg = `to prevent duplicate keys, key must use partten:"appid|project|key"
	 such as your appid=1000,project=packer,key=article.1234
	 key=1000|packer|article.1234`

func CheckKeyFormat(key string) error {
	p := strings.Split(key, "|")
	if len(p) < 2 {
		return errors.New(keyFormatMsg)
	}
	return nil
}

// NewDefaultClient namespace表示数据库,set表示表
func NewDefaultClient(psm string) *Client {
	defer func() {

		if r := recover(); r != nil {
			// 打印错误栈
			stack := string(debug.Stack())
			// 使用fmt.Print 打印到run.log
			fmt.Printf("[NewDefaultClient] err %v", psm)
			fmt.Print(stack)
		}
	}()
	hostsStr := os.Getenv(strings.ToUpper(psm) + ".HOSTS")
	hostsList := strings.Split(hostsStr, ",")
	fmt.Printf("psm %v,ip=%v \n", psm, hostsStr)
	port := 3000
	defaultHosts := []string{"59.63.188.105:3000", "59.63.188.106:3000", "59.63.188.107:3000", "59.63.188.108:3000", "59.63.188.109:3000"}
	if len(hostsList) == 0 {
		hostsList = defaultHosts
	}

	// port := 3000

	hosts, err := aerospike.NewHosts(hostsList...)
	client, err := aerospike.NewClientWithPolicyAndHost(aerospike.NewClientPolicy(), hosts...)
	if client == nil || err != nil {
		panic(fmt.Errorf(fmt.Sprintf("NewDefaultClient psm:%v fail ip=%v,err=%v", psm, hostsList, err)))
	}
	set := strings.ReplaceAll(psm, ".", "__")

	c := &Client{
		Psm:       psm,
		Host:      string(hostsStr),
		Namespace: "wealth",
		Port:      port, // aerospike 统一默认端口3000
		Policy:    aerospike.NewClientPolicy(),
		Client:    client,
		Set:       set,
	}
	fmt.Printf("NewDefaultClient psm %v,hosts=%v success,v=%v \n", psm, hostsList, putils.ToJsonSonic(c.GetClient()))
	return c
}
func NewClientWithPolicy(psm string, namespace string, set string, policy *aerospike.ClientPolicy) *Client {
	ip, err := env.ResolvePsmToIp(psm)
	if err != nil || len(ip) == 0 {
		panic(fmt.Sprintf("parse psm:%v fail ip=%v,err=%v", psm, ip, err))
	}
	if policy == nil {
		policy = aerospike.NewClientPolicy()
		policy.ConnectionQueueSize = 1024
	}
	port := 3000

	client, err := aerospike.NewClientWithPolicy(policy, string(ip), port)
	if client == nil || err != nil {
		panic(fmt.Sprintf("NewClientWithPolicy psm:%v fail ip=%v,err=%v", psm, ip, err))
	}
	c := &Client{
		Psm:       psm,
		Host:      string(ip),
		Port:      port, // aerospike 统一默认端口3000
		Policy:    policy,
		Client:    client,
		Namespace: namespace,
		Set:       set,
	}

	return c
}
func (i *Client) GetClient() *aerospike.Client {
	return i.Client
}

// 格式必须是 key=1000|packer|article.1234
func (i *Client) Put(key string, value string, ttl uint32) error {
	if err := CheckKeyFormat(key); err != nil {
		return err
	}
	client := i.GetClient()
	keySpike, err := aerospike.NewKey(i.Namespace, i.Set, key)
	if err != nil {
		return err
	}
	writePolicy := aerospike.NewWritePolicy(0, ttl)

	r := client.Put(writePolicy, keySpike, aerospike.BinMap{DefaultBin: value})
	// client.Put(aerospike.NewWritePolicy(10, 2), key , obj interface{})
	// client.Get(policy *aerospike.BasePolicy, key *aerospike.Key, binNames ...string)
	return r
}

// 格式必须是 key=1000|packer|article.1234
func (i *Client) PutAsync(key string, value string, ttl uint32) error {
	if err := CheckKeyFormat(key); err != nil {
		return err
	}
	go func() error {
		defer func() {

			if r := recover(); r != nil {
				// 打印错误栈
				stack := string(debug.Stack())
				// 使用fmt.Print 打印到run.log
				fmt.Printf("[PanicHandler] %v", value)
				fmt.Print(stack)
			}
		}()
		// defer putils.TimeCostWithMsg(context.Background(), fmt.Sprintf("aerospike key=%v", key))()
		client := i.GetClient()
		keySpike, err := aerospike.NewKey(i.Namespace, i.Set, key)
		if err != nil {
			return err
		}
		writePolicy := aerospike.NewWritePolicy(0, ttl)

		r := client.PutBins(writePolicy, keySpike, aerospike.NewBin(DefaultBin, value))
		// client.Put(aerospike.NewWritePolicy(10, 2), key , obj interface{})
		// client.Get(policy *aerospike.BasePolicy, key *aerospike.Key, binNames ...string)
		return r
	}()
	return nil
}

// 格式必须是 key=1000|packer|article.1234
func (i *Client) Get(key string) (string, error) {
	if err := CheckKeyFormat(key); err != nil {
		return "", err
	}
	client := i.GetClient()
	keySpike, err := aerospike.NewKey(i.Namespace, i.Set, key)
	if err != nil {
		return "", nil
	}

	r, err := client.Get(aerospike.NewPolicy(), keySpike, DefaultBin)
	// client.Put(aerospike.NewWritePolicy(10, 2), key , obj interface{})
	// client.Get(policy *aerospike.BasePolicy, key *aerospike.Key, binNames ...string)
	if r == nil || err != nil {
		return "", err
	}
	if v, ok := r.Bins[DefaultBin]; ok {
		if vStr, isStr := v.(string); isStr {
			return vStr, nil
		} else {
			return "", fmt.Errorf("value is not str")

		}
	}
	return "", err
}

// 格式必须是 key=1000|packer|article.1234
func (i *Client) Delete(key string) error {

	client := i.GetClient()
	keySpike, err := aerospike.NewKey(i.Namespace, i.Set, key)
	if err != nil {
		return err
	}

	_, err = client.Delete(nil, keySpike)
	// client.Put(aerospike.NewWritePolicy(10, 2), key , obj interface{})
	// client.Get(policy *aerospike.BasePolicy, key *aerospike.Key, binNames ...string)

	return err
}

// 格式必须是 key=1000|packer|article.1234
// 如果batchSize 小于等于0，则赋值batchSize = 100
func (c *Client) GetBatch(keyStrs []string, batchSize int) ([]string, error) {
	// Batch gets into one call.
	keyLen := len(keyStrs)
	if batchSize <= 0 {
		batchSize = MaxBatchSize
	}
	resStr := make([]string, keyLen)
	keysBatch := make([][]*aerospike.Key, keyLen/batchSize+1)
	wg := &sync.WaitGroup{}
	for iBatch := 0; iBatch < len(keysBatch); iBatch++ {
		iBatchIdx := iBatch
		left := putils.MaxInt(0, iBatchIdx*batchSize)
		right := putils.MinInt(keyLen, batchSize*(iBatchIdx+1))
		keysBatch[iBatchIdx] = make([]*aerospike.Key, right-left)
		for iKey := left; iKey < right; iKey++ {
			batchIdx := iKey % batchSize
			keysBatch[iBatchIdx][batchIdx], _ = aerospike.NewKey(c.Namespace, c.Set, keyStrs[iKey])
		}
		wg.Add(1)
		go putils.GoFuncDone(context.Background(), wg, nil, func(ctx context.Context, param interface{}) {
			records, err := c.Client.BatchGet(nil, keysBatch[iBatchIdx], DefaultBin)
			logs.CtxDebugf(ctx, "keysBatch batchIdx=%v,left=%v,right=%v,batch[idx]=%v,batch=%v", iBatchIdx, left, right, putils.ToJson(keysBatch[iBatchIdx]), putils.ToJson(keysBatch))
			if err != nil || len(records) != len(keysBatch[iBatchIdx]) {
				return
			}

			for batchIdx := 0; batchIdx < len(records); batchIdx++ {

				record := records[batchIdx]

				var value interface{}

				if record != nil {

					value = record.Bins[DefaultBin]
				}
				if v, ok := value.(string); ok {
					realIdx := iBatchIdx*batchSize + batchIdx
					if realIdx > keyLen {
						continue
					}
					resStr[realIdx] = v
				}
			}
		})
	}
	wg.Wait()

	return resStr, nil
}

/*

格式必须是 key=1000|packer|article.1234  分别是app_id|project|key
*/

func (c *Client) Operate(key string, ops []*aerospike.Operation, ttl uint32, policy *aerospike.WritePolicy) (r *aerospike.Record, err error) {
	if err := CheckKeyFormat(key); err != nil {
		fmt.Println("Failed to check key format in Aerospike: ", err)
		return nil, err

	}
	keySpike, _ := aerospike.NewKey(c.Namespace, c.Set, key)
	if policy == nil {
		policy = aerospike.NewWritePolicy(0, ttl)
	} else {
		if ttl > 0 {
			policy.Expiration = ttl

		}
	}

	r, err = c.Client.Operate(policy, keySpike, ops...)
	if err != nil {
		fmt.Println("Failed to append data to the list in Aerospike: ", err)
		return r, err
	}
	return r, err
}

/*
格式必须是 key=1000|packer|article.1234  分别是app_id|project|key
*/
func (c *Client) GetBins(key string, bins []string) (r *aerospike.Record, err error) {
	if err := CheckKeyFormat(key); err != nil {
		fmt.Println("Failed to check key format in Aerospike: ", err)
		return nil, err

	}
	keySpike, _ := aerospike.NewKey(c.Namespace, c.Set, key)
	r, err = c.GetClient().Get(nil, keySpike, bins...)
	if err != nil {
		fmt.Println("Failed to append data to the list in Aerospike: ", err)
		return
	}

	return r, err
}
