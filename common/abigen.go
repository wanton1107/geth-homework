package common

import (
	"context"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func BalanceOfWithABIGEN(address string) *big.Int {
	contractAddressHex := "0x610178dA211FEF7D417bC0e6FeD39F05609AD788"
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	client, err := ethclient.DialContext(ctx, "http://127.0.0.1:8545")
	if err != nil {
		log.Fatal(err)
	}

	contractAddress := common.HexToAddress(contractAddressHex)

	contractABI, err := NewMyTokenCaller(contractAddress, client)
	if err != nil {
		log.Fatal(err)
	}

	value, err := contractABI.BalanceOf(&bind.CallOpts{}, common.HexToAddress(address))
	if err != nil {
		log.Fatal(err)
	}

	return value

}
