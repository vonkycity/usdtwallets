# usdt 充值 订单系统

-------

本人水平有限，代码写的很烂，但不接受批评，不喜欢可以自己去写。

回调部分需自己完成，在Order.go第419行附近。

本系统可以接入各种商城系统，可自行接入，如需本人支持，可**有偿服务**，微信: tonyvicky1。

第一次运行需要按照提示进行。

下订单成功之后会返回钱包地址，一定要在10分钟之内付款才可以正常回调。

```
http://localhost:9000/payorder/
下订单地址 http://localhost:9000/payorder/订单号/appid/gid/time/price/sign/callbackurl

订单号一定要保证唯一
appid是4方用户的id
gid是用户唯一id
time是时间戳
price是价格
sign签名，算法   md5(appid + gid + price + 订单号 + time + appKey)
callbackurl 回调地址，如果使用get传入参数，一定要两次urlencode

其他接口请自己查看MainRouter.go:
router.GET("/payorder/:sdorderno/:appid/:gid/:t/:price/:sign/:callbackurl", orderController.WebGenOrder)
router.POST("/payorder", orderController.WebGenOrder)
router.GET("/orderstatus/:sdorderno/:appid/:t/:sign", orderController.WebOrderStatus)
router.POST("/orderstatus", orderController.WebOrderStatus)
router.GET("/admin/collectmoney/:wallet/:accesstoken", adminController.WebCollectMoney)
router.POST("/admin/collectmoney", adminController.WebCollectMoney)
router.GET("/admin/systemstatus/:accesstoken", adminController.WebSystemStatus)
router.POST("/admin/systemstatus", adminController.WebSystemStatus)
router.GET("/admin/walletlist/:page/:accesstoken", adminController.WebGetWalletList)
```

```
config.yaml是配置文件

如不需要ssl支持，可改为如下:
ssl:
  crt: ""
  key: ""
  port: 9001
```