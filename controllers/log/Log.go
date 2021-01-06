package log

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"eth.url4g.com/config"
	adminController "eth.url4g.com/controllers/admin"
	orderController "eth.url4g.com/controllers/order"
	"eth.url4g.com/models"
	"eth.url4g.com/myutils"
	"eth.url4g.com/token"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// logTransfer ..//
type logTransfer struct {
	From   common.Address
	To     common.Address
	Tokens *big.Int
}

func init() {
	log.Println(orderController.GetOrderMap())
}

//UsdtEventRoutine 订阅事件
func UsdtEventRoutine() {
	db := models.GetDb()
	defer func() {
		db.Close()
	}()
	config := config.GetConfig()
	client, err := ethclient.Dial(config.GethURL)
	if err != nil {
		log.Fatal(err)
	}

	contractAddress := common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7")
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
	}
	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		log.Println("订阅 连接错误:")
		log.Fatal(err)
	}
	contractAbi, _ := abi.JSON(strings.NewReader(string(token.TokenABI)))
	logTransferSig := []byte("Transfer(address,address,uint256)")
	logTransferSigHash := crypto.Keccak256Hash(logTransferSig)
	log.Println("开始log服务")
	for {
		select {
		case err := <-sub.Err():
			log.Println("订阅错误了")
			log.Fatal(err)
		case vLog := <-logs:
			//fmt.Println(vLog.BlockNumber) // pointer to event log
			//log.Println("BlockNumber:", vLog.BlockNumber, "     TxHash:", vLog.TxHash.Hex())
			if vLog.Topics[0].Hex() == logTransferSigHash.Hex() {
				var transferEvent logTransfer
				contractAbi.Unpack(&transferEvent, "Transfer", vLog.Data)
				transferEvent.From = common.HexToAddress(vLog.Topics[1].Hex())
				//transferEvent.From = common.HexToAddress("0x57EA3A2D605d028b81Ce6aDaC145B71e744eC9fb")
				transferEvent.To = common.HexToAddress(vLog.Topics[2].Hex())
				loadV, ok := orderController.GetOrderMap().Load(strings.ToUpper(transferEvent.To.Hex()))
				if ok {
					// 记录到转账日志中
					outOrderInfo := loadV.(orderController.OutOrderInfo)
					//log.Println("订单id: ", outOrderInfo.ID)
					// 检查转账金额是否和订单金额一致
					resultAction := make(map[string]interface{})
					resultAction["action"] = "TXAction"
					contentsAction := make(map[string]interface{})
					contentsAction["blockNumber"] = strconv.FormatUint(vLog.BlockNumber, 10)
					contentsAction["txHash"] = vLog.TxHash.Hex()
					contentsAction["tokens"] = transferEvent.Tokens.String()
					resultAction["contents"] = contentsAction
					jsonBytes, _ := json.Marshal(resultAction)
					myutils.SafeWriteStringChannel(outOrderInfo.ReadChan, string(jsonBytes))
				}
				//log.Println(transferEvent.From.Hex())

				loadV2, ok2 := adminController.GetOrderMap().Load(strings.ToUpper(transferEvent.From.Hex()))
				if ok2 {
					//log.Println("收到了......")
					outOrderInfo := loadV2.(adminController.OutOrderInfo)
					resultAction := make(map[string]interface{})
					resultAction["action"] = "TXAction"
					contentsAction := make(map[string]interface{})
					contentsAction["blockNumber"] = strconv.FormatUint(vLog.BlockNumber, 10)
					contentsAction["txHash"] = vLog.TxHash.Hex()
					contentsAction["tokens"] = transferEvent.Tokens.String()
					resultAction["contents"] = contentsAction
					jsonBytes, _ := json.Marshal(resultAction)
					myutils.SafeWriteStringChannel(outOrderInfo.ReadChan, string(jsonBytes))
				}

			}
		}
	}
}

//UsdtLogRoutine 记录usdt log
func UsdtLogRoutine() {
	db := models.GetDb()
	defer func() {
		db.Close()
	}()
	config := config.GetConfig()
	for {
		time.Sleep(3 * time.Second)
		//log.Println(config.GethURL)
		myutils.Println(config.GethURL)
		client, err := ethclient.Dial(config.GethURL)

		if err == nil {
			startBlock := models.GetSetting(db, "StartBlock")
			//myutils.Println(startBlock)
			var startBlockInt int64
			// 获取最新的区块
			header, _ := client.HeaderByNumber(context.Background(), nil)
			//myutils.Println("最新区块:", header.Number.String())
			if startBlock == "0" {
				startBlockInt = header.Number.Int64() - 1
			} else {
				tmpInt, err := strconv.ParseInt(startBlock, 10, 64)
				if err != nil {
					startBlockInt = header.Number.Int64() - 1
				} else {
					startBlockInt = tmpInt
				}
			}
			endBlockInt := header.Number.Int64()
			db.Model(&models.Setting{}).Where("skey = ?", "StartBlock").Update(map[string]interface{}{"svalue": header.Number.String(), "update_at": time.Now()})

			myutils.Println("start:", startBlockInt+1, "   end:", endBlockInt)

			contractAddress := common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7")
			if startBlockInt+1 <= endBlockInt {
				query := ethereum.FilterQuery{
					FromBlock: big.NewInt(startBlockInt + 1),
					ToBlock:   big.NewInt(endBlockInt),
					Addresses: []common.Address{
						contractAddress,
					},
				}
				logs, _ := client.FilterLogs(context.Background(), query)
				contractAbi, _ := abi.JSON(strings.NewReader(string(token.TokenABI)))
				logTransferSig := []byte("Transfer(address,address,uint256)")

				logTransferSigHash := crypto.Keccak256Hash(logTransferSig)
				for _, vLog := range logs {
					fmt.Printf("Log Block Number: %d\n", vLog.BlockNumber)
					fmt.Printf("Log Index: %d\n", vLog.Index)
					fmt.Println(vLog.TxHash.Hex())
					block, _ := client.BlockByNumber(context.Background(), big.NewInt(int64(vLog.BlockNumber)))
					fmt.Println("block time:", block.Time())
					//receipt, _ := client.TransactionReceipt(context.Background(), vLog.TxHash)
					//fmt.Println("receipt status:", receipt.Status)
					switch vLog.Topics[0].Hex() {
					case logTransferSigHash.Hex():
						fmt.Printf("Log Name: Transfer\n")
						var transferEvent logTransfer
						contractAbi.Unpack(&transferEvent, "Transfer", vLog.Data)
						transferEvent.From = common.HexToAddress(vLog.Topics[1].Hex())
						transferEvent.To = common.HexToAddress(vLog.Topics[2].Hex())
						fmt.Printf("From: %s\n", transferEvent.From.Hex())
						fmt.Printf("To: %s\n", transferEvent.To.Hex())
						fmt.Printf("Tokens: %s\n", transferEvent.Tokens.String())
						fmt.Printf("------------------\n")
					}
				}
			}
			client.Close()
		} else {
			log.Println(err)
		}
	}
}
