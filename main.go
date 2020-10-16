package main

import (
	"du_aware_demon/handlers"
	"du_aware_demon/module"
	"fmt"
	"net/http"

	"github.com/alibaba/sentinel-golang/core/flow"
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
	module.InitAwarent(module.Awarent{
		ServiceName: "ddv",
		Nacos:       "192.168.1.71:8848",
	})
	var rules []*flow.Rule
	f1 := &flow.Rule{
		Resource:               "bigdata",
		TokenCalculateStrategy: flow.Direct,
		ControlBehavior:        flow.Reject,
		Threshold:              10,
		StatIntervalInMs:       1000,
	}
	f2 := &flow.Rule{
		Resource:               "test",
		TokenCalculateStrategy: flow.Direct,
		ControlBehavior:        flow.Reject,
		Threshold:              10,
		StatIntervalInMs:       1000,
	}
	rules = append(rules, f1, f2)
	// go func() {
	// 	for i := 0; i < 100; i++ {
	// 		r := rand.Intn(5)
	// 		time.Sleep(time.Duration(r) * time.Second)
	// 		module.Metrics()
	// 	}

	// }()
	module.LoadRules(rules)
	e := gin.New()
	e.Use(gin.Recovery())

	e.Use(module.AwarentSentinel("cid"))
	e.GET("/awarent", module.AwarentMetrics())
	e.GET("/q", handlers.GetDDV)
	srv := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: e,
	}
	if err := srv.ListenAndServe(); err != nil {
		fmt.Printf("start server error:%v\n", err)
	}
}
