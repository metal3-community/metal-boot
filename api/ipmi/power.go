package ipmi

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/squarefactory/ipmitool"
)

func PowerOn(c *gin.Context) {
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

	if status == ipmitool.PowerStateOff {
		err := cl.Power.On()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Success"})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Compute node already powered on"})
	}
}

func PowerOff(c *gin.Context) {
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

	if status == ipmitool.PowerStateOn {
		err := cl.Power.Off()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Success"})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Compute node already powered off"})
	}
}

func Cycle(c *gin.Context) {
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
	if status == ipmitool.PowerStateOn {
		err := cl.Power.Cycle()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Success"})

	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Compute node already powered off"})
	}
}

func Soft(c *gin.Context) {
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

	if status == ipmitool.PowerStateOn {
		err := cl.Power.Soft()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Success"})

	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Compute node already powered off"})
	}
}

func Reset(c *gin.Context) {
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

	if status == ipmitool.PowerStateOn {
		err := cl.Power.Reset()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Success"})

	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Compute node already powered off"})
	}
}
