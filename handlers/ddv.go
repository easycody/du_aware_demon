package handlers

import (
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
)

//GetDDV demon
func GetDDV(c *gin.Context) {
	r := rand.Intn(10)
	time.Sleep(time.Duration(r) * time.Millisecond)
	// cid := c.Query("cid")
	// log.Printf("cid=%s", cid)
	c.String(200, "%s", time.Now().Format(time.RFC3339))
}
