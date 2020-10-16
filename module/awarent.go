package module

import (
	"bytes"
	"net/http"
	"os"
	"time"

	gadapter "github.com/alibaba/sentinel-golang/adapter/gin"
	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/config"
	"github.com/alibaba/sentinel-golang/core/flow"
	metric "github.com/alibaba/sentinel-golang/core/log/metric"
	"github.com/gin-gonic/gin"
)

//Awarent entry struct
type Awarent struct {
	ServiceName string `yml:"serviceName" toml:"serviceName" json:"serviceName"`
	Nacos       string `yml:"nacos" toml:"nacos" json:"nacos"`
}

type awarentConfig struct {
	serviceName string
	logDir      string
	nacos       string
}

var (
	curConfig awarentConfig
)

//InitAwarent init awarent module
func InitAwarent(entity Awarent) {
	confEntity := config.NewDefaultConfig()
	confEntity.Sentinel.App.Name = entity.ServiceName
	confEntity.Sentinel.Log.Dir = os.TempDir() + string(os.PathSeparator) + entity.ServiceName
	curConfig.logDir = confEntity.Sentinel.Log.Dir
	curConfig.serviceName = entity.ServiceName
	curConfig.nacos = entity.Nacos
	sentinel.InitWithConfig(confEntity)
}

//LoadRules load flow control rules
func LoadRules(rules []*flow.Rule) (bool, error) {
	return flow.LoadRules(rules)
}

// AwarentMetrics wrappers the standard http.Handler to gin.HandlerFunc
func AwarentMetrics() gin.HandlerFunc {
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

func AwarentIPFilter(options FilterOptions) gin.HandlerFunc {
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

//AwarentSentinel awarent gin use middleware
func AwarentSentinel(param string) gin.HandlerFunc {
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
