package common

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type Block struct {
	Number      string         `json:"number"`
	Hash        common.Hash    `json:"hash"`
	ParentHash  common.Hash    `json:"parentHash"`
	UncleHash   common.Hash    `json:"sha3Uncles"`
	Coinbase    common.Address `json:"miner"`
	Root        common.Hash    `json:"stateRoot"`
	TxHash      common.Hash    `json:"transactionsRoot"`
	ReceiptHash common.Hash    `json:"receiptsRoot"`
	Bloom       hexutil.Bytes  `json:"logsBloom"`
	Difficulty  *hexutil.Big   `json:"difficulty"`
	GasLimit    hexutil.Uint64 `json:"gasLimit"`
	GasUsed     hexutil.Uint64 `json:"gasUsed"`
	Time        hexutil.Uint64 `json:"timestamp"`
	Extra       hexutil.Bytes  `json:"extraData"`
	MixDigest   common.Hash    `json:"mixHash"`
	Nonce       hexutil.Bytes  `json:"nonce"`
	BaseFee     *hexutil.Big   `json:"baseFeePerGas"`
}

func (b Block) String() string {
	result := ""
	result += fmt.Sprintln("------------------------- 区块信息 ----------------------------")
	num, err := strconv.ParseInt(b.Number, 0, 64)
	if err != nil {
		fmt.Println(err)
	}
	result += fmt.Sprintf("Number:%d\n", num)
	result += fmt.Sprintf("Hash:%s\n", b.Hash.Hex())
	result += fmt.Sprintf("ParentHash:%s\n", b.ParentHash.Hex())
	result += fmt.Sprintf("UncleHash:%s\n", b.UncleHash.Hex())
	result += fmt.Sprintf("Coinbase:%s\n", b.Coinbase.Hex())
	result += fmt.Sprintf("Root:%s\n", b.Root.Hex())
	result += fmt.Sprintf("TxHash:%s\n", b.TxHash.Hex())
	result += fmt.Sprintf("ReceiptHash:%s\n", b.ReceiptHash.Hex())
	if b.Difficulty != nil {
		result += fmt.Sprintf("Difficulty:%s\n", b.Difficulty.String())
	}
	result += fmt.Sprintf("Nonce:%d\n", b.Nonce)
	result += fmt.Sprintf("BaseFee:%s\n", b.BaseFee.String())
	return result
}

// GetBlockByTag 获取safe和finalized区块
func GetBlockByTag(ctx context.Context, client *ethclient.Client, tag string) (*types.Header, common.Hash, error) {
	// 获取底层RPC客户端
	rpcClient := client.Client()

	var raw json.RawMessage
	// false只获取header，不包含交易信息
	err := rpcClient.CallContext(ctx, &raw, "eth_getBlockByNumber", tag, false)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("eth_getBlockByNumber: %w", err)
	}

	if len(raw) == 0 || string(raw) == "null" {
		return nil, common.Hash{}, fmt.Errorf("%s block not found", tag)
	}

	var block Block
	err = json.Unmarshal(raw, &block)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("json.Unmarshal: %w", err)
	}

	// 将区号16进制转为10进制
	num, ok := new(big.Int).SetString(block.Number[2:], 16)
	if !ok {
		return nil, common.Hash{}, fmt.Errorf("invalid block number: %s", block.Number)
	}

	header := &types.Header{
		ParentHash:  block.ParentHash,
		UncleHash:   block.UncleHash,
		Coinbase:    block.Coinbase,
		Root:        block.Root,
		TxHash:      block.TxHash,
		ReceiptHash: block.ReceiptHash,
		Bloom:       types.BytesToBloom(block.Bloom),
		Difficulty:  big.NewInt(0),
		Number:      num,
		GasLimit:    uint64(block.GasLimit),
		GasUsed:     uint64(block.GasUsed),
		Time:        uint64(block.Time),
		Extra:       block.Extra,
		MixDigest:   block.MixDigest,
		BaseFee:     nil,
	}

	// 设置difficulty
	if block.Difficulty != nil {
		header.Difficulty = block.Difficulty.ToInt()
	}

	// 设置BaseFee(EIP-1559)
	if block.BaseFee != nil {
		header.BaseFee = block.BaseFee.ToInt()
	}

	// Nonce
	if len(block.Nonce) >= 8 {
		var nonceBytes [8]byte
		copy(nonceBytes[:], block.Nonce[:8])
		header.Nonce = nonceBytes
	}

	// 注意：手动构造的Header计算出的hash可能不准确，因为：
	// 1. RPC返回的某些字段可能格式不完全匹配go-ethereum的内部格式
	// 2. Header的内部缓存字段可能未正确初始化
	// 因此，我们应该直接使用RPC返回的hash，它与浏览器显示的hash一致
	return header, block.Hash, nil
}

func GetBlockWithRetry(ctx context.Context, client *ethclient.Client, blockNumber *big.Int, maxRetries int) (*types.Block, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		reqCtx, cancel := context.WithTimeout(ctx, time.Second*10)
		block, err := client.BlockByNumber(reqCtx, blockNumber)
		cancel()

		if err == nil {
			return block, nil
		}

		lastErr = err
		if i < maxRetries-1 {
			backoff := time.Millisecond * 500 * time.Duration(i+1)
			log.Printf("[WARN] failed to fetch block %s, retry %d/%d after %v: %v",
				blockNumber.String(), i+1, maxRetries, backoff, err)
			time.Sleep(backoff)
		}
	}
	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

//func GetBlockWithRange(ctx context.Context, client *ethclient.Client, start, end uint64, duration time.Duration) (*types.Block, error) {
//}

func PrintBlockInfo(title string, block *types.Block) {
	fmt.Println("======================================")
	fmt.Println(title)
	fmt.Println("======================================")
	fmt.Printf("Block: %+v\n", block)

	// 基本信息
	fmt.Printf("Number       : %d\n", block.Number().Uint64())
	fmt.Printf("Hash         : %s\n", block.Hash().Hex())
	fmt.Printf("Parent Hash  : %s\n", block.ParentHash().Hex())

	// 时间信息
	blockTime := time.Unix(int64(block.Time()), 0)
	fmt.Printf("Time         : %s\n", blockTime.Format(time.RFC3339))
	fmt.Printf("Time (Local) : %s\n", blockTime.Local().Format("2006-01-02 15:04:05 MST"))

	// Gas 信息
	gasUsed := block.GasUsed()
	gasLimit := block.GasLimit()
	gasUsagePercent := float64(gasUsed) / float64(gasLimit) * 100
	fmt.Printf("Gas Used     : %d (%.2f%%)\n", gasUsed, gasUsagePercent)
	fmt.Printf("Gas Limit    : %d\n", gasLimit)

	// 交易信息
	txCount := len(block.Transactions())
	fmt.Printf("Tx Count     : %d\n", txCount)

	// 区块根信息（Merkle 树根）
	fmt.Printf("State Root   : %s\n", block.Root().Hex())
	fmt.Printf("Tx Root      : %s\n", block.TxHash().Hex())
	fmt.Printf("Receipt Root : %s\n", block.ReceiptHash().Hex())

	// 区块大小估算（简化版，实际大小还包括其他字段）
	if txCount > 0 {
		fmt.Printf("\nFirst Tx Hash: %s\n", block.Transactions()[0].Hash().Hex())
		if txCount > 1 {
			fmt.Printf("Last Tx Hash : %s\n", block.Transactions()[txCount-1].Hash().Hex())
		}
	}

	// 难度信息（PoW 相关，PoS 后基本固定）
	fmt.Printf("Difficulty   : %s\n", block.Difficulty().String())

	// 区块奖励相关信息
	coinbase := block.Coinbase()
	if coinbase != (common.Address{}) {
		fmt.Printf("Coinbase     : %s\n", coinbase.Hex())
	}

	fmt.Println("======================================")
	fmt.Println()
}

func SubscribeBlock(rpcUrl string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	headerChan := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(ctx, headerChan)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Subscribed to new blocks via %s\n", rpcUrl)

	// 捕获Ctrl+C
	cancelSig := make(chan os.Signal, 1)
	signal.Notify(cancelSig, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case header := <-headerChan:
			if header == nil {
				continue
			}
			fmt.Printf("[%s] New Block - Number: %d, Hash: %s\n",
				time.Now().Format(time.RFC3339),
				header.Number.Uint64(),
				header.Hash().Hex(),
			)
		case sig := <-cancelSig:
			fmt.Printf("received signal %s, shutting down...\n", sig.String())
			return
		case err := <-sub.Err():
			log.Fatal(err)
			return
		case <-ctx.Done():
			fmt.Println("context cancelled, exiting...")
			return
		}
	}
}
