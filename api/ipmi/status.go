package ipmi

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/squarefactory/ipmitool"
)

func Status(c *gin.Context) {

	hostname := c.Param("host")
	hostIP := os.Getenv(hostname)
	data, err := c.GetRawData()

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	credential := Credential{}
	if err := json.Unmarshal(data, &credential); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(hostIP) == 0 {
		c.JSON(http.StatusNoContent, gin.H{"error": "Host not defined"})
		return
	}

	cl, err := ipmitool.NewClient(hostIP, 0, credential.Username, credential.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	status, err := cl.Power.Status()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": status})
}
