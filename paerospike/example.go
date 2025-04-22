package paerospike

import (
	"fmt"

	aerospike "github.com/aerospike/aerospike-client-go/v6"
)

func getClientTest() *Client {
	client := NewDefaultClient("wealth.stock.common")
	return client
}
func UpdateAerospikeList(psm string, keyStr string, list []string) (err error) {
	// 获取列表
	client := getClientTest()

	// 创建Aerospike客户端连接

	defer client.GetClient().Close()

	// 要添加到列表的新数据
	newData := 6
	// 直接在Aerospike中添加新数据到列表
	ops := []*aerospike.Operation{
		aerospike.ListAppendOp("list_bin", newData),
	}
	_, err = client.Operate("1000.test.list_key_test", ops, 30, nil)
	if err != nil {
		fmt.Println("Failed to append data to the list in Aerospike: ", err)
		return
	}
	return
}
func GetAerospikeList(psm string, keyStr string) (err error) {
	client := getClientTest()

	// 创建Aerospike客户端连接

	defer client.GetClient().Close()
	// 直接在Aerospike中添加新数据到列表
	r, err := client.GetBins("1000.test.list_key_test", []string{"list_bin"})
	fmt.Printf("key=%s, value=%v\n", keyStr, r)
	return
}
