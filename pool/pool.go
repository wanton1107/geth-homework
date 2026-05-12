package pool

import (
	"awesomeProject/util"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ChainNode struct {
	Url    string
	Client *ethclient.Client
	Active bool
}

type ChainNodePool struct {
	sync.RWMutex
	Nodes       []*ChainNode
	PrimaryIdx  int
	ReadOnlyIdx int
}

func NewETHClientPool(urls []string) (*ChainNodePool, error) {
	if len(urls) == 0 {
		return nil, errors.New("urls is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nodes := make([]*ChainNode, 0, len(urls))

	for _, url := range urls {
		client, err := ethclient.DialContext(ctx, url)
		if err != nil {
			chainNode := &ChainNode{Url: url, Client: nil, Active: false}
			nodes = append(nodes, chainNode)
			log.Printf("[WARN] connect rpc failed, url=%s, err=%v", url, err)
			continue
		}
		chainNode := &ChainNode{Url: url, Client: client, Active: true}
		nodes = append(nodes, chainNode)
		log.Printf("[INFO] connect rpc success, url=%s", url)
	}

	if len(nodes) == 0 {
		return nil, errors.New("all nodes are not available")
	}

	return &ChainNodePool{Nodes: nodes, PrimaryIdx: 0, ReadOnlyIdx: 0}, nil
}

func (p *ChainNodePool) GetReadOnlyNode() *ChainNode {
	p.RLock()
	defer p.RUnlock()

	n := len(p.Nodes)
	for i := 0; i < n; i++ {
		// 从当前位置往后查找可用节点
		currentIdx := (p.ReadOnlyIdx + i) % n
		node := p.Nodes[currentIdx]
		if node.Active && node.Client != nil {
			// 更新当前位置
			p.ReadOnlyIdx = (currentIdx + 1) % n
			log.Printf("[INFO] get readonly node success, url=%s", node.Url)
			return node
		}
	}
	log.Printf("[WARN] all readonly nodes are not available")
	return nil
}

func (p *ChainNodePool) GetPrimaryNode() *ChainNode {
	p.RLock()
	defer p.RUnlock()

	n := len(p.Nodes)

	if n > 0 && p.PrimaryIdx < n {
		node := p.Nodes[p.PrimaryIdx]
		if node.Active && node.Client != nil {
			log.Printf("[INFO] get primary node success, url=%s", node.Url)
			return node
		}
	}

	for i := 0; i < n; i++ {
		node := p.Nodes[i]
		if node.Active && node.Client != nil {
			p.PrimaryIdx = i
			log.Printf("[INFO] get primary node success, url=%s", node.Url)
			return node
		}
	}
	log.Printf("[WARN] all primary nodes are not available")
	return nil
}

func (p *ChainNodePool) markNodeDeaded(url string, cause error) {
	p.Lock()
	defer p.Unlock()

	for _, node := range p.Nodes {
		if node.Url == url {
			if node.Active {
				log.Printf("[WARN] node %s is deaded, cause=%v", url, cause)
				node.Active = false
				node.Client = nil
			}
			return
		}
	}
}

/**
 * @description: 获取指定地址的余额
 * @param {context.Context} ctx
 * @param {string} address 地址
 * @return {*big.Int, error} 余额, 错误
 */
func (p *ChainNodePool) GetBalance(ctx context.Context, address string) (*big.Int, error) {
	node := p.GetReadOnlyNode()
	if node == nil {
		return nil, errors.New("no available node")
	}

	balance, err := node.Client.BalanceAt(ctx, common.HexToAddress(address), nil)
	if err != nil {
		p.markNodeDeaded(node.Url, err)
		return nil, err
	}
	return balance, nil
}

/**
 * @description: 获取最新区块高度
 * @param {context.Context} ctx
 * @return {*big.Int, error} 最新区块高度, 错误信息
 */
func (p *ChainNodePool) GetLastestBlockNumber(ctx context.Context) (*big.Int, error) {
	node := p.GetReadOnlyNode()
	if node == nil {
		return nil, errors.New("no available node")
	}

	blockNumber, err := node.Client.BlockNumber(ctx)
	if err != nil {
		p.markNodeDeaded(node.Url, err)
		return nil, err
	}
	return new(big.Int).SetUint64(blockNumber), nil
}

/**
 * @description: 获取指定区块高度的区块信息
 * @param {context.Context} ctx
 * @param {string} blockNumber 区块高度
 * @return {*types.Block, error} 区块信息, 错误信息
 */
func (p *ChainNodePool) GetBlockByNumber(ctx context.Context, blockNumber string) (*types.Block, error) {
	node := p.GetReadOnlyNode()
	if node == nil {
		return nil, errors.New("no available node")
	}

	num, ok := new(big.Int).SetString(blockNumber, 10)
	if !ok {
		return nil, fmt.Errorf("invalid block number: %s", blockNumber)
	}
	block, err := node.Client.BlockByNumber(ctx, num)
	if err != nil {
		p.markNodeDeaded(node.Url, err)
		return nil, err
	}
	return block, nil
}

/**
 * @description: 向指定地址转账
 * @param {context.Context} ctx
 * @param {string} privateKeyHex 私钥
 * @param {string} toAddressHex 转账地址
 * @param {float64} amountETH 转账金额
 * @return {error} 错误信息
 */
func (p *ChainNodePool) Transfer(ctx context.Context, privateKeyHex string, toAddressHex string, amountETH string) error {
	node := p.GetPrimaryNode()
	if node == nil {
		return errors.New("no available node")
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return err
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return errors.New("error casting public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	toAddress := common.HexToAddress(toAddressHex)

	balance, err := node.Client.BalanceAt(ctx, fromAddress, nil)
	if err != nil {
		return err
	}

	amount, err := util.ParseTokenAmount(amountETH, 18)
	if err != nil {
		return err
	}

	chainId, err := node.Client.ChainID(ctx)
	if err != nil {
		return err
	}

	nonce, err := node.Client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return err
	}

	gasTipCap, err := node.Client.SuggestGasTipCap(ctx)
	if err != nil {
		return err
	}
	gasLimit := 21000

	header, err := node.Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}

	baseFee := header.BaseFee
	if baseFee == nil {
		// 不支持EIP1559,使用传统方法计算
		gasPrice, err := node.Client.SuggestGasPrice(ctx)
		if err != nil {
			return err
		}
		baseFee = gasPrice
	}

	// 计算gas总用量
	// baseFee*2+gasTipCap
	gasFeeCap := new(big.Int).Add(gasTipCap, new(big.Int).Mul(baseFee, big.NewInt(2)))

	// 计算gas总费用
	totalGasCost := new(big.Int).Mul(gasFeeCap, big.NewInt(int64(gasLimit)))

	total := new(big.Int).Add(amount, totalGasCost)

	if balance.Cmp(total) < 0 {
		return errors.New("not enough balance")
	}

	txData := &types.DynamicFeeTx{
		ChainID:   chainId,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       uint64(gasLimit),
		To:        &toAddress,
		Value:     amount,
		Data:      nil,
	}
	tx := types.NewTx(txData)

	signer := types.NewLondonSigner(chainId)
	signTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return err
	}

	err = node.Client.SendTransaction(ctx, signTx)
	if err != nil {
		return err
	}

	util.WaitForTransaction(ctx, node.Client, signTx.Hash())
	log.Printf("[INFO] transfer success, tx hash=%s", signTx.Hash().Hex())
	return nil
}
