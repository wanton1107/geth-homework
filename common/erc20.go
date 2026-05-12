package common

import (
	"awesomeProject/util"
	"context"
	"crypto/ecdsa"
	"errors"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
)

func BalanceOf(ctx context.Context, client *ethclient.Client, parsedABI *abi.ABI, contractAddressHex, addrHex string) (*big.Int, error) {
	if contractAddressHex == "" || addrHex == "" {
		log.Fatal("contractAddress or addrHex is empty")
	}

	contractAddress := common.HexToAddress(contractAddressHex)
	targetAddress := common.HexToAddress(addrHex)

	data, err := parsedABI.Pack("balanceOf", targetAddress)
	if err != nil {
		log.Fatal(err)
	}

	callMsg := ethereum.CallMsg{
		To:   &contractAddress,
		Data: data,
	}

	output, err := client.CallContract(ctx, callMsg, nil)
	if err != nil {
		log.Fatal(err)
	}

	var balance *big.Int
	err = parsedABI.UnpackIntoInterface(&balance, "balanceOf", output)
	if err != nil {
		log.Fatal(err)
	}

	return balance, nil

}

func getTokenDecimal(ctx context.Context, client *ethclient.Client, parsedABI *abi.ABI, contractAddressHex string) (uint8, error) {
	data, err := parsedABI.Pack("decimals")
	if err != nil {
		log.Fatal(err)
	}

	contractAddress := common.HexToAddress(contractAddressHex)

	callMsg := ethereum.CallMsg{
		To:   &contractAddress,
		Data: data,
	}

	output, err := client.CallContract(ctx, callMsg, nil)
	if err != nil {
		log.Fatal(err)
	}

	var decimals uint8
	err = parsedABI.UnpackIntoInterface(&decimals, "decimals", output)
	if err != nil {
		log.Fatal(err)
	}

	return decimals, nil
}

func Transfer(ctx context.Context, client *ethclient.Client, parsedABI *abi.ABI, contractAddressHex, toAddressHex string, amountStr string) {
	// 私钥
	privateKeyHex := "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

	// 解析私钥
	privateKey, err := crypto.HexToECDSA(util.TrimHexPrefix(privateKeyHex))
	if err != nil {
		log.Fatal(err)
	}

	// 获取公钥
	publicKey := privateKey.Public()
	// 解析公钥，publicKey为接口，断言类型转换
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	// 获取地址
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	contractAddress := common.HexToAddress(contractAddressHex)
	toAddress := common.HexToAddress(toAddressHex)

	// 查询代币经度
	decimal, err := getTokenDecimal(ctx, client, parsedABI, contractAddressHex)
	if err != nil {
		log.Fatal(err)
	}

	// 解析金额，如果包含小数点则乘以decimals，否则解析为最小单位的整数
	amount, err := parseTokenAmount(amountStr, decimal)
	if err != nil {
		log.Fatal(err)
	}

	// 获取链ID和nonce
	chainId, err := client.ChainID(ctx)
	if err != nil {
		log.Fatal(err)
	}
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		log.Fatal(err)
	}

	// 打包函数调用和参数
	callData, err := parsedABI.Pack("transfer", toAddress, amount)
	if err != nil {
		log.Fatal(err)
	}

	// 计算gas数量
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From: fromAddress,
		To:   &contractAddress,
		Data: callData,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 增加20%缓冲
	gasLimit = gasLimit * 120 / 100

	// 获取gas价格
	gasTipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 获取base fee
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	baseFee := header.BaseFee
	if baseFee == nil {
		gasPrice, err := client.SuggestGasPrice(ctx)
		if err != nil {
			log.Fatal(err)
		}
		baseFee = gasPrice
	}

	// 计算gas总数
	// baseFee*2+gasTipCap
	gasFeeCap := new(big.Int).Add(
		gasTipCap, new(big.Int).Mul(baseFee, big.NewInt(2)))

	// 计算总费用
	totalGasCost := new(big.Int).Mul(gasFeeCap, big.NewInt(int64(gasLimit)))

	// 查询余额
	balance, err := client.BalanceAt(ctx, fromAddress, nil)
	if err != nil {
		log.Fatal(err)
	}

	if balance.Cmp(totalGasCost) < 0 {
		// 余额不足
		log.Fatal("not enough balance")
	}

	// 构造交易
	txData := types.DynamicFeeTx{
		ChainID:   chainId,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &contractAddress,
		Value:     big.NewInt(0),
		Data:      callData,
	}

	tx := types.NewTx(&txData)

	// 签名
	signer := types.NewLondonSigner(chainId)
	signTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		log.Fatal(err)
	}

	// 发送交易
	err = client.SendTransaction(ctx, signTx)
	if err != nil {
		log.Fatal(err)
	}

	// 输出交易信息
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	log.Printf("ERC-20 Transfer Transaction Sent\n")
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	log.Printf("From          : %s\n", fromAddress.Hex())
	log.Printf("To            : %s\n", toAddress.Hex())
	log.Printf("Contract      : %s\n", contractAddress.Hex())
	log.Printf("Token Decimals: %d\n", decimal)
	// 显示代币数量（根据 decimals 转换）
	log.Printf("Amount        : %s tokens (raw units)\n", amount.String())
	log.Printf("Gas Limit     : %d\n", gasLimit)
	log.Printf("Gas Tip Cap   : %s Wei\n", gasTipCap.String())
	log.Printf("Gas Fee Cap   : %s Wei\n", gasFeeCap.String())
	log.Printf("Estimated Cost: %s Wei\n", totalGasCost.String())
	log.Printf("Nonce         : %d\n", nonce)
	log.Printf("Tx Hash       : %s\n", signTx.Hash().Hex())
	log.Printf("\n")
	log.Printf("Transaction is pending. Waiting for confirmation...\n")
	log.Printf("\n")

	waitForTransaction(ctx, client, signTx.Hash())
}

// 等待交易完成
func waitForTransaction(ctx context.Context, client *ethclient.Client, txHash common.Hash) {
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

func parseTokenAmount(amountStr string, decimals uint8) (*big.Int, error) {
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
