package common

import (
	"awesomeProject/util"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func QueryTxByHash(ctx context.Context, client *ethclient.Client, hexHash string) {
	hash := common.HexToHash(hexHash)
	tx, pending, err := client.TransactionByHash(ctx, hash)

	if err != nil {
		log.Println("QueryTxByHash err:", err)
	}

	log.Println("=============== Transaction ==================")
	printTxBasicInfo(tx, pending)

	receipt, err := client.TransactionReceipt(ctx, hash)
	if err != nil {
		log.Printf("failed to get receipt (maybe pending): %v", err)
		return
	}

	log.Println("=============== Transaction Receipt ==================")
	printReceiptInfo(receipt)
}

func SendTransaction(toAddressHex string, amountETH float64) (*types.Transaction, error) {
	const privateKeyHex = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := ethclient.DialContext(ctx, "http://127.0.0.1:8545")
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// 解析私钥，去除0x
	privateKey, err := crypto.HexToECDSA(util.TrimHexPrefix(privateKeyHex))
	if err != nil {
		return nil, err
	}

	// 通过私钥解析地址
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	toAddress := common.HexToAddress(toAddressHex)

	// 获取链ID
	chainId, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	// 获取发送方nonce
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return nil, err
	}

	// 获取建议的 Gas 价格（使用 EIP-1559 动态费用）
	gasTipPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	// 获取BaseFee
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	baseFee := header.BaseFee
	if baseFee == nil {
		// 不支持EIP1559,使用传统方法计算
		gasPrice, err := client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, err
		}
		baseFee = gasPrice
	}

	// fee cap = base fee * 2 + tip cap（简单策略）
	feeCap := new(big.Int).Add(gasTipPrice, new(big.Int).Mul(baseFee, big.NewInt(2)))
	gasLimit := 21000

	// 计算转账金额
	amountWei := new(big.Float).Mul(
		big.NewFloat(amountETH), big.NewFloat(1e18))
	valueWei, _ := amountWei.Int(nil)

	// 检查余额
	// total=转账金额+gasUsed*gasPrice
	balance, err := client.BalanceAt(ctx, fromAddress, nil)
	if err != nil {
		return nil, err
	}

	// 计算所需总额
	total := new(big.Int).Add(
		valueWei, new(big.Int).Mul(feeCap, big.NewInt(int64(gasLimit))))
	if balance.Cmp(total) < 0 {
		return nil, errors.New("insufficient funds")
	}

	txData := &types.DynamicFeeTx{
		ChainID:   chainId,
		Nonce:     nonce,
		GasTipCap: gasTipPrice,
		GasFeeCap: feeCap,
		Gas:       uint64(gasLimit),
		To:        &toAddress,
		Value:     valueWei,
		Data:      nil,
	}
	tx := types.NewTx(txData)

	// 对交易签名
	singer := types.NewLondonSigner(chainId)
	signTx, err := types.SignTx(tx, singer, privateKey)
	if err != nil {
		return nil, err
	}

	// 发送交易
	err = client.SendTransaction(ctx, signTx)
	if err != nil {
		return nil, err
	}

	// 输出交易信息
	log.Println("=== Transaction Sent ===")
	log.Printf("From       : %s\n", fromAddress.Hex())
	log.Printf("To         : %s\n", toAddress.Hex())
	log.Printf("Value      : %s ETH (%s Wei)\n", fmt.Sprintf("%.6f", amountETH), valueWei.String())
	log.Printf("Gas Limit  : %d\n", gasLimit)
	log.Printf("Gas Tip Cap: %s Wei\n", gasTipPrice.String())
	log.Printf("Gas Fee Cap: %s Wei\n", feeCap.String())
	log.Printf("Nonce      : %d\n", nonce)
	log.Printf("Tx Hash    : %s\n", signTx.Hash().Hex())
	// log.Println("\nTransaction is pending. Use --tx flag to query status:")
	// log.Printf("  go run main.go --tx %s\n", signTx.Hash().Hex())

	return signTx, nil
}

func printTxBasicInfo(tx *types.Transaction, isPending bool) {
	log.Printf("Hash        : %s\n", tx.Hash().Hex())
	log.Printf("Nonce       : %d\n", tx.Nonce())
	log.Printf("Gas         : %d\n", tx.Gas())
	log.Printf("Gas Price   : %s\n", tx.GasPrice().String())
	log.Printf("To          : %v\n", tx.To())
	log.Printf("Value (Wei) : %s\n", tx.Value().String())
	log.Printf("Data Len    : %d bytes\n", len(tx.Data()))
	log.Printf("Pending     : %v\n", isPending)
}

func printReceiptInfo(r *types.Receipt) {
	log.Printf("Status      : %d\n", r.Status)
	log.Printf("BlockNumber : %d\n", r.BlockNumber.Uint64())
	log.Printf("BlockHash   : %s\n", r.BlockHash.Hex())
	log.Printf("TxIndex     : %d\n", r.TransactionIndex)
	log.Printf("Gas Used    : %d\n", r.GasUsed)
	log.Printf("Logs        : %d\n", len(r.Logs))
	if len(r.Logs) > 0 {
		log.Printf("First Log Address : %s\n", r.Logs[0].Address.Hex())
	}
}
