package util

import (
	"context"
	"errors"
	"log"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

/**
 * @description: 去除字符串中的0x前缀
 * @param {string} str 字符串
 * @return {string} 去除0x后的字符串
 */
func TrimHexPrefix(str string) string {
	if len(str) > 2 && str[:2] == "0x" {
		return str[2:]
	}
	return str
}

/**
 * @description: 等待交易完成
 * @param {context.Context} ctx
 * @param {*ethclient.Client} client
 * @param {common.Hash} txHash
 */
func WaitForTransaction(ctx context.Context, client *ethclient.Client, txHash common.Hash) {
	// 超时时间
	waitCtx, cancel := context.WithTimeout(ctx, time.Minute*2)
	defer cancel()

	// 3s轮询一次
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	log.Printf("Waiting for receipt...\n")
	for {
		select {
		case <-waitCtx.Done():
			log.Printf("\nTimeout waiting for transaction confirmation.\n")
			return
		case <-ticker.C:
			receipt, err := client.TransactionReceipt(ctx, txHash)
			if err != nil {
				// 交易还在pending
				continue
			}

			// 交易已确认
			log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			log.Printf("Transaction Confirmed!\n")
			log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			log.Printf("Status       : %d (1=success, 0=failed)\n", receipt.Status)
			log.Printf("Block Number : %d\n", receipt.BlockNumber.Uint64())
			log.Printf("Block Hash   : %s\n", receipt.BlockHash.Hex())
			log.Printf("Gas Used     : %d / %d\n", receipt.GasUsed, receipt.GasUsed)
			log.Printf("Logs Count   : %d\n", len(receipt.Logs))

			if receipt.Status == 0 {
				// 交易失败
				log.Printf("Transaction failed!\n")
			} else {
				log.Printf("Transaction success!\n")
			}
			return
		}
	}
}

/**
 * @description: 解析代币金额
 * @param {string} amountStr 金额字符串
 * @param {uint8} decimals 代币精度
 * @return {*big.Int, error} 金额, 错误
 */
func ParseTokenAmount(amountStr string, decimals uint8) (*big.Int, error) {
	if strings.Contains(amountStr, ".") {
		// 包含小数点，解析为浮点数
		amountFloat, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return nil, err
		}

		bigFloat := big.NewFloat(amountFloat)

		multiplier := new(big.Float).SetFloat64(math.Pow10(int(decimals)))
		// bigFloat*=multiplier
		bigFloat.Mul(bigFloat, multiplier)

		amount, _ := bigFloat.Int(nil)
		return amount, nil
	} else {
		// 解析为整数
		amount, ok := new(big.Int).SetString(amountStr, 10)
		if !ok {
			return nil, errors.New("invalid amount")
		}
		return amount, nil
	}
}
