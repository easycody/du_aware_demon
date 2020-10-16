package awarent

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	gadapter "github.com/alibaba/sentinel-golang/adapter/gin"
	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/config"
	"github.com/alibaba/sentinel-golang/core/flow"
	metric "github.com/alibaba/sentinel-golang/core/log/metric"
	"github.com/gin-gonic/gin"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
)

type Awarenter interface {
	Register(serviceName string)
	DeRegister(serviceName string)
	GetConfig() (string,error)
	ConfigOnChange() func()
	LoadRules(rules []*flow.Rule)
}

//Awarent entry struct
type Awarent struct {
	ServiceName string `yml:"serviceName" toml:"serviceName" json:"serviceName"`
	Nacos       Nacos `yml:"nacos" toml:"nacos" json:"nacos"`
}

type Nacos struct {
	Ip string `yml:"ip" toml:"ip" json:"ip"`
	Port uint64 `yml:"port" toml:"port" json:"port"`
}

type awarentConfig struct {
	serviceName string
	logDir      string
	nacosIP       string
	nacosPort    uint64
	// sc          constant.ServerConfig //nacos server config
	// cc          constant.ClientConfig //nacos client config
	nameClient  naming_client.INamingClient
	configClient config_client.IConfigClient
}

var (
	curConfig awarentConfig
)

//InitAwarent init awarent module
func InitAwarent(entity Awarent) error {
	confEntity := config.NewDefaultConfig()
	confEntity.Sentinel.App.Name = entity.ServiceName
	confEntity.Sentinel.Log.Dir = os.TempDir() + string(os.PathSeparator) + entity.ServiceName
	
	curConfig.logDir = confEntity.Sentinel.Log.Dir
	curConfig.serviceName = entity.ServiceName
	curConfig.nacosIP = entity.Nacos.Ip
	curConfig.nacosPort = entity.Nacos.Port
	nacosAddrs := strings.Split(curConfig.nacos, ":")

	sc := []constant.ServerConfig{
		{
			IpAddr: curConfig.nacosIP, //"192.168.1.71"
			Port:   curConfig.nacosPort, //8848
		},
	}
	cc := constant.ClientConfig{
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              confEntity.Sentinel.Log.Dir,
		CacheDir:            confEntity.Sentinel.Log.Dir,
		RotateTime:          "1h",
		MaxAge:              3,
		LogLevel:            "debug",
	}
	nameClient, err := clients.CreateNamingClient(map[string]interface{}{
		"serverConfigs": sc,
		"clientConfig":  cc,
	})
	if err != nil {
		return err
	}
	curConfig.nameClient = nameClient 

	configClient, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": sc,
		"clientConfig":  cc,
	})
	if err != nil {
		return err 
	}
	curConfig.configClient = configClient
	registerService(nameClient, param vo.RegisterInstanceParam)

	
	sentinel.InitWithConfig(confEntity)

}

//LoadRules load flow control rules
func LoadRules(rules []*flow.Rule) (bool, error) {
	return flow.LoadRules(rules)
}

func Register() {
}

// Metrics wrappers the standard http.Handler to gin.HandlerFunc
func Metrics() gin.HandlerFunc {
	searcher, err := metric.NewDefaultMetricSearcher(curConfig.logDir, curConfig.serviceName)
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

func IPFilter(options FilterOptions) gin.HandlerFunc {
	opts := options
	ipfilter := New(opts)
	return func(c *gin.Context) {
		if ipfilter.urlPath == c.Request.URL.Path {
			blocked := false
			ip := c.ClientIP()
			if !ipfilter.Allowed(ip) {
				blocked = true
			} else if !ipfilter.Authorized(ip, ipfilter.urlParam) {
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
func Sentinel(param string) gin.HandlerFunc {
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
