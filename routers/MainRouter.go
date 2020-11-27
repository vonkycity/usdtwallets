package routers

import (
	"net/http"

	adminController "eth.url4g.com/controllers/admin"
	orderController "eth.url4g.com/controllers/order"
	"eth.url4g.com/myutils"
	"github.com/gin-gonic/gin"
)

//GetRouter 返回主router
func GetRouter() *gin.Engine {
	if myutils.EnableDebug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()
	router.Use(crossdomain())
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"Name":    "OrderServer",
			"Version": "1.0",
		})
	})
	router.GET("/payorder/:sdorderno/:appid/:gid/:t/:price/:sign/:callbackurl", orderController.WebGenOrder)
	router.POST("/payorder", orderController.WebGenOrder)
	router.GET("/orderstatus/:sdorderno/:appid/:t/:sign", orderController.WebOrderStatus)
	router.POST("/orderstatus", orderController.WebOrderStatus)
	router.GET("/admin/collectmoney/:wallet/:accesstoken", adminController.WebCollectMoney)
	router.POST("/admin/collectmoney", adminController.WebCollectMoney)
	router.GET("/admin/systemstatus/:accesstoken", adminController.WebSystemStatus)
	router.POST("/admin/systemstatus", adminController.WebSystemStatus)
	router.GET("/admin/walletlist/:page/:accesstoken", adminController.WebGetWalletList)
	return router
}

func crossdomain() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		c.Writer.Header().Add("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Add("Access-Control-Allow-Headers", "x-requested-with, Authorization, Content-Type")

		if method == "OPTIONS" {
			c.String(http.StatusOK, "")
		}
		c.Next()
	}
}
