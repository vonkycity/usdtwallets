package order

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"eth.url4g.com/config"
	"eth.url4g.com/myutils"
	"eth.url4g.com/token"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/viper"

	"eth.url4g.com/models"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

//OutOrderInfo 订单信息
type OutOrderInfo struct {
	ID       uint //订单id
	ReadChan chan string
	StopChan chan bool
}

var orderMap *sync.Map
var orderMapOnce sync.Once

func init() {

}

//GetOrderMap 返回进行中的订单
func GetOrderMap() *sync.Map {
	orderMapOnce.Do(func() {
		orderMap = new(sync.Map)
		//go orderDeamon()
	})
	return orderMap
}

//GetOrderCount 返回正在进行的订单数量
func GetOrderCount() int {
	count := 0
	GetOrderMap().Range(
		func(k, v interface{}) bool {
			log.Println(k)
			log.Println(v)
			count++
			return true
		})
	return count
}

//NResult select sum
type NResult struct {
	//N int64 //or int ,or some else
	N float64
}

//GetOrderSystemStatus 返回钱包数量
func GetOrderSystemStatus() map[string]interface{} {
	rtOrderCount := GetOrderCount()
	log.Println(rtOrderCount)
	db := models.GetDb()
	defer func() {
		db.Close()
	}()
	//db.Model(&User{}).Where("name = ?", "jinzhu").Count(&count)
	memberCount := 0 //用户数量
	db.Model(&models.Member{}).Where("type = ?", 1).Count(&memberCount)
	walletCount := 0 //钱包数量
	db.Model(&models.MemberWallet{}).Where("status = ? or status = ? or status = ? or status = ?", 0, 1, -1, -2).Count(&walletCount)
	var n NResult
	db.Table(db.NewScope(models.OutOrder{}).TableName()).Where("status = ?", 1).Select("sum(wei / 1000000) as n").Scan(&n)
	totalPrice := n.N //成功并且回调的订单金额
	log.Println("totalPrice ", totalPrice)
	orderCount := 0 //总订单数量
	db.Model(&models.OutOrder{}).Count(&orderCount)
	successOrderCount := 0 //成功的订单数
	db.Model(&models.OutOrder{}).Where("status = ?", 1).Count(&successOrderCount)

	contentsAction := make(map[string]interface{})
	contentsAction["rtOrderCount"] = rtOrderCount
	contentsAction["memberCount"] = memberCount
	contentsAction["walletCount"] = walletCount
	contentsAction["successPrice"] = strconv.FormatFloat(totalPrice, 'f', 2, 64) + " USDT"
	contentsAction["orderCount"] = orderCount
	contentsAction["successOrderCount"] = successOrderCount
	log.Println(contentsAction)
	return contentsAction

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

//WebOrderSystemStatus 返回订单系统状态
func WebOrderSystemStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":     true,
		"orderCount": GetOrderSystemStatus(),
	})
}

//WebOrderStatus //查看订单状态
func WebOrderStatus(c *gin.Context) {
	//func GetOrderStatus(outOrderID string, t int64, appid string, sign string) (bool, string)
	outOrderID := getWebParam(c, "sdorderno")
	appid := getWebParam(c, "appid")

	tStr := getWebParam(c, "t")
	t, _ := strconv.ParseInt(tStr, 10, 64)
	sign := getWebParam(c, "sign")
	if t == 0 || appid == "0" {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": "参数不全",
		})
	} else {
		status, objmap := GetOrderStatus(outOrderID, t, appid, sign)

		c.JSON(http.StatusOK, gin.H{
			"status":      status,
			"price":       objmap["price"],
			"payPrice":    objmap["payPrice"],
			"wallet":      objmap["wallet"],
			"orderStatus": objmap["status"],
			"createAt":    objmap["createAt"],
			"updateAt":    objmap["updateAt"],
		})
	}
}

//WebGenOrder 通过http生成订单
func WebGenOrder(c *gin.Context) {
	outOrderID := getWebParam(c, "sdorderno")
	appid := getWebParam(c, "appid")
	gid := getWebParam(c, "gid")
	tStr := getWebParam(c, "t")
	t, _ := strconv.ParseInt(tStr, 10, 64)
	priceStr := getWebParam(c, "price")
	callbackURL := getWebParam(c, "callbackurl")
	sign := getWebParam(c, "sign")
	if gid == "" || t == 0 || appid == "0" || priceStr == "0" || callbackURL == "" {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": "参数不全",
		})
	} else {
		//status, walletAddress, message := GenOrder(outOrderID, gid, t, priceStr, appid, callbackURL, sign)
		status, message, walletAddress := GenOrder(outOrderID, gid, t, priceStr, appid, callbackURL, sign)
		if status {
			c.JSON(http.StatusOK, gin.H{
				"status":  status,
				"id":      message,
				"address": walletAddress,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"status":  status,
				"message": message,
			})
		}
	}
}

//GetOrderStatus 返回订单状态
func GetOrderStatus(outOrderID string, t int64, appid string, sign string) (bool, map[string]interface{}) {
	db := models.GetDb()
	defer func() {
		db.Close()
	}()
	//检查时间
	tt := time.Now().Unix()
	if tt-t > 3600 {
		//return false, "", "时间错误"
	}
	//检查签名
	if !verifyGetStatusSign(db, outOrderID, t, appid, sign) {
		return false, nil
	}
	order := models.OutOrder{}
	if err := db.Where("out_id = ?", outOrderID).First(&order).Error; gorm.IsRecordNotFoundError(err) {
		return false, nil
	}
	contentsAction := make(map[string]interface{})
	contentsAction["price"] = strconv.FormatUint(uint64(order.Wei), 10) + " wei USDT"
	contentsAction["payPrice"] = strconv.FormatUint(uint64(order.PayWei), 10) + " wei USDT"
	contentsAction["wallet"] = order.MemberWalletAddress
	contentsAction["status"] = order.Status
	contentsAction["createAt"] = order.CreateAt
	contentsAction["updateAt"] = order.UpdateAt

	return true, contentsAction
}

//GenOrder 生成订单 返回状态、序列号、钱包地址
func GenOrder(outOrderID string, gid string, t int64, priceStr string, appid string, callbackURL string, sign string) (bool, string, string) {
	db := models.GetDb()
	defer func() {
		db.Close()
	}()

	callbackURL, _ = url.QueryUnescape(callbackURL)
	//检查金额
	price, err := strconv.ParseFloat(priceStr, 64)
	//price, err := strconv.ParseInt(priceStr, 10, 64)
	if err != nil {
		return false, "金额错误", "金额错误"
	}
	wei := int64(price * 1000000) // usdt需要补6个0
	myutils.Println("订单金额: ", wei, " wei")

	//检查时间
	tt := time.Now().Unix()
	if tt-t > 3600 {
		//return false, "", "时间错误"
	}

	//检查签名
	if !verifyOrderSign(db, outOrderID, gid, t, priceStr, appid, callbackURL, sign) {
		return false, "签名错误", "签名错误"
	}

	//检查用户和钱包是否创建
	ok, walletAddress := createMember(db, appid, gid)
	if !ok {
		return false, "", walletAddress
	}
	//生成订单
	order := models.OutOrder{}
	//检查订单号是否重复
	if err := db.Where("out_id = ?", outOrderID).First(&order).Error; gorm.IsRecordNotFoundError(err) {
		order.OutID = outOrderID
		appidUint, err := strconv.ParseUint(appid, 10, 64)
		if err != nil {
			return false, "appid错误", "appid错误"
		}
		order.FourthMemberID = uint(appidUint)
		order.GID = gid
		order.TokenType = "usdt"
		order.Wei = uint(wei)
		order.CallbackURL = callbackURL
		order.MemberWalletAddress = walletAddress
		order.TimeStamp = uint(t)
		order.TxHash = ""
		order.Status = 0
		order.CreateAt = time.Now()
		order.UpdateAt = time.Now()
		db.Create(&order)
	} else {
		return false, "订单号重复", "订单号重复"
	}

	//加入map,生成新的routine监控余额
	readChan := make(chan string)
	stopChan := make(chan bool)
	outOrderInfo := OutOrderInfo{ID: order.ID, ReadChan: readChan, StopChan: stopChan}
	GetOrderMap().Store(strings.ToUpper(walletAddress), outOrderInfo)
	go watchOrderRoutine(walletAddress)
	go timeoutOrderRoutine(walletAddress, order.ID)

	return true, strconv.FormatUint(uint64(order.ID), 10), walletAddress
}

func timeoutOrderRoutine(walletAddress string, orderID uint) {
	//1200秒后超时
	time.Sleep(1200 * time.Second)
	var outOrderInfo OutOrderInfo
	loadV, ok := GetOrderMap().Load(strings.ToUpper(walletAddress))
	if ok {
		outOrderInfo = loadV.(OutOrderInfo)
		if outOrderInfo.ID != orderID {
			// 不是这个routine该处理的超时
			return
		}
		myutils.SafeWriteBoolChannel(outOrderInfo.StopChan, true)
		//查看数据库中订单状态，如果是0，则改为超时状态
		db := models.GetDb()
		defer db.Close()
		outOrder := models.OutOrder{}
		if err := db.Where("id = ?", outOrderInfo.ID).First(&outOrder).Error; gorm.IsRecordNotFoundError(err) {
			myutils.Println("error: ", "找不到订单 ", outOrderInfo.ID)
		} else {
			if outOrder.Status == 0 {
				outOrder.Status = -1
				outOrder.UpdateAt = time.Now()
				db.Save(&outOrder)
				myutils.Println("订单超时，订单id: ", outOrderInfo.ID)
			} else {
				myutils.Println("已过900秒，订单完成，订单id: ", outOrderInfo.ID)
			}
		}
	} else {
		myutils.Println("已经释放过此订单，钱包地址: ", walletAddress)
	}
}

func watchOrderRoutine(walletAddress string) {
	//startTime := time.Now()
	var outOrderInfo OutOrderInfo
	loadV, ok := GetOrderMap().Load(strings.ToUpper(walletAddress))
	if ok {
		outOrderInfo = loadV.(OutOrderInfo)
		//readChan = outOrderInfo.ReadChan
	LOOP:
		for {
			select {
			case readStr, opened := <-outOrderInfo.ReadChan:
				if !opened {
					break LOOP
				}
				//log.Println("收到消息:")
				//log.Println(readStr)
				go handlMessage(walletAddress, readStr)
			case stop := <-outOrderInfo.StopChan:
				if stop {
					break LOOP
				}
			}
		}
		//从map中删除
		GetOrderMap().Delete(strings.ToUpper(walletAddress))
	}
}

//更新钱包余额
func updateWalletBalance(walletAddress string) {
	db := models.GetDb()
	defer func() {
		db.Close()
	}()
	config := config.GetConfig()
	client, err := ethclient.Dial(config.GethURL)
	if err != nil {
		//log.Fatal(err)
		return
	}
	contractAddress := common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7")
	instance, err := token.NewToken(contractAddress, client)
	if err != nil {
		//log.Fatal(err)
		return
	}
	address := common.HexToAddress(walletAddress)
	bal, err := instance.BalanceOf(&bind.CallOpts{}, address)
	if err != nil {
		//log.Fatal(err)
		return
	}
	//log.Println("balance usdt wei: ", bal)
	bal2, err := client.BalanceAt(context.Background(), address, nil)
	//log.Println("balance eth wei: ", bal2)
	db.Model(&models.MemberWallet{}).Where("address = ?", walletAddress).Update(map[string]interface{}{"usdt_wei": bal.String(), "eth_wei": bal2.String(), "update_at": time.Now()})
	//db.Model(&models.MemberWallet{}).Where("address = ?", strings.ToLower(walletAddress)).Update(map[string]interface{}{"usdt_wei": bal.String(), "eth_wei": bal2.String(), "update_at": time.Now()})
}

func handlMessage(walletAddress string, message string) {
	//log.Println(message, "-----------")
	// 如果收到成功转账消息，那么获取钱包余额，更新数据库
	vp := viper.New()
	vp.SetConfigType("json")
	vp.ReadConfig(strings.NewReader(message))
	action := vp.GetString("action")
	if action == "TXAction" {
		db := models.GetDb()
		defer db.Close()
		//获取订单id，更新订单状态
		var outOrderInfo OutOrderInfo
		loadV, ok := GetOrderMap().Load(strings.ToUpper(walletAddress))
		if ok {
			outOrderInfo = loadV.(OutOrderInfo)
			order := models.OutOrder{}
			if err := db.Where("id = ?", outOrderInfo.ID).First(&order).Error; gorm.IsRecordNotFoundError(err) {
				// 错误 没有这个订单
			} else {
				// 更新数据库
				order.PayWei = vp.GetUint("contents.tokens")
				order.TxHash = vp.GetString("contents.txHash")
				order.Status = 1
				order.UpdateAt = time.Now()
				db.Save(&order)
				//更新余额
				updateWalletBalance(walletAddress)
				//回调(先获取appkey)
				member := models.Member{}
				if err := db.Where("id = ?", order.FourthMemberID).First(&member).Error; gorm.IsRecordNotFoundError(err) {
					log.Println("无法进行回调，没有找到四方用户，用户id:", order.FourthMemberID, " 订单id:", order.ID)
				} else {
					appKey := member.AppKey
					//config := config.GetConfig()
					notifyAction := make(map[string]interface{})
					notifyAction["queryType"] = "paymentNotify"
					notifyAction["queryUrl"] = order.CallbackURL
					notifyAction["queryFormat"] = "POST"
					computStr := strconv.FormatInt(int64(order.ID), 10) + strconv.FormatInt(int64(order.PayWei/1000000), 10) + strconv.FormatInt(int64(order.Wei/1000000), 10) + order.OutID + strconv.FormatInt(int64(order.Status), 10) + appKey
					sign := myutils.GetMd5(computStr)
					notifyAction["queryData"] = "orderid=" + strconv.FormatInt(int64(order.ID), 10) + "&payprice=" + strconv.FormatInt(int64(order.PayWei/1000000), 10) + "&price=" + strconv.FormatInt(int64(order.Wei/1000000), 10) + "&sdorderno=" + order.OutID + "&status=" + strconv.FormatInt(int64(order.Status), 10) + "&sign=" + sign
					//notifyAction里包含订单状态，请自行写这部分代码
					log.Println("订单成功，这里请写上你自己的回调方法")
				}
			}
			myutils.SafeWriteBoolChannel(outOrderInfo.StopChan, true)
		}
	}
}

//获取或创建用户 返回用户钱包地址
func createMember(db *gorm.DB, appid string, gid string) (bool, string) {
	//检查用户是否存在 检查用户是否和appid匹配
	if db == nil {
		db = models.GetDb()
		defer db.Close()
	}
	member := models.Member{}
	//Related(&topic.Category, "CategoryId")
	//if err := db.Where("g_id = ?", gid).Preload("Parent").First(&member).Error; gorm.IsRecordNotFoundError(err) {
	if err := db.Preload("Wallets").Where("g_id = ?", gid).First(&member).Error; gorm.IsRecordNotFoundError(err) {
		// 创建钱包
		ok, address, file := createWallet()
		//myutils.Println(ok, address, file)
		if !ok {
			return false, "创建钱包失败"
		}
		// 创建用户
		parentMember := models.Member{}
		if err := db.Where("id = ?", appid).First(&parentMember).Error; gorm.IsRecordNotFoundError(err) {
			return false, "没有此appid"
		}
		if parentMember.Type != 2 {
			return false, "没有此appid"
		}
		//return false, ""
		member.GID = gid
		member.ParentID = sql.NullInt64{Int64: int64(parentMember.ID), Valid: true}
		member.Mobile = myutils.GenUUID()
		member.Type = 1
		member.Status = 1
		member.Wallets = []models.MemberWallet{
			{
				Address:  address,
				File:     file,
				UsdtWei:  "0",
				EthWei:   "0",
				Status:   0,
				CreateAt: time.Now(),
				UpdateAt: time.Now(),
			},
		}
		member.AccessToken = myutils.GetMd5(myutils.GenUUID())
		member.CreateAt = time.Now()
		member.UpdateAt = time.Now()
		db.Create(&member)
		//db.Save(&member)
		if member.ID == 0 {
			return false, "创建用户失败"
		}
	}
	if strconv.FormatUint(uint64(member.ParentID.Int64), 10) != appid {
		return false, "appid不匹配"
	}
	//选取空闲钱包
	//如果没有空闲钱包，则创建钱包
	//如果钱包数量到达最大限额，则返回失败
	//log.Println("wallets: ", member.Wallets)
	walletAddress := ""
	walletNum := 0
	for _, wallet := range member.Wallets {
		myutils.Println("add: ", wallet.Address, "        status: ", wallet.Status)
		walletNum++
		if wallet.Status == 0 {
			walletAddress = wallet.Address
			wallet.Status = 1
			db.Save(&wallet)
			//log.Println("-------选取空闲钱包-------------")
			break
		}
		log.Println(wallet.Status)
	}
	//return false, ""
	memberWalletLimit, err := strconv.Atoi(models.GetSetting(db, "MemberWalletLimit"))
	if err != nil {
		return false, "未知错误"
	}
	if walletNum < memberWalletLimit && walletAddress == "" {
		// 创建一个新钱包
		ok, address, file := createWallet()
		if !ok {
			return false, "创建钱包失败"
		}
		wallet := models.MemberWallet{
			MemberID: member.ID,
			//Member:   member,
			Address:  address,
			File:     file,
			UsdtWei:  "0",
			EthWei:   "0",
			Status:   1,
			CreateAt: time.Now(),
			UpdateAt: time.Now(),
		}
		db.Create(&wallet)
		if wallet.ID == 0 {
			return false, "创建钱包失败"
		}
		walletAddress = address
	}
	myutils.Println("可用钱包: ", walletAddress)
	if walletAddress == "" {
		myutils.Println("无可用钱包")
		return false, "没有可用钱包"
	}
	return true, walletAddress
}

//创建钱包 返回 成功，钱包地址，ks文件地址
func createWallet() (bool, string, string) {
	now := time.Now()
	y := now.Format("2006")
	m := now.Format("01")
	log.Println(myutils.GetCurrentPath() + "keystore/" + y + m)
	ks := keystore.NewKeyStore(myutils.GetCurrentPath()+"/keystore/"+y+m, keystore.StandardScryptN, keystore.StandardScryptP)
	password := config.GetConfig().Sec.Kspassword
	account, err := ks.NewAccount(password)
	if err != nil {
		return false, "", ""
	}
	//myutils.Println(account)
	_, fileName := filepath.Split(account.URL.Path)
	filePath := "keystore/" + y + m + "/" + fileName
	return true, account.Address.Hex(), filePath
}

func verifyOrderSign(db *gorm.DB, outOrderID string, gid string, t int64, priceStr string, appid string, callbackURL string, sign string) bool {
	//转换用户id
	//memberID := myutils.AnyToDec(appid, 36)
	member := models.Member{}
	if err := db.Where("id = ?", appid).First(&member).Error; gorm.IsRecordNotFoundError(err) {
		return false
	}
	if member.Type != 2 {
		return false
	}
	appKey := member.AppKey
	md5Str := ""
	//md5Str = myutils.GetMd5(appid + callbackURL + gid + priceStr + outOrderID + strconv.FormatInt(t, 10) + appKey)
	md5Str = myutils.GetMd5(appid + gid + priceStr + outOrderID + strconv.FormatInt(t, 10) + appKey)
	if md5Str == sign {
		return true
	}
	myutils.Println("签名应该是: ", md5Str)
	return false
}

func verifyGetStatusSign(db *gorm.DB, outOrderID string, t int64, appid string, sign string) bool {
	member := models.Member{}
	if err := db.Where("id = ?", appid).First(&member).Error; gorm.IsRecordNotFoundError(err) {
		return false
	}
	if member.Type != 2 {
		return false
	}
	//检查订单是否属于此appid
	order := models.OutOrder{}
	if err := db.Where("out_id = ?", outOrderID).First(&order).Error; gorm.IsRecordNotFoundError(err) {
		return false
	}
	//log.Println(order.FourthMemberID, member.ID)
	if order.FourthMemberID != member.ID {
		return false
	}
	//log.Println(order.FourthMemberID, member.ID)
	appKey := member.AppKey
	md5Str := ""
	md5Str = myutils.GetMd5(appid + outOrderID + strconv.FormatInt(t, 10) + appKey)
	if md5Str == sign {
		return true
	}
	myutils.Println("签名应该是: ", md5Str)
	return false
}
