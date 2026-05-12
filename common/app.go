package common

import (
	"awesomeProject/pool"
	"awesomeProject/util"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type App struct {
	urls        []string
	pool        *pool.ChainNodePool
	currentMenu uint8
}

func NewApp(urls []string) (*App, error) {
	pool, err := pool.NewETHClientPool(urls)
	if err != nil {
		return nil, err
	}
	return &App{urls: urls, pool: pool}, nil
}

/**
 * @description: 主菜单
 * @receiver a *App
 * @return uint8
 */
func (a *App) Menu() uint8 {
	fmt.Println("=============== 请选择要执行的操作 ===============")
	fmt.Println("请输入菜单编号(1-4):")
	fmt.Println("1. 获取区块信息，默认为最新区块")
	fmt.Println("2. 向指定地址转账")
	fmt.Println("3. 与指定ERC20合约交互")
	fmt.Println("4. 退出")
	fmt.Println("请输入菜单编号:")

	var menu uint8
	fmt.Scanln(&menu)
	return menu
}

/**
 * @description: 获取用户输入的合约操作
 * @receiver a *App
 * @return string
 */
func (a *App) InputContractOperation() (operation string) {
	fmt.Println("请输入操作(balanceof, decimals, transfer):")
	fmt.Scanln(&operation)
	return operation
}

/**
 * @description: 获取用户输入的区块高度
 * @receiver a *App
 * @return string, error
 */
func (a *App) InputBlockNumber() (string, error) {
	fmt.Println("请输入区块高度:")
	var blockNumber string
	fmt.Scanln(&blockNumber)
	if blockNumber == "" {
		return "", errors.New("区块号不能为空")
	}
	return blockNumber, nil
}

/**
 * @description: 获取用户输入的转账信息
 * @receiver a *App
 * @return string, string, string
 */
func (a *App) InputTransferInfo() (privateKeyHex string, toAddressHex string, amountETHStr string) {
	fmt.Println("请输入私钥:")
	fmt.Scanln(&privateKeyHex)
	fmt.Println("请输入转账地址:")
	fmt.Scanln(&toAddressHex)
	fmt.Println("请输入转账金额:")
	fmt.Scanln(&amountETHStr)
	return privateKeyHex, toAddressHex, amountETHStr
}

/**
 * @description: 获取用户输入的合约地址
 * @receiver a *App
 * @return string
 */
func (a *App) InputContractAddress() (contractAddressHex string) {
	fmt.Println("请输入合约地址:")
	fmt.Scanln(&contractAddressHex)
	return contractAddressHex
}

/**
 * @description: 根据区块高度获取区块信息
 * @receiver a *App
 * @param blockNumber string
 * @return *types.Block, error
 */
func (a *App) SearchBlock(blockNumber string) (block *types.Block, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	block, err = a.pool.GetBlockByNumber(ctx, blockNumber)
	if err != nil {
		return nil, err
	}
	return block, nil
}

/**
 * @description: 向指定地址转账
 * @receiver a *App
 * @param privateKeyHex string
 * @param toAddressHex string
 * @param amountETH string
 * @return error
 */
func (a *App) Transfer(privateKeyHex string, toAddressHex string, amountETH string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	err = a.pool.Transfer(ctx, privateKeyHex, toAddressHex, amountETH)
	if err != nil {
		return err
	}
	return nil
}

/**
 * @description: 与指定ERC20合约交互
 * @receiver a *App
 * @param contractAddressHex string
 * @param operation string
 * @return error
 */
func (a *App) Contract(contractAddressHex, operation string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	node := a.pool.GetPrimaryNode()
	if node == nil {
		return errors.New("no available node")
	}

	contractAddress := common.HexToAddress(contractAddressHex)
	contractABI, err := NewMyToken(contractAddress, node.Client)
	if err != nil {
		return err
	}
	operation = strings.ToLower(operation)
	switch operation {
	case "name":
		value, err := contractABI.Name(&bind.CallOpts{})
		if err != nil {
			return err
		}
		fmt.Printf("name: %+v\n", value)
	case "symbol":
		value, err := contractABI.Symbol(&bind.CallOpts{})
		if err != nil {
			return err
		}
		fmt.Printf("symbol: %+v\n", value)
	case "totalSupply":
		value, err := contractABI.TotalSupply(&bind.CallOpts{})
		if err != nil {
			return err
		}
		fmt.Printf("totalSupply: %+v\n", value)
	case "owner":
		value, err := contractABI.Owner(&bind.CallOpts{})
		if err != nil {
			return err
		}
		fmt.Printf("owner: %+v\n", value)
	case "balanceof":
		var address string
		fmt.Println("请输入地址:")
		fmt.Scanln(&address)
		value, err := contractABI.BalanceOf(&bind.CallOpts{}, common.HexToAddress(address))
		if err != nil {
			return err
		}
		fmt.Printf("balanceOf: %+v\n", value)
	case "decimals":
		value, err := contractABI.Decimals(&bind.CallOpts{})
		if err != nil {
			return err
		}
		fmt.Printf("decimals: %+v\n", value)
	case "transfer":
		chainId, err := node.Client.ChainID(ctx)
		if err != nil {
			return err
		}
		var privateKeyHex string
		fmt.Println("请输入私钥:")
		fmt.Scanln(&privateKeyHex)
		privateKey, err := crypto.HexToECDSA(privateKeyHex)
		if err != nil {
			return err
		}
		transactor, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
		if err != nil {
			return err
		}
		var toAddress string
		fmt.Println("请输入转账地址:")
		fmt.Scanln(&toAddress)
		var amount string
		fmt.Println("请输入转账金额:")
		fmt.Scanln(&amount)
		amountBigInt, err := util.ParseTokenAmount(amount, 18)
		if err != nil {
			return err
		}
		value, err := contractABI.Transfer(transactor, common.HexToAddress(toAddress), amountBigInt)
		if err != nil {
			return err
		}
		util.WaitForTransaction(ctx, node.Client, value.Hash())
		fmt.Printf("transfer success, tx hash=%s\n", value.Hash().Hex())
	default:
		return errors.New("invalid operation")
	}
	return nil
}
