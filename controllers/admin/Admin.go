package admin

import (
	"context"
	"crypto/ecdsa"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"eth.url4g.com/config"
	orderController "eth.url4g.com/controllers/order"
	"eth.url4g.com/models"
	"eth.url4g.com/myutils"
	"eth.url4g.com/token"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
	"golang.org/x/crypto/sha3"
)

//OutOrderInfo 订单信息
type OutOrderInfo struct {
	Wallet   string //钱包地址
	ReadChan chan string
	StopChan chan bool
}

var orderMap *sync.Map
var orderMapOnce sync.Once

func init() {
	log.Println("admin init")
	//log.Println(myutils.SafeWriteBoolChannel(nil, true))
	//collectWallet("0x57EA3A2D605d028b81Ce6aDaC145B71e744eC9fb")
}

//GetOrderMap 返回进行中的订单
func GetOrderMap() *sync.Map {
	orderMapOnce.Do(func() {
		orderMap = new(sync.Map)
	})
	return orderMap
}

func getWebParam(c *gin.Context, paramName string) string {
	paramValue := c.DefaultQuery(paramName, "")
	if paramValue == "" {
		paramValue = c.Param(paramName)
	}
	if paramValue == "" {
		paramValue = c.DefaultPostForm(paramName, "")
	}
	return paramValue
}

//MemberWalletRow 钱包列表结果集
type MemberWalletRow struct {
	ID       uint      `gorm:"AUTO_INCREMENT;PRIMARY_KEY;"`
	MemberID uint      `gorm:"INDEX;"`                          // 用户ID
	Address  string    `gorm:"TYPE:VARCHAR(150);UNIQUE_INDEX;"` // 钱包地址
	File     string    `gorm:"TYPE:VARCHAR(255);"`              // ks文件名
	UsdtWei  string    `gorm:"TYPE:VARCHAR(128);"`              // usdt余额
	EthWei   string    `gorm:"TYPE:VARCHAR(128);"`              // 以太币余额
	Status   int       `gorm:"TYPE:INT;DEFAULT:0;INDEX;"`       // 状态 -1=异常 0=空闲 1=繁忙 2=繁忙 在规集
	CreateAt time.Time `gorm:"INDEX;"`
	UpdateAt time.Time `gorm:"INDEX;"`
}

//GetWalletList 返回钱包列表
func GetWalletList(page int64) map[string]interface{} {
	time.Sleep(3 * time.Second)
	db := models.GetDb()
	defer func() {
		db.Close()
	}()
	count := 0
	db.Model(&models.MemberWallet{}).Count(&count)
	if page == 0 {
		page = 1
	}
	pageSize := int64(100)
	totalPage := math.Ceil(float64(count) / float64(pageSize))
	if page > int64(totalPage) {
		page = int64(totalPage)
	}
	walletList := []MemberWalletRow{}
	offset := (page - 1) * pageSize
	db.Table(db.NewScope(models.MemberWallet{}).TableName()).Offset(offset).Limit(pageSize).Order("usdt_wei desc").Scan(&walletList)
	//db.Offset(offset).Limit(pageSize).Order("usdt_wei desc").Find(&walletList)
	//log.Println(walletList)
	contentsAction := make(map[string]interface{})
	contentsAction["totalPage"] = totalPage
	contentsAction["list"] = walletList
	//jsonBytes, _ := json.Marshal(contentsAction)
	//log.Println(string(jsonBytes))
	return contentsAction
}

//WebGetWalletList 获取钱包列表
func WebGetWalletList(c *gin.Context) {
	db := models.GetDb()
	defer db.Close()
	accesstoken := getWebParam(c, "accesstoken")
	pageStr := getWebParam(c, "page")
	page, _ := strconv.ParseInt(pageStr, 10, 64)
	member := models.Member{}
	if err := db.Where("access_token = ? and type = ?", accesstoken, 99).First(&member).Error; gorm.IsRecordNotFoundError(err) {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":   true,
		"contents": GetWalletList(page),
	})
}

//WebSystemStatus web
func WebSystemStatus(c *gin.Context) {
	db := models.GetDb()
	defer db.Close()
	accesstoken := getWebParam(c, "accesstoken")
	member := models.Member{}
	if err := db.Where("access_token = ? and type = ?", accesstoken, 99).First(&member).Error; gorm.IsRecordNotFoundError(err) {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
		})
		return
	}
	orderController.WebOrderSystemStatus(c)
}

//WebCollectMoney web
func WebCollectMoney(c *gin.Context) {
	db := models.GetDb()
	defer db.Close()
	wallet := getWebParam(c, "wallet")
	accesstoken := getWebParam(c, "accesstoken")
	member := models.Member{}
	if err := db.Where("access_token = ? and type = ?", accesstoken, 99).First(&member).Error; gorm.IsRecordNotFoundError(err) {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
		})
		return
	}
	collectWallet(wallet)
	c.JSON(http.StatusOK, gin.H{
		"status": true,
	})
}

//归集
func collectWallet(wallet string) {
	client, err := ethclient.Dial(config.GetConfig().GethURL)
	if err != nil {
		return
	}
	defer client.Close()
	db := models.GetDb()
	defer db.Close()
	//检查钱包状态，如果繁忙直接停止
	w := models.MemberWallet{}
	if err := db.Where("address = ?", wallet).First(&w).Error; gorm.IsRecordNotFoundError(err) {
		log.Println("没有钱包地址")
		return
	}
	if w.Status != 0 {
		log.Println("钱包繁忙")
		return
	}
	w.Status = 2
	w.UpdateAt = time.Now()
	db.Save(&w)

	account := common.HexToAddress(wallet)
	gasPrice := getGasPrice(client)
	if gasPrice != nil {
		log.Println("gas price:", gasPrice.String())
		usdtGasLimit := 60000
		fee := gasPrice.Uint64() * uint64(usdtGasLimit)
		log.Println("手续费需要:", fee)
		ethBalance, _ := client.BalanceAt(context.Background(), account, nil)
		log.Println("当前余额:", ethBalance)
		tmpPrice := uint64(0)
		if ethBalance.Uint64() < fee {
			//需要额外转账
			tmpPrice = fee - ethBalance.Uint64() + 100000
		}
		txPrice := new(big.Int)
		//test
		//tmpPrice = uint64(100000)
		txPrice.SetUint64(tmpPrice)
		log.Println("需要额外转账:", txPrice, " wei")
		readChan := make(chan string)
		stopChan := make(chan bool)
		outOrderInfo := OutOrderInfo{Wallet: wallet, ReadChan: readChan, StopChan: stopChan}

		if transEth(client, wallet, txPrice, gasPrice) {
			GetOrderMap().Store(strings.ToUpper(wallet), outOrderInfo)
			go transUsdtRoutine(wallet, fee, gasPrice)
			go watchOrderRoutine(wallet)
			//log.Println("map:", GetOrderMap())
		}
	}
}

//更新钱包余额
func updateWalletBalance(wallet string) {
	db := models.GetDb()
	defer func() {
		db.Close()
	}()
	config := config.GetConfig()
	client, err := ethclient.Dial(config.GethURL)
	if err != nil {
		return
	}
	contractAddress := common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7")
	instance, err := token.NewToken(contractAddress, client)
	if err != nil {
		return
	}
	address := common.HexToAddress(wallet)
	bal, err := instance.BalanceOf(&bind.CallOpts{}, address)
	if err != nil {
		return
	}
	//log.Println("balance usdt wei: ", bal)
	bal2, err := client.BalanceAt(context.Background(), address, nil)
	//log.Println("balance eth wei: ", bal2)
	db.Model(&models.MemberWallet{}).Where("address = ?", wallet).Update(map[string]interface{}{"usdt_wei": bal.String(), "eth_wei": bal2.String(), "status": 0, "update_at": time.Now()})
	//db.Model(&models.MemberWallet{}).Where("address = ?", strings.ToLower(wallet)).Update(map[string]interface{}{"usdt_wei": bal.String(), "eth_wei": bal2.String(), "status": 0, "update_at": time.Now()})
}

func handlMessage(wallet string, message string) {
	vp := viper.New()
	vp.SetConfigType("json")
	vp.ReadConfig(strings.NewReader(message))
	action := vp.GetString("action")
	if action == "TXAction" {
		db := models.GetDb()
		defer db.Close()
		var outOrderInfo OutOrderInfo
		loadV, ok := GetOrderMap().Load(strings.ToUpper(wallet))
		if ok {
			outOrderInfo = loadV.(OutOrderInfo)
			// 更新钱包余额
			updateWalletBalance(outOrderInfo.Wallet)
			myutils.SafeWriteBoolChannel(outOrderInfo.StopChan, true)
		}
	}
}

func watchOrderRoutine(wallet string) {
	var outOrderInfo OutOrderInfo
	loadV, ok := GetOrderMap().Load(strings.ToUpper(wallet))
	//log.Println("ok? ", ok)
	if ok {
		outOrderInfo = loadV.(OutOrderInfo)
	LOOP:
		for {
			select {
			case readStr, opened := <-outOrderInfo.ReadChan:
				if !opened {
					break LOOP
				}
				//log.Println("收到消息:")
				//log.Println(readStr)
				go handlMessage(wallet, readStr)
			case stop := <-outOrderInfo.StopChan:
				if stop {
					break LOOP
				}
			}
		}

		log.Println("删除map obj")
		GetOrderMap().Delete(strings.ToUpper(wallet))
	}
}

//转账usdt
func transUsdtRoutine(wallet string, fee uint64, gasPrice *big.Int) {
	db := models.GetDb()
	defer db.Close()
	w := models.MemberWallet{}
	if err := db.Where("address = ?", wallet).First(&w).Error; gorm.IsRecordNotFoundError(err) {
		return
	}
	password := config.GetConfig().Sec.Kspassword
	ksFile := myutils.GetCurrentPath() + w.File
	keyjson, err := ioutil.ReadFile(ksFile)
	key, err := keystore.DecryptKey(keyjson, password)
	if err != nil {
		log.Fatal(err)
	}
	privateKey := key.PrivateKey
	maxCount := 120
	index := 0
LOOP:
	for {
		time.Sleep(9 * time.Second)
		if index >= maxCount {
			log.Println("归集超时了")
			break LOOP
		}
		index++
		//检查以太币余额是否大于等于fee
		//如果大于等于，那么执行把所有的usdt转到指定地址 break LOOP
		client, err := ethclient.Dial(config.GetConfig().GethURL)
		if err != nil {
			//log.Println("---1")
			break LOOP
		}
		contractAddress := common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7")
		instance, err := token.NewToken(contractAddress, client)
		if err != nil {
			//log.Println("---2")
			client.Close()
			break LOOP
		}
		address := common.HexToAddress(wallet)
		bal, err := instance.BalanceOf(&bind.CallOpts{}, address)
		if err != nil {
			//log.Println("---3")
			client.Close()
			break LOOP
		}
		if bal.Int64() == int64(0) {
			//log.Println("余额为0，不进行规集")
			myutils.Println("余额为0，不进行规集")
			client.Close()
			break LOOP
		}
		//log.Println("balance usdt wei: ", bal)
		bal2, err := client.BalanceAt(context.Background(), address, nil)
		if bal2.Uint64() >= fee {
			//转账
			//log.Println(bal)
			fromAddress := common.HexToAddress(wallet)
			nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
			if err != nil {
				//log.Println("---4")
				client.Close()
				break LOOP
			}
			value := big.NewInt(0)
			toAddress := common.HexToAddress(config.GetConfig().Sec.Wallet)
			transferFnSignature := []byte("transfer(address,uint256)")
			hash := sha3.NewLegacyKeccak256()
			hash.Write(transferFnSignature)
			methodID := hash.Sum(nil)[:4]

			paddedAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
			paddedAmount := common.LeftPadBytes(bal.Bytes(), 32)
			var data []byte
			data = append(data, methodID...)
			data = append(data, paddedAddress...)
			data = append(data, paddedAmount...)
			gasLimit := uint64(60000)
			tx := types.NewTransaction(nonce, contractAddress, value, gasLimit, gasPrice, data)
			chainID, err := client.NetworkID(context.Background())
			if err != nil {
				//log.Println("---5")
				client.Close()
				break LOOP
			}

			signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
			if err != nil {
				//log.Println("---6")
				client.Close()
				break LOOP
			}

			//log.Println(tx, chainID)
			err = client.SendTransaction(context.Background(), signedTx)
			if err != nil {
				//log.Println("---7")
				client.Close()
				break LOOP
			}
			//fmt.Printf("规集 usdt tx sent: %s \n", signedTx.Hash().Hex())
			myutils.Println("规集 usdt tx sent: ", signedTx.Hash().Hex())
			client.Close()
			break LOOP
		}
		client.Close()
	}
	/*
		loadV, ok := GetOrderMap().Load(strings.ToUpper(wallet))
		if ok {
			outOrderInfo := loadV.(OutOrderInfo)
			myutils.SafeWriteBoolChannel(outOrderInfo.StopChan, true)
		}
	*/
}

//转账eth
func transEth(client *ethclient.Client, wallet string, price *big.Int, gasPrice *big.Int) bool {
	if price.Uint64() != uint64(0) {
		// 进行转账操作
		privateKey, err := crypto.HexToECDSA(config.GetConfig().Sec.Ethkey)
		if err != nil {
			return false
		}
		publicKey := privateKey.Public()
		publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
		if !ok {
			return false
		}

		fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
		nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
		if err != nil {
			return false
		}
		gasLimit := uint64(21000)
		toAddress := common.HexToAddress(wallet)
		var data []byte
		tx := types.NewTransaction(nonce, toAddress, price, gasLimit, gasPrice, data)
		chainID, err := client.NetworkID(context.Background())
		if err != nil {
			return false
		}

		signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
		if err != nil {
			return false
		}
		err = client.SendTransaction(context.Background(), signedTx)
		if err != nil {
			return false
		}
		log.Printf("以太币转账 tx sent: %s", signedTx.Hash().Hex())
		return true
	}
	return true
}

//计算gas
func getGasPrice(client *ethclient.Client) *big.Int {
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil
	}
	return gasPrice
}
