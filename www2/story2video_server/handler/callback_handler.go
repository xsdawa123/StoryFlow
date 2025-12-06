package handler

import (
"log"
"net/http"

"story2video/utils" // 新增：导入utils包
"github.com/gin-gonic/gin"
)

// -------------------------- 分镜回调（处理异步状态更新）--------------------------
func StoryResultCallback(c *gin.Context) {
var callbackData map[string]interface{}
if err := c.ShouldBindJSON(&callbackData); err != nil {
log.Printf("【分镜回调】解析参数失败: %v", err)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"msg":  "参数错误",
})
return
}

log.Printf("【分镜回调】接收AI Server数据 | 数据: %+v", callbackData)

// 文档流程1：回调可能先返回状态（project_status=generating）
projectID, ok := callbackData["project_id"].(string)
if !ok || projectID == "" {
log.Printf("【分镜回调】project_id缺失 | 数据: %+v", callbackData)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"msg":  "project_id缺失",
})
return
}

// 处理状态更新（对齐文档story_status映射）
if projectStatus, ok := callbackData["project_status"].(string); ok {
if utils.DB != nil {
_, err := utils.DB.Exec(`
UPDATE projects SET project_status=?, update_time=NOW()
WHERE project_id=?
`, projectStatus, projectID)
if err != nil {
log.Printf("【分镜回调】更新projects状态失败 | 项目ID: %s | 状态: %s | 错误: %v", projectID, projectStatus, err)
} else {
log.Printf("【分镜回调】更新projects状态成功 | 项目ID: %s | 状态: %s", projectID, projectStatus)
}
}
}

// 处理完整响应（若回调携带shot数据，补充写入shots表）
for key, value := range callbackData {
if len(key) >= 5 && key[:5] == "shot_" {
shotMap, ok := value.(map[string]interface{})
if !ok {
continue
}
sceneTitle := shotMap["scene_title"].(string)
prompt := shotMap["prompt"].(string)
narration := shotMap["narration"].(string)
transition := shotMap["transition"].(string)
shotStatus := shotMap["shot_status"].(string)

if utils.DB != nil { // 新增：判空，避免nil指针
_, err := utils.DB.Exec(`
INSERT INTO shots (project_id, scene_title, prompt, narration, transition, shot_status, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())
ON DUPLICATE KEY UPDATE update_time=NOW()
`, projectID, sceneTitle, prompt, narration, transition, shotStatus)
if err != nil {
log.Printf("【分镜回调】写入shots表失败 | 项目ID: %s | shot: %s | 错误: %v", projectID, key, err)
}
}
}
}

c.JSON(http.StatusOK, gin.H{
"code": 0,
"msg":  "分镜回调接收成功",
})
}

// -------------------------- 图片回调（处理异步状态更新）--------------------------
func ImageResultCallback(c *gin.Context) {
var callbackData map[string]interface{}
if err := c.ShouldBindJSON(&callbackData); err != nil {
log.Printf("【图片回调】解析参数失败: %v", err)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"msg":  "参数错误",
})
return
}

log.Printf("【图片回调】接收AI Server数据 | 数据: %+v", callbackData)

projectID, ok := callbackData["project_id"].(string)
if !ok || projectID == "" {
log.Printf("【图片回调】project_id缺失 | 数据: %+v", callbackData)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"msg":  "project_id缺失",
})
return
}

// 处理shot状态更新（对齐文档image_status映射）
for key, value := range callbackData {
if len(key) >= 5 && key[:5] == "shot_" {
shotMap, ok := value.(map[string]interface{})
if !ok {
continue
}
shotID := shotMap["shot_id"].(string)
shotStatus := shotMap["shot_status"].(string)

// 若携带image_url，同步更新（对齐image_response）
imageURL := ""
if url, ok := shotMap["image_url"].(string); ok {
imageURL = url
}

if utils.DB != nil { // 新增：判空，避免nil指针
_, err := utils.DB.Exec(`
UPDATE shots SET shot_status=?, image_url=?, update_time=NOW()
WHERE project_id=? AND id=?
`, shotStatus, imageURL, projectID, shotID)
if err != nil {
log.Printf("【图片回调】更新shots表失败 | 项目ID: %s | shot_id: %s | 错误: %v", projectID, shotID, err)
} else {
log.Printf("【图片回调】更新shots表成功 | 项目ID: %s | shot_id: %s | 状态: %s", projectID, shotID, shotStatus)
}
}
}
}

c.JSON(http.StatusOK, gin.H{
"code": 0,
"msg":  "图片回调接收成功",
})
}

// -------------------------- 音频回调（处理异步状态更新）--------------------------
func AudioResultCallback(c *gin.Context) {
var callbackData map[string]interface{}
if err := c.ShouldBindJSON(&callbackData); err != nil {
log.Printf("【音频回调】解析参数失败: %v", err)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"msg":  "参数错误",
})
return
}

log.Printf("【音频回调】接收AI Server数据 | 数据: %+v", callbackData)

projectID, ok := callbackData["project_id"].(string)
if !ok || projectID == "" {
log.Printf("【音频回调】project_id缺失 | 数据: %+v", callbackData)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"msg":  "project_id缺失",
})
return
}

// 处理shot状态更新（对齐文档audio_status映射）
for key, value := range callbackData {
if len(key) >= 5 && key[:5] == "shot_" {
shotMap, ok := value.(map[string]interface{})
if !ok {
continue
}
shotID := shotMap["shot_id"].(string)
shotStatus := shotMap["shot_status"].(string)

// 若携带audio_url，同步更新（对齐audio_response）
audioURL := ""
if url, ok := shotMap["audio_url"].(string); ok {
audioURL = url
}

if utils.DB != nil { // 新增：判空，避免nil指针
_, err := utils.DB.Exec(`
UPDATE shots SET shot_status=?, audio_url=?, update_time=NOW()
WHERE project_id=? AND id=?
`, shotStatus, audioURL, projectID, shotID)
if err != nil {
log.Printf("【音频回调】更新shots表失败 | 项目ID: %s | shot_id: %s | 错误: %v", projectID, shotID, err)
} else {
log.Printf("【音频回调】更新shots表成功 | 项目ID: %s | shot_id: %s | 状态: %s", projectID, shotID, shotStatus)
}
}
}
}

c.JSON(http.StatusOK, gin.H{
"code": 0,
"msg":  "音频回调接收成功",
})
}
