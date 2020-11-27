package models

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"eth.url4g.com/myutils"

	"eth.url4g.com/config"
	_ "github.com/go-sql-driver/mysql" // init mysql
	"github.com/jinzhu/gorm"
)

// Setting 系统设置表
type Setting struct {
	Skey     string    `gorm:"TYPE:VARCHAR(100);PRIMARY_KEY;"`
	Svalue   string    `gorm:"TYPE:TEXT;"`
	CreateAt time.Time `gorm:"INDEX;"`
	UpdateAt time.Time `gorm:"INDEX;"`
}

// Member 用户表
type Member struct {
	ID          uint   `gorm:"AUTO_INCREMENT;PRIMARY_KEY;"`
	GID         string `gorm:"TYPE:VARCHAR(100);UNIQUE_INDEX;"`
	Mobile      string `gorm:"TYPE:VARCHAR(100);UNIQUE_INDEX;"`
	Name        string `gorm:"TYPE:VARCHAR(100);"`
	Password    string `gorm:"TYPE:VARCHAR(128);"`
	Salt        string `gorm:"TYPE:VARCHAR(16);"`
	Type        uint   `gorm:"TYPE:INT;DEFAULT:1;INDEX;"` // 用户类型 1=普通用户(分配钱包，等待充值) 2=四方(四方用户会有appkey) 99=管理员
	AccessToken string `gorm:"TYPE:VARCHAR(128);UNIQUE_INDEX;"`
	AppKey      string `gorm:"TYPE:VARCHAR(128);"`
	//Parent      *Member `gorm:"ForeignKey:ParentID"`
	ParentID sql.NullInt64
	Wallets  []MemberWallet
	Status   int       `gorm:"TYPE:INT;DEFAULT:1;INDEX;"` // 用户状态 1=正常 -1=禁用
	CreateAt time.Time `gorm:"INDEX;"`
	UpdateAt time.Time `gorm:"INDEX;"`
}

// MemberWallet 用户钱包表
type MemberWallet struct {
	ID       uint      `gorm:"AUTO_INCREMENT;PRIMARY_KEY;"`
	Member   Member    `gorm:"FOREIGNKEY:MemberID;"`            // 用户
	MemberID uint      `gorm:"INDEX;"`                          // 用户ID
	Address  string    `gorm:"TYPE:VARCHAR(150);UNIQUE_INDEX;"` // 钱包地址
	File     string    `gorm:"TYPE:VARCHAR(255);"`              // ks文件名
	UsdtWei  string    `gorm:"TYPE:VARCHAR(128);"`              // usdt余额
	EthWei   string    `gorm:"TYPE:VARCHAR(128);"`              // 以太币余额
	Status   int       `gorm:"TYPE:INT;DEFAULT:0;INDEX;"`       // 状态 -1=异常 0=空闲 1=繁忙 2=繁忙 在规集
	CreateAt time.Time `gorm:"INDEX;"`
	UpdateAt time.Time `gorm:"INDEX;"`
}

// WalletLog 钱包转账日志表
type WalletLog struct {
	FromAddress   string    `gorm:"TYPE:VARCHAR(150);INDEX"` // 钱包地址
	ToAddress     string    `gorm:"TYPE:VARCHAR(150);INDEX"` // 钱包地址
	BlockNumber   uint64    `gorm:"TYPE:BIGINT;"`            // 区块号
	BlockTime     time.Time `gorm:"INDEX;"`                  // 区块时间
	TxHash        string    `gorm:"TYPE:VARCHAR(150);INDEX"` // 交易hash
	Tokens        uint64    `gorm:"TYPE:BIGINT;"`            // 交易金额
	ReceiptStatus int       `gorm:"TYPE:INT;INDEX;"`         // 票据状态 1=成功交易
	CreateAt      time.Time `gorm:"INDEX;"`
	UpdateAt      time.Time `gorm:"INDEX;"`
}

//OutOrder 订单表
type OutOrder struct {
	ID                  uint      `gorm:"AUTO_INCREMENT;PRIMARY_KEY;"`
	OutID               string    `gorm:"TYPE:VARCHAR(160);UNIQUE_INDEX:idx_outid_memberid;"` // 外部订单号(四方提供)
	FourthMember        Member    `gorm:"FOREIGNKEY:FourthMemberID;"`                         // 四方用户
	FourthMemberID      uint      `gorm:"INDEX;UNIQUE_INDEX:idx_outid_memberid;"`             // 四方用户ID
	GID                 string    `gorm:"TYPE:VARCHAR(160);"`                                 // 唯一用户id(用来判断是否需要注册用户、以及分配钱包)
	TokenType           string    `gorm:"TYPE:VARCHAR(32);INDEX;"`                            // 货币类型 usdt
	Wei                 uint      `gorm:"TYPE:BIGINT;"`                                       // 金额
	PayWei              uint      `gorm:"TYPE:BIGINT;DEFAULT:0;"`                             // 实际支付金额
	CallbackURL         string    `gorm:"TYPE:VARCHAR(500);"`                                 // 回调地址
	MemberWalletAddress string    `gorm:"INDEX;"`                                             // 钱包地址
	TimeStamp           uint      `gorm:"TYPE:INT;"`                                          // 时间戳
	TxHash              string    `gorm:"INDEX;"`                                             // 交易hash
	Status              int       `gorm:"TYPE:INT;DEFAULT:0;INDEX;"`                          // 订单状态 0=等待支付 1=完成 -1=超时
	CreateAt            time.Time `gorm:"INDEX;"`
	UpdateAt            time.Time `gorm:"INDEX;"`
}

//GetDb 返回数据库实例
func GetDb() *gorm.DB {
	config := config.GetConfig()
	//log.Println(config.Mysql.Connect)
	db, err := gorm.Open("mysql", config.Mysql.Connect)
	if err != nil {
		panic(err)
	}
	gorm.DefaultTableNameHandler = func(db *gorm.DB, defaultTableName string) string {
		return config.Mysql.Tableprefix + defaultTableName
	}

	db.LogMode(myutils.EnableDebug)
	//db.LogMode(true)
	return db
}

func init() {
	log.Println("初始化mysql")
	db := GetDb()
	defer func() {
		db.Close()
	}()

	db.Set("gorm:table_options", "ENGINE=InnoDB AUTO_INCREMENT=30501").AutoMigrate(&Member{})
	db.Set("gorm:table_options", "ENGINE=InnoDB").AutoMigrate(&Setting{}, &MemberWallet{}, &WalletLog{}, &OutOrder{})

	db.Model(&MemberWallet{}).Updates(map[string]interface{}{"status": 0})
	db.Where("status = ?", 0).Model(&OutOrder{}).Updates(map[string]interface{}{"status": -2})

	if err := db.Where("type = ?", 99).First(&Member{}).Error; gorm.IsRecordNotFoundError(err) {
		adminMobile := ""
		adminPassword := ""
		fmt.Println("请输入管理员手机号(用来登录): ")
		fmt.Scanln(&adminMobile)
		fmt.Println("请输入管理员密码: ")
		fmt.Scanln(&adminPassword)
		adminMember := Member{}
		adminMember.GID = myutils.GenUUID()
		adminMember.Mobile = adminMobile
		adminMember.Salt = myutils.GenRandomString(16)
		adminMember.Password = myutils.GetMd5(myutils.GetMd5(adminPassword) + adminMember.Salt)
		adminMember.Type = 99
		adminMember.AccessToken = myutils.GetMd5(myutils.GenUUID())
		adminMember.CreateAt = time.Now()
		adminMember.UpdateAt = time.Now()
		db.Create(&adminMember)
	}
	if err := db.Where("type = ?", 2).First(&Member{}).Error; gorm.IsRecordNotFoundError(err) {
		fmt.Println("是否添加测试数据: ")
		insertTestData := ""
		fmt.Scanln(&insertTestData)
		if insertTestData == "y" || insertTestData == "Y" {
			fourthMember := Member{}
			fourthMember.GID = myutils.GenUUID()
			fourthMember.Mobile = "18630100000"
			fourthMember.Salt = myutils.GenRandomString(16)
			fourthMember.Password = myutils.GetMd5(myutils.GetMd5(fourthMember.Mobile) + fourthMember.Salt)
			fourthMember.Type = 2
			fourthMember.AppKey = myutils.GenRandomString(16)
			fourthMember.AccessToken = myutils.GetMd5(myutils.GenUUID())
			fourthMember.CreateAt = time.Now()
			fourthMember.UpdateAt = time.Now()
			db.Create(&fourthMember)
			fmt.Println("四方测试帐号: " + fourthMember.Mobile)
			fmt.Println("四方测试密码: " + fourthMember.Mobile)
			//fmt.Println("四方测试AppID: " + myutils.DecToAny(int64(fourthMember.ID), 36))
			fmt.Println("四方测试AppID: ", fourthMember.ID)
			fmt.Println("四方测试AppKey: " + fourthMember.AppKey)
		}
	}

	if err := db.Where("skey = ?", "StartBlock").First(&Setting{}).Error; gorm.IsRecordNotFoundError(err) {
		st := Setting{Skey: "StartBlock", Svalue: "0", CreateAt: time.Now(), UpdateAt: time.Now()}
		db.Create(&st)
	}
	if err := db.Where("skey = ?", "UsdtGasLimit").First(&Setting{}).Error; gorm.IsRecordNotFoundError(err) {
		st := Setting{Skey: "UsdtGasLimit", Svalue: "60000", CreateAt: time.Now(), UpdateAt: time.Now()}
		db.Create(&st)
	}
	if err := db.Where("skey = ?", "EthGasLimit").First(&Setting{}).Error; gorm.IsRecordNotFoundError(err) {
		st := Setting{Skey: "EthGasLimit", Svalue: "21000", CreateAt: time.Now(), UpdateAt: time.Now()}
		db.Create(&st)
	}
	if err := db.Where("skey = ?", "MemberWalletLimit").First(&Setting{}).Error; gorm.IsRecordNotFoundError(err) {
		// 每用户最多创建3个钱包账号
		st := Setting{Skey: "MemberWalletLimit", Svalue: "3", CreateAt: time.Now(), UpdateAt: time.Now()}
		db.Create(&st)
	}
	/*
		log.Println(GetSetting(db, "EthGasLimit"))
		log.Println(GetSetting(nil, "EthGasLimit"))
		log.Println(GetSetting(db, "UsdtGasLimit"))
		log.Println(GetSetting(nil, "UsdtGasLimit"))
		log.Println(myutils.EnableDebug)
	*/
}

// GetSetting 返回设置表中的内容
func GetSetting(db *gorm.DB, skey string) string {
	needClose := false
	if db == nil {
		db = GetDb()
		needClose = true
	}
	defer func() {
		if needClose {
			db.Close()
		}
	}()
	st := Setting{}
	if err := db.Where("skey = ?", skey).First(&st).Error; gorm.IsRecordNotFoundError(err) {
		return ""
	}
	return st.Svalue
}
