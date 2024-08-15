package main

import (
	"context"
	// "context"
	// "encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	// "github.com/gorilla/websocket"
	"github.com/wewe/abis"
	"log"
	"math/big"
	"os"
	"strings"
	"time"
)

func main() {
	fmt.Println("hello world")

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	privateKeyStr := os.Getenv("PRIVATE_KEY")
	routerAddr := os.Getenv("ROUTER")
	wethAddr := os.Getenv("WETH")
	vultAddr := os.Getenv("VULT")
	amountInConfig := os.Getenv("AMOUNT_IN")
	amountOutExpect := os.Getenv("AMOUNT_EXPECTED")
	gasConfig := os.Getenv("GAS_PRICE")

	routerContract := common.HexToAddress(routerAddr)
	wethToken := common.HexToAddress(wethAddr)
	vultToken := common.HexToAddress(vultAddr)

	// usdt: 0xdAC17F958D2ee523a2206206994597C13D831ec7
	// vult: xxxx

	fee := big.NewInt(3000)
	amountETHIn := big.NewInt(0)
	amoutOut := big.NewInt(0)
	gasPrice := big.NewInt(0)

	amountETHIn.SetString(amountInConfig, 10)
	amoutOut.SetString(amountOutExpect, 10)
	gasPrice.SetString(gasConfig, 10)

	rotateEndoints := []string{
		"https://eth.llamarpc.com",
		"https://rpc.ankr.com/eth",
		"https://eth.drpc.org",
		"https://eth.meowrpc.com",
	}

	if len(rotateEndoints) == 0 {
		log.Panic("can u fucking config endpoint pls")
	}

	currentEndPointIndex := 0
	clientInst, err := ethclient.Dial(rotateEndoints[currentEndPointIndex])
	if err != nil {
		log.Panic("you asshole, config death endpoint ", err)
	}

	privateKey, err := crypto.HexToECDSA(privateKeyStr)
	if err != nil {
		log.Panic(err.Error())
	}

	chainId, _ := clientInst.ChainID(context.Background())
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		log.Panic(err.Error())
	}

	// timer
	channel1 := make(chan string)
	pushInChannel := func() {
		channel1 <- "loop"
	}
	// contract router
	uniV3Router, err := abis.NewRouter(routerContract, clientInst)
	if err != nil {
		log.Panic(err.Error())
	}

	// single swap
	singleSwapData := abis.IV3SwapRouterExactInputSingleParams{
		TokenIn:           wethToken,
		TokenOut:          vultToken,
		Fee:               fee,
		Recipient:         auth.From,
		AmountIn:          amountETHIn,
		AmountOutMinimum:  amoutOut,
		SqrtPriceLimitX96: big.NewInt(0),
	}

	routerABI, err := abi.JSON(strings.NewReader(abis.RouterMetaData.ABI))
	if err != nil {
		log.Fatal(err)
	}

	//uniV3Router.ExactInputSingle()
	swapSingleInputData, err := routerABI.Pack("exactInputSingle", singleSwapData)
	if err != nil {
		log.Fatal(err)
	}

	// sweep token
	refundETHData, err := routerABI.Pack("refundETH")
	if err != nil {
		log.Fatal(err)
	}

	go pushInChannel()
	for {
		//
		v := <-channel1
		fmt.Println(v)
		auth.GasPrice = gasPrice
		auth.Value = amountETHIn
		fmt.Println("value: ", amountETHIn.String())
		fmt.Println("gasPrice: ", auth.GasPrice)
		fmt.Println("from: ", auth.From.String())

		tx, err := uniV3Router.Multicall1(auth, [][]byte{swapSingleInputData, refundETHData})
		if err != nil {
			fmt.Println(err.Error())
			if err.Error() != "execution reverted" {
				// optional
				for {
					currentEndPointIndex = (currentEndPointIndex + 1) % len(rotateEndoints)
					fmt.Printf("new endpoint %v \n", rotateEndoints[currentEndPointIndex])
					clientInst, err = ethclient.Dial(rotateEndoints[currentEndPointIndex])
					if err != nil {
						continue
					}

					break
				}

				go pushInChannel()
				continue
			}
		} else {
			fmt.Println("success created tx")
			fmt.Println(tx.Hash().String())
			// check tx success
		}
		time.Sleep(5 * time.Second)
		go pushInChannel()
	}
}
