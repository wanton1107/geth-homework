package web3

import (
	"awesomeProject/common"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
	"time"
)

func ConnectDemo(rpcUrl, privateKey string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcUrl)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Network connected!")
	defer client.Close()

	chainId, err := client.ChainID(ctx)
	if err != nil {
		log.Fatal(err)
	}

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("RPC URL:%s\n", rpcUrl)
	log.Printf("Chain ID:%s\n", chainId.String())
	log.Printf("Latest block number:%d\n", header.Number.Uint64())
	log.Printf("Latest block hash:%s\n", header.Hash().Hex())
	log.Printf("Latest block timestamp:%s\n", time.Unix(int64(header.Time), 0).Format("2006-01-02 15:04:05"))

	// 上一个区块
	if header.Number.Uint64() > 0 {
		lastNum := new(big.Int).Sub(header.Number, big.NewInt(1))
		if prevBlock, err := client.HeaderByNumber(ctx, lastNum); err == nil {
			fmt.Printf("Previous block number:%d\n", prevBlock.Number.Uint64())
			fmt.Printf("Previous block hash:%s\n", prevBlock.Hash().Hex())
		}
	}

	// safe区块
	safeBlock, safeHash, err := common.GetBlockByTag(ctx, client, "safe")
	if err != nil {
		log.Printf("GetBlockByTag err:%s", err.Error())
	} else {
		fmt.Printf("\n============ SafeBlock =============\n")
		fmt.Printf("Block Number:%d\n", safeBlock.Number.Uint64())
		fmt.Printf("Block Hash:%s\n", safeHash.Hex())
		fmt.Printf("Calculated Hash:%s\n", safeBlock.Hash().Hex())
		fmt.Printf("Block Timestamp:%s\n", time.Unix(int64(safeBlock.Time), 0).Format("2006-01-02 15:04:05"))
		fmt.Printf("Comfirmations:%d\n", header.Number.Uint64()-safeBlock.Number.Uint64())
		fmt.Printf("======================================")
	}

	// finalized区块
	finalizedHeader, finalizedHash, err := common.GetBlockByTag(ctx, client, "finalized")
	if err != nil {
		log.Printf("GetBlockByTag err:%s", err.Error())
	} else {
		fmt.Printf("\n============ FinalizedBlock =============\n")
		fmt.Printf("Block Number:%d\n", finalizedHeader.Number.Uint64())
		fmt.Printf("Block Hash:%s\n", finalizedHash.Hex())
		fmt.Printf("Calculated Hash:%s\n", finalizedHeader.Hash().Hex())
		fmt.Printf("Block Timestamp:%s\n", time.Unix(int64(finalizedHeader.Time), 0).Format("2006-01-02 15:04:05"))
		fmt.Printf("Comfirmations:%d\n", header.Number.Uint64()-finalizedHeader.Number.Uint64())
		fmt.Printf("======================================")
	}
}
