package main

import (
	"du_aware_demon/awarent"
	"du_aware_demon/handlers"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

var (
	cfgFile string // 配置文件
)

// func init() {
// 	flag.StringVar(&cfgFile, "c", "conf/config.toml", "config file")
// }

// func Initialize() {

// }

func main() {
	// flag.Parse()
	aware, err := awarent.InitAwarent(awarent.AwarentConfig{
		ServiceName: "ddv",
		Port:        8080,
		Nacos: awarent.Nacos{
			Ip:   "192.168.1.71",
			Port: 8848,
		},
		Group:  "DDV_TEST",
		RuleID: "DDV_RULES",
	})
	if err != nil {
		panic("init awarent client error")
	}

	e := gin.New()
	e.Use(gin.Recovery())
	e.Use(aware.IPFilter())
	e.Use(aware.Sentinel())
	e.GET("/", func(c *gin.Context) {
		c.String(200, "OK")
	})
	e.HEAD("/", func(c *gin.Context) {
		c.AbortWithStatus(200)
	})
	e.GET("/awarent", aware.Metrics())
	e.GET("/q", handlers.GetDDV)
	srv := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: e,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			fmt.Printf("start server error:%v\n", err)
		}
	}()
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	<-quit
	aware.Deregister()
}
