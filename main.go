package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/websocket"
	"github.com/wewe/abis"
	"log"
	"math/big"
	"os"
	"strings"
	"time"
)

type EthSubscribePayload struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type EthSubscribeTxResult struct {
	Method string        `json:"method"`
	Params ParamTxResult `json:"params"`
}
type ParamTxResult struct {
	Result       jsonTx `json:"result"`
	Subscription string `json:"subscription"`
}

type jsonTx struct {
	BlockHash            *common.Hash    `json:"blockHash"`
	BlockNumber          *big.Int        `json:"blockNumber"`
	From                 common.Address  `json:"from"`
	Gas                  string          `json:"gas"`
	GasPrice             string          `json:"gasPrice"`
	MaxFeePerGas         string          `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string          `json:"maxPriorityFeePerGas"`
	Hash                 common.Hash     `json:"hash"`
	Input                string          `json:"input"`
	Nonce                string          `json:"nonce"`
	To                   *common.Address `json:"to"`
	TransactionIndex     *uint           `json:"transactionIndex"`
	Value                string          `json:"value"`
	Type                 string          `json:"type"`
	ChainId              string          `json:"chainId"`
	V                    string          `json:"v"`
	R                    string          `json:"r"`
	S                    string          `json:"s"`
	YParity              string          `json:"yParity"`
}

func main() {
	fmt.Println("hello world")

	privateKeyStr := os.Getenv("PRIVATE_KEY")
	weweToken := common.HexToAddress("0x6b9bb36519538e0c073894e964e90172e1c0b41f")
	//mergeContractAddr := common.HexToAddress("0x157ECfd5bAB3d0294ecFE76B1e4D6DB04de209B0")
	routerContract := common.HexToAddress("0x6b9bb36519538e0c073894e964e90172e1c0b41f")
	vultToken := common.HexToAddress("")
	wethToken := common.HexToAddress("")
	fee := big.NewInt(3000)
	amountETHIn := big.NewInt(0)
	amountETHIn.SetString("10000000000", 10)

	amoutOut := big.NewInt(0)
	amoutOut.SetString("10000000000", 10)

	rotateEndoints := []string{
		"https://base.drpc.org",
		"https://base.api.onfinality.io/public",
		"https://base-mainnet.public.blastapi.io",
	}

	if len(rotateEndoints) == 0 {
		log.Panic("can u fucking config endpoint pls")
	}

	currentEndPointIndex := 0
	clientInst, err := ethclient.Dial(rotateEndoints[currentEndPointIndex])
	if err != nil {
		log.Panic("you asshole, config death endpoint ", err)
	}

	chainId, err := clientInst.NetworkID(context.Background())
	if err != nil {
		log.Panic(err.Error())
	}

	privateKey, err := crypto.HexToECDSA(privateKeyStr)
	if err != nil {
		log.Panic(err.Error())
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		log.Panic(err.Error())
	}

	// get wewe balance
	weweContract, err := abis.NewWeweabi(weweToken, clientInst)
	if err != nil {
		log.Panic(err.Error())
	}

	weweBal, err := weweContract.BalanceOf(nil, auth.From)
	if err != nil {
		log.Panic(err.Error())
	}
	fmt.Printf("wewe balance: %v \n", weweBal.String())
	fmt.Println("dcm 1")
	// timer
	channel1 := make(chan string)
	pushInChannel := func() {
		channel1 <- "loop"
	}
	// go ListenMempool("")
	// contract router
	uniV3Router, err := abis.NewRouter(routerContract, clientInst)
	if err != nil {
		log.Panic(err.Error())
	}

	//

	// single swap
	singleSwapData := abis.ISwapRouterExactInputSingleParams{
		TokenIn:           wethToken,
		TokenOut:          vultToken,
		Fee:               fee,
		Recipient:         auth.From,
		Deadline:          big.NewInt(100000000),
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
		//tx, err := weweContract.ApproveAndCall(auth, mergeContractAddr, weweBal, []byte{})
		tx, err := uniV3Router.Multicall(auth, [][]byte{swapSingleInputData, refundETHData})
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

					weweContract, err = abis.NewWeweabi(weweToken, clientInst)
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
		time.Sleep(1 * time.Second)
		go pushInChannel()
	}
}

func ListenMempool(wslistener string) {
	fmt.Println("dcm why not show anything")

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wslistener, nil)
	if err != nil {
		log.Fatalf("failed to connect to WebSocket: %v", err)
	}
	defer func(conn *websocket.Conn) {
		err := conn.Close()
		if err != nil {
			log.Fatalf("failed to disconnect to WebSocket: %v", err)
		}
	}(conn)

	payload := EthSubscribePayload{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  "eth_subscribe",
		Params:  []interface{}{"alchemy_pendingTransactions"},
	}

	// Convert the payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}
	fmt.Println(string(jsonPayload))
	if err := conn.WriteJSON(payload); err != nil {
		log.Fatalf("failed to subscribe to newPendingTransactions: %v", err)
	}
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Fatalf("error on message receipt: %v", err)
		}
		fmt.Println("message", string(message))

		// res := &EthSubscribeTxResult{}
		// err = json.Unmarshal(message, res)
		// if err != nil {
		// 	log.Fatalf("Error unmarshaling JSON: %v\n", err)
		// }
	}
}
