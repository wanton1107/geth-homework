package main

import (
	"awesomeProject/common"
	"fmt"
	"log"
	"time"
)

//const RPC_URL = "https://sepolia.drpc.org"
//const RPC_URL_WSS = "wss://sepolia.drpc.org"

// const RPC_URL = "http://127.0.0.1:8545"
// const RPC_URL_WSS = "ws://127.0.0.1:8545"

var urls = []string{
	"https://sepolia.drpc.org",
	"https://ethereum-sepolia-rpc.publicnode.com",
	"https://eth-sepolia-testnet.api.pocket.network",
}

func main() {
	// 仅用于初次连接各 RPC；连接完成后应 cancel，避免把短生命周期 context 绑在整个 App 上。
	app, err := common.NewApp(urls)
	if err != nil {
		log.Fatal(err)
	}
	for {
		menu := app.Menu()
		if menu == 1 {
			blockNumber, err := app.InputBlockNumber()
			if err != nil {
				log.Fatal(err)
			}
			block, err := app.SearchBlock(blockNumber)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("\nblock hash: %+v\n", block.Hash().Hex())
			fmt.Printf("\nblock parent hash: %+v\n", block.ParentHash().Hex())
			fmt.Printf("\nblock number: %+v\n", block.Number().Uint64())
			fmt.Printf("\nblock time: %+v\n", time.Unix(int64(block.Time()), 0).Format("2006-01-02 15:04:05"))
			fmt.Printf("\nblock gas limit: %+v\n", block.GasLimit())
			fmt.Printf("\nblock gas used: %+v\n", block.GasUsed())
		} else if menu == 2 {
			privateKeyHex, toAddressHex, amountETHStr := app.InputTransferInfo()
			err := app.Transfer(privateKeyHex, toAddressHex, amountETHStr)
			if err != nil {
				log.Fatal(err)
			}
		} else if menu == 3 {
			contractAddressHex := app.InputContractAddress()
			operation := app.InputContractOperation()
			err := app.Contract(contractAddressHex, operation)
			if err != nil {
				log.Fatal(err)
			}
		} else if menu == 4 {
			break
		}
	}
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	// defer cancel()
	// urls := []string{"https://0xrpc.io/sep", "https://ethereum-sepolia-rpc.publicnode.com", "https://eth-sepolia-testnet.api.pocket.network"}
	// pool, err := pool.NewETHClientPool(ctx, urls)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// for i := 0; i < 3; i++ {
	// 	blockNumber, err := pool.GetLastestBlockNumber(ctx)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	fmt.Printf("\nblockNumber: %v\n", blockNumber.String())
	// }
	// defer pool.Close()
	//common.SubscribeLogs(RPC_URL_WSS)
	//common.SubscribeBlock(RPC_URL_WSS)
	//web3.ConnectDemo(RPC_URL, "")
	//timeout, cancel := context.WithTimeout(context.Background(), time.Second*20)
	//defer cancel()
	//client, err := ethclient.DialContext(timeout, RPC_URL)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//defer client.Close()
	//
	//file, err := os.ReadFile("./contracts/MyToken.abi.json")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//parsedABI, err := abi.JSON(strings.NewReader(string(file)))
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//contractAddr := "0x610178dA211FEF7D417bC0e6FeD39F05609AD788"
	//targetAddr := "0x70997970c51812dc3a010c7d01b50e0d17dc79c8"
	//balance, err := common.BalanceOf(timeout, client, &parsedABI, contractAddr, targetAddr)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//log.Printf("Balance: %v", balance.String())
	//common.Transfer(timeout, client, &parsedABI, contractAddr, targetAddr, "1.0")
	//balance, err = common.BalanceOf(timeout, client, &parsedABI, contractAddr, targetAddr)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//log.Printf("Balance: %v", balance.String())

	// value := common.BalanceOfWithABIGEN("0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266")
	// fmt.Println(value)
	//
	//txHash, err := common.SendTransaction("0x70997970c51812dc3a010c7d01b50e0d17dc79c8", 0.5)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//log.Println("txtxtx:", txHash.Hash().Hex())
	//common.QueryTxByHash(timeout, client, txHash.Hash().Hex())fmt
}
