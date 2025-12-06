package response

import "github.com/gin-gonic/gin"

// 响应结构体（严格对齐前端格式）
type Response struct {
Type    string      `json:"type"`
Payload interface{} `json:"payload"`
}

// 成功响应
func Success(data interface{}) Response {
return Response{
Type:    "ACTION_TYPE_SUCCESS",
Payload: data,
}
}

// 失败响应
func Failed(msg string, data interface{}) Response {
if data == nil {
data = map[string]string{"error_msg": msg}
}
return Response{
Type:    "ACTION_TYPE_FAILED",
Payload: data,
}
}

// Gin响应助手（简化接口层代码）
func GinSuccess(c *gin.Context, data interface{}) {
c.JSON(200, Success(data))
}

func GinFailed(c *gin.Context, msg string, data interface{}) {
c.JSON(200, Failed(msg, data))
}
