package common

import (
	"bytes"
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func SubscribeLogs(rpcUrl string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// 解析 abi 文件内容（不能直接把文件路径字符串传给 abi.JSON）
	abiData, err := os.ReadFile("contracts/MyToken.abi.json")
	if err != nil {
		// 兼容从不同工作目录启动程序的场景
		abiData, err = os.ReadFile("../contracts/MyToken.abi.json")
		if err != nil {
			log.Fatal(err)
		}
	}
	parsedABI, err := abi.JSON(bytes.NewReader(abiData))
	if err != nil {
		log.Fatal(err)
	}

	contract := common.HexToAddress("0x610178dA211FEF7D417bC0e6FeD39F05609AD788")

	query := ethereum.FilterQuery{
		Addresses: []common.Address{contract},
	}

	logsCh := make(chan types.Log)

	sub, err := client.SubscribeFilterLogs(ctx, query, logsCh)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Subscribed to logs of contract %s via %s\n", contract.Hex(), rpcUrl)

	cancelCh := make(chan os.Signal, 1)
	signal.Notify(cancelCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
			return
		case vLog := <-logsCh:
			parseEvent(&vLog, parsedABI)
		case sig := <-cancelCh:
			log.Printf("Received signal \"%v\", shutting down...\n", sig)
			return
		case <-ctx.Done():
			log.Println("Terminating due to context cancellation")
			return
		}
	}
}

// 解析事件及参数
func parseEvent(vLog *types.Log, parsedABI abi.ABI) {
	if len(vLog.Topics) == 0 {
		return
	}

	// 获取事件类型
	eventTopic := vLog.Topics[0]

	var eventName string
	var eventSig abi.Event

	for name, event := range parsedABI.Events {
		eventSigHash := crypto.Keccak256Hash([]byte(event.Sig))
		if eventSigHash == eventTopic {
			eventName = name
			eventSig = event
			break
		}
	}

	if eventName == "" {
		// 无法识别事件类型
		log.Printf("[%s] Unknown Event - Block: %d, Tx %s, Topic[0]: %s\n",
			time.Now().Format("2006-01-02 15:04:05"),
			vLog.BlockNumber,
			vLog.TxHash.Hex(),
			eventTopic.Hex())
		return
	}

	// 解析事件参数
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	log.Printf("[%s] Event: %s\n", time.Now().Format(time.RFC3339), eventName)
	log.Printf("  Block Number: %d\n", vLog.BlockNumber)
	log.Printf("  Tx Hash     : %s\n", vLog.TxHash.Hex())
	log.Printf("  Log Index   : %d\n", vLog.Index)
	log.Printf("  Contract    : %s\n", vLog.Address.Hex())
	log.Printf("  Topics Count: %d\n", len(vLog.Topics))

	// indexed参数从Topics中解析
	// 只有前3个indexed参数会放在Topics中
	log.Println("Indexed Parameters (from Topics):")

	indexedParamIndex := 0
	for i, input := range eventSig.Inputs {
		if !input.Indexed {
			continue
		}

		// indexed参数从第二个Topic开始
		topicIndex := 1 + indexedParamIndex
		indexedParamIndex++

		if topicIndex >= len(vLog.Topics) {
			continue
		}

		topic := vLog.Topics[topicIndex]
		log.Printf("[%d] %s (%s): ", i+1, input.Name, input.Type)

		switch input.Type.T {
		case abi.AddressTy:
			// address类型，去除前12字节的0填充，后20字节是地址
			addr := common.BytesToAddress(topic.Bytes())
			log.Printf("%s\n", addr.Hex())
		case abi.IntTy, abi.UintTy:
			// 整型
			value := new(big.Int).SetBytes(topic.Bytes())
			log.Printf("%s\n", value.String())
		case abi.BoolTy:
			// 布尔
			log.Printf("%t\n", topic[31] != 0)
		case abi.BytesTy:
			// bytes类型
			log.Printf("%s\n", topic.Hex())
		default:
			log.Printf("%s(raw)\n", topic.Hex())
		}
	}

	// 解析非indexed参数，从Data中解析
	if len(vLog.Data) > 0 {
		log.Printf("Non-indexed Parameters (from Data):\n")

		// 实际应用中需要根据具体事件定义结构体
		nonIndexedInputs := make([]abi.Argument, 0)
		for _, input := range eventSig.Inputs {
			if !input.Indexed {
				nonIndexedInputs = append(nonIndexedInputs, input)
			}
		}

		if len(nonIndexedInputs) > 0 {
			// 使用ABI解码Data字段
			// 方法1：使用UnpackIntoInterface(需预定义结构体)
			// 方法2：使用Unpack(返回[]interface{})
			values, err := parsedABI.Unpack(eventName, vLog.Data)
			if err != nil {
				log.Fatal(err)
			} else {
				nonIndexedIdx := 0
				for i, input := range eventSig.Inputs {
					if !input.Indexed {
						if nonIndexedIdx < len(values) {
							value := values[nonIndexedIdx]
							log.Printf("[%d] %s (%s): ", i+1, input.Name, input.Type)

							switch v := value.(type) {
							case *big.Int:
								log.Printf("%s\n", v.String())
							case common.Address:
								log.Printf("%s\n", v.Hex())
							case []byte:
								log.Printf("0x%x\n", v)
							default:
								log.Printf("%v\n", v)
							}
							nonIndexedIdx++
						}
					}
				}
			}
		}
	} else {
		log.Printf("\n Non-indexed Parameters\n")
	}

	log.Printf("-----------------------------------------\n\n")
}
