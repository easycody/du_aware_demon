package awarent

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	gadapter "github.com/alibaba/sentinel-golang/adapter/gin"
	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/config"
	"github.com/alibaba/sentinel-golang/core/flow"
	metric "github.com/alibaba/sentinel-golang/core/log/metric"
	"github.com/gin-gonic/gin"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/util"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"gopkg.in/yaml.v2"
)

type Awarenter interface {
	Register() (bool, error)
	Deregister() (bool, error)
	GetConfig(configID string) (string, error)
	ConfigOnChange(configID string) error
}

//AwarentConfig entry struct
type AwarentConfig struct {
	ServiceName string `yaml:"serviceName" toml:"serviceName" json:"serviceName"`
	Port        uint64 `yaml:"port" toml:"port" json:"port"`
	Group       string `yaml:"group" toml:"group" json:"group"`
	Nacos       Nacos  `yaml:"nacos" toml:"nacos" json:"nacos"`
	ConfigID    string `yaml:"configId" toml:"configId" json:"configId"`
	RuleID      string `yaml:"ruleId" toml:"ruleId" json:"ruleId"`
}

type Nacos struct {
	Ip   string `yaml:"ip" toml:"ip" json:"ip"`
	Port uint64 `yaml:"port" toml:"port" json:"port"`
}

type Awarent struct {
	serviceName  string
	port         uint64
	group        string
	logDir       string
	nacosIP      string
	nacosPort    uint64
	configID     string
	ruleID       string
	nameClient   naming_client.INamingClient
	configClient config_client.IConfigClient
	rule         Rule
}

type FlowControlOption struct {
	Resource  string  `json:"resource"`
	Threshold float64 `json:"threshold"`
}

type Rule struct {
	ResourceParam    string              `yaml:"resource-param"`
	FlowControlRules []FlowControlOption `yaml:"flow-control-rules"`
	IPFilterRules    FilterOptions       `yaml:"ip-filter-rules"`
}

//InitAwarent init awarent module
func InitAwarent(entity AwarentConfig) (*Awarent, error) {
	logDir := os.TempDir() + string(os.PathSeparator) + entity.ServiceName

	awarent := &Awarent{
		serviceName: entity.ServiceName,
		group:       entity.Group,
		port:        entity.Port,
		logDir:      logDir,
		nacosIP:     entity.Nacos.Ip,
		nacosPort:   entity.Nacos.Port,
		configID:    entity.ConfigID,
		ruleID:      entity.RuleID,
	}

	sentinelConfig := config.NewDefaultConfig()
	sentinelConfig.Sentinel.App.Name = entity.ServiceName
	sentinelConfig.Sentinel.Log.Dir = logDir
	sc := []constant.ServerConfig{
		{
			IpAddr: awarent.nacosIP,   //"192.168.1.71"
			Port:   awarent.nacosPort, //8848
		},
	}
	cc := constant.ClientConfig{
		TimeoutMs:           5000,
		ListenInterval:      10000,
		NotLoadCacheAtStart: true,
		LogDir:              sentinelConfig.Sentinel.Log.Dir,
		CacheDir:            sentinelConfig.Sentinel.Log.Dir,
		RotateTime:          "1h",
		MaxAge:              3,
		LogLevel:            "debug",
	}
	nameClient, err := clients.CreateNamingClient(map[string]interface{}{
		"serverConfigs": sc,
		"clientConfig":  cc,
	})
	if err != nil {
		return nil, err
	}
	awarent.nameClient = nameClient

	configClient, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": sc,
		"clientConfig":  cc,
	})
	if err != nil {
		return nil, err
	}
	awarent.configClient = configClient
	sentinel.InitWithConfig(sentinelConfig)
	rc, err := awarent.GetConfig(awarent.ruleID)
	if err != nil {
		fmt.Printf("get flow control rule error:%v\n", err)
		return nil, err
	}
	yamlDecoder := yaml.NewDecoder(strings.NewReader(rc))
	var rule Rule
	if err = yamlDecoder.Decode(&rule); err != nil {
		fmt.Printf("decode yaml error:%v\n", err)
	}
	awarent.rule = rule
	fmt.Printf("load flow control rules:%s\n", util.ToJsonString(rule.FlowControlRules))
	awarent.LoadRules(rule.FlowControlRules...)
	awarent.Register()
	awarent.Subscribe()
	awarent.ConfigOnChange(awarent.ruleID)
	return awarent, nil
}

//Register register service
func (a *Awarent) Register() (bool, error) {
	regParam := vo.RegisterInstanceParam{
		ServiceName: a.serviceName,
		Ip:          util.LocalIP(),
		Port:        a.port,
		Weight:      10,
		Healthy:     true,
		Enable:      true,
		GroupName:   a.group,
	}

	return a.nameClient.RegisterInstance(regParam)
}

func (a *Awarent) Subscribe() error {
	subCallback := func(services []model.SubscribeService, err error) {
		if len(services) > 0 {
			actives := float64(len(services))
			fmt.Printf("subscribe callback return services:%s \n\n", util.ToJsonString(services))
			var newFlowControlRules []FlowControlOption
			for _, fr := range a.rule.FlowControlRules {
				newFlowRule := fr
				newFlowRule.Threshold = fr.Threshold / actives
				newFlowControlRules = append(newFlowControlRules, newFlowRule)
			}
			fmt.Printf("balanced flow control:%s \n", util.ToJsonString(newFlowControlRules))
			a.LoadRules(newFlowControlRules...)
		}
	}
	subParam := &vo.SubscribeParam{
		ServiceName:       a.serviceName,
		GroupName:         a.group,
		SubscribeCallback: subCallback,
	}
	return a.nameClient.Subscribe(subParam)
}

//Deregister deregister service
func (a *Awarent) Deregister() (bool, error) {
	vo := vo.DeregisterInstanceParam{
		Ip:        util.LocalIP(),
		Port:      a.port,
		GroupName: a.group,
	}
	return a.nameClient.DeregisterInstance(vo)
}

//GetConfig get config from nacos with config dataid
func (a *Awarent) GetConfig(configID string) (string, error) {
	return a.configClient.GetConfig(vo.ConfigParam{
		DataId: configID,
		Group:  a.group,
	})
}

//ConfigOnChange listen on config change
func (a *Awarent) ConfigOnChange(configID string) error {
	onChange := func(ns, group, dataId, data string) {
		fmt.Printf("config:%s changed, content:%s\n", configID, data)
		yamlDecoder := yaml.NewDecoder(strings.NewReader(data))
		var rule Rule
		if err := yamlDecoder.Decode(&rule); err != nil {
			fmt.Printf("decode yaml error:%v\n", err)
		}
		a.rule = rule

		//reload rules
		a.LoadRules(rule.FlowControlRules...)
		//reload ip filter rules
		newIPFilter := New(rule.IPFilterRules)
		ipfilter = newIPFilter
	}
	vo := vo.ConfigParam{
		Group:    a.group,
		DataId:   configID,
		OnChange: onChange,
	}
	return a.configClient.ListenConfig(vo)
}

//LoadRules load flow control rules
func (a *Awarent) LoadRules(rules ...FlowControlOption) (bool, error) {
	var sentinelRules []*flow.Rule
	for _, ruleItem := range rules {
		sentinelRule := &flow.Rule{
			Resource:               ruleItem.Resource,
			Threshold:              ruleItem.Threshold,
			TokenCalculateStrategy: flow.Direct,
			ControlBehavior:        flow.Reject,
			StatIntervalInMs:       1000,
		}
		sentinelRules = append(sentinelRules, sentinelRule)
	}
	return flow.LoadRules(sentinelRules)
}

// Metrics wrappers the standard http.Handler to gin.HandlerFunc
func (a *Awarent) Metrics() gin.HandlerFunc {
	searcher, err := metric.NewDefaultMetricSearcher(a.logDir, a.serviceName)
	if err != nil {
		return func(c *gin.Context) {
			c.AbortWithStatus(http.StatusInternalServerError)
		}
	}
	return func(c *gin.Context) {
		beginTimeMs := uint64((time.Now().Add(-2 * time.Second)).UnixNano() / 1e6)
		beginTimeMs = beginTimeMs - beginTimeMs%1000
		items, err := searcher.FindByTimeAndResource(beginTimeMs, beginTimeMs, "")
		if err != nil {
			c.String(http.StatusInternalServerError, "500 - Something bad")
			return
		}
		b := bytes.Buffer{}
		for _, item := range items {
			if len(item.Resource) == 0 {
				item.Resource = "__default__"
			}
			if fatStr, err := item.ToFatString(); err == nil {
				b.WriteString(fatStr)
				b.WriteByte('\n')
			}

		}
		c.String(http.StatusOK, b.String())
	}
}

var ipfilter *Filter

//IPFilter ip filter with options
func (a *Awarent) IPFilter() gin.HandlerFunc {
	opts := a.rule.IPFilterRules
	ipfilter = New(opts)
	return func(c *gin.Context) {
		if ipfilter.urlPath == c.Request.URL.Path {
			param := c.Query(ipfilter.urlParam)
			blocked := false
			ip := c.ClientIP()
			if !ipfilter.Allowed(ip) {
				blocked = true
			} else if !ipfilter.Authorized(ip, param) {
				blocked = true
			}
			if blocked {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
		}
		c.Next()
	}
}

//Sentinel awarent gin use middleware
func (a *Awarent) Sentinel() gin.HandlerFunc {
	param := a.rule.ResourceParam
	return gadapter.SentinelMiddleware(
		// customize resource extractor if required
		// method_path by default
		gadapter.WithResourceExtractor(func(ctx *gin.Context) string {
			return ctx.Query(param)
		}),
		// customize block fallback if required
		// abort with status 429 by default
		gadapter.WithBlockFallback(func(ctx *gin.Context) {
			ctx.AbortWithStatusJSON(400, map[string]interface{}{
				"err":  "too many request; the quota used up",
				"code": 10222,
			})
		}),
	)
}
