package main

import (
	"flag"
	"strconv"

	"eth.url4g.com/config"
	logController "eth.url4g.com/controllers/log"

	//orderController "eth.url4g.com/controllers/order"
	_ "eth.url4g.com/controllers/admin"
	_ "eth.url4g.com/models"
	"eth.url4g.com/myutils"
	"eth.url4g.com/routers"
)

func main() {
	var serverType string

	flag.StringVar(&serverType, "t", "web", "服务类型，可选[web log]")
	flag.BoolVar(&myutils.EnableDebug, "d", false, "打开debug模式，默认为false")
	flag.Parse()
	//myutils.Println("服务器类型:", serverType)
	//if serverType == "log" {
	//	logController.UsdtEventRoutine()
	//}

	cfg := config.GetConfig()
	go logController.UsdtEventRoutine()
	mainRouter := routers.GetRouter()
	if cfg.Ssl.Crt != "" && cfg.Ssl.Key != "" && cfg.Ssl.Port != 0 {
		go mainRouter.RunTLS(cfg.Server.Addr+":"+strconv.Itoa(cfg.Ssl.Port), myutils.GetCurrentPath()+cfg.Ssl.Crt, myutils.GetCurrentPath()+cfg.Ssl.Key)
	}
	mainRouter.Run(cfg.Server.Addr + ":" + strconv.Itoa(cfg.Server.Port))
}
