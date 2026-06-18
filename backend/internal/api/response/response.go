package response

import "github.com/gin-gonic/gin"

func OK(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{"code": 0, "data": data})
}

func Message(c *gin.Context, msg string) {
	c.JSON(200, gin.H{"code": 0, "message": msg})
}

func Error(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"code": status, "error": msg})
}
