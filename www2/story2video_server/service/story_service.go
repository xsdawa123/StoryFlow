package service

import (
"fmt"
"time"
"bytes"
"encoding/json"
"story2video_server/db"
"story2video_server/model"
"story2video_server/utils"
"story2video_server/response"
"github.com/google/uuid"
"github.com/gin-gonic/gin"
"go.uber.org/zap"
"net/http"
)

// 1. 生成故事（对应前端场景1-2）
func GenerateStory(c *gin.Context, rawStory, style string) {
// 生成Project ID和名称（名称简化，前端可自定义显示）
projectID := uuid.NewString()
projName := fmt.Sprintf("story_proj_%d", time.Now().Unix())

// 创建Project记录（写入RDS）
project := &model.Project{
ID:       projectID,
Name:     projName,
RawStory: rawStory,
Style:    style,
}
if err := db.DB.Create(project).Error; err != nil {
zap.L().Error("创建Project失败", zap.Error(err), zap.String("project_id", projectID))
response.GinFailed(c, "存储项目失败："+err.Error(), map[string]string{"name": projName})
return
}

// 调用AI Server生成分镜（按AI Server接口要求传参）
aiCfg := utils.GlobalConfig.AIServer
requestUrl := aiCfg.BaseUrl + aiCfg.GenerateStoryPath
requestData := map[string]string{
"project_id": projectID,
"raw_story":  rawStory,
"style":      style,
}
jsonData, _ := json.Marshal(requestData)

// 发送HTTP请求（设置超时30秒）
client := &http.Client{Timeout: 30 * time.Second}
req, _ := http.NewRequest("POST", requestUrl, bytes.NewBuffer(jsonData))
req.Header.Set("Content-Type", "application/json; charset=utf-8")
resp, err := client.Do(req)
if err != nil {
zap.L().Error("调用AI Server分镜接口失败", zap.Error(err), zap.String("project_id", projectID))
response.GinFailed(c, "分镜生成请求失败："+err.Error(), map[string]string{"name": projName})
return
}
defer resp.Body.Close()

// 解析AI Server返回的分镜数据（按AI Server响应格式）
var aiStoryResponse struct {
ProjectID string   `json:"project_id"`
Shot1     struct {
SceneTitle  string `json:"scene_title"`
Prompt      string `json:"prompt"`
Narration   string `json:"narration"`
Transition  string `json:"transition"`
ImageUrl    string `json:"image_url"`
} `json:"shot_1"`
Shot2     struct {
SceneTitle  string `json:"scene_title"`
Prompt      string `json:"prompt"`
Narration   string `json:"narration"`
Transition  string `json:"transition"`
ImageUrl    string `json:"image_url"`
} `json:"shot_2"`
}
if err := json.NewDecoder(resp.Body).Decode(&aiStoryResponse); err != nil {
zap.L().Error("解析AI Server分镜数据失败", zap.Error(err), zap.String("project_id", projectID))
response.GinFailed(c, "分镜数据解析失败", map[string]string{"name": projName})
return
}

// 生成前端要求的shotId（UUID），构造分镜数据
shot1Id := uuid.NewString()
shot2Id := uuid.NewString()
shots := []model.Shot{
{
ShotId:     shot1Id,
ProjectID:  projectID,
SceneTitle: aiStoryResponse.Shot1.SceneTitle,
Prompt:     aiStoryResponse.Shot1.Prompt,
Narration:  aiStoryResponse.Shot1.Narration,
Transition: aiStoryResponse.Shot1.Transition,
ShotStatus: "generated", // 对齐前端状态值
ImageUrl:   aiStoryResponse.Shot1.ImageUrl,
},
{
ShotId:     shot2Id,
ProjectID:  projectID,
SceneTitle: aiStoryResponse.Shot2.SceneTitle,
Prompt:     aiStoryResponse.Shot2.Prompt,
Narration:  aiStoryResponse.Shot2.Narration,
Transition: aiStoryResponse.Shot2.Transition,
ShotStatus: "generated",
ImageUrl:   aiStoryResponse.Shot2.ImageUrl,
},
}

// 存储分镜数据到RDS
if err := db.DB.Create(&shots).Error; err != nil {
zap.L().Error("存储分镜数据失败", zap.Error(err), zap.String("project_id", projectID))
response.GinFailed(c, "分镜存储失败", map[string]string{"name": projName})
return
}

// 按前端要求格式返回响应（对应前端场景2）
response.GinSuccess(c, map[string]interface{}{
"id":          projectID,
"name":        projName,
"style":       style,
"createTime":  project.CreateTime,
"storyboards": shots, // 对齐前端storyboards字段
})
}

// 2. 重新生成图像（对应前端场景3-4）
func RegenerateImage(c *gin.Context, projectId, shotId, prompt, style string) {
// 验证项目和分镜是否存在
var project model.Project
var shot model.Shot
if err := db.DB.Where("id = ?", projectId).First(&project).Error; err != nil {
zap.L().Error("项目不存在", zap.Error(err), zap.String("project_id", projectId))
response.GinFailed(c, "项目不存在", nil)
return
}
if err := db.DB.Where("shot_id = ? AND project_id = ?", shotId, projectId).First(&shot).Error; err != nil {
zap.L().Error("分镜不存在", zap.Error(err), zap.String("shot_id", shotId))
response.GinFailed(c, "分镜不存在", nil)
return
}

// 调用AI Server重新生成图片（按AI Server接口要求传参）
aiCfg := utils.GlobalConfig.AIServer
requestUrl := aiCfg.BaseUrl + aiCfg.GenerateImagePath
requestData := map[string]string{
"project_id": projectId,
"shot_id":    shotId,
"prompt":     prompt,
"style":      style,
}
jsonData, _ := json.Marshal(requestData)

// 发送HTTP请求
client := &http.Client{Timeout: 30 * time.Second}
req, _ := http.NewRequest("POST", requestUrl, bytes.NewBuffer(jsonData))
req.Header.Set("Content-Type", "application/json; charset=utf-8")
resp, err := client.Do(req)
if err != nil {
zap.L().Error("调用AI Server图片接口失败", zap.Error(err), zap.String("shot_id", shotId))
response.GinFailed(c, "图片重新生成失败", nil)
return
}
defer resp.Body.Close()

// 解析AI Server返回的新图片URL
var aiImageResponse struct {
ShotId   string `json:"shot_id"`
ImageUrl string `json:"image_url"`
Status   string `json:"status"`
}
if err := json.NewDecoder(resp.Body).Decode(&aiImageResponse); err != nil {
zap.L().Error("解析AI Server图片数据失败", zap.Error(err), zap.String("shot_id", shotId))
response.GinFailed(c, "图片数据解析失败", nil)
return
}

// 更新分镜数据（只更新变化的字段）
if err := db.DB.Model(&shot).Updates(map[string]interface{}{
"prompt":     prompt,
"image_url":  aiImageResponse.ImageUrl,
"shot_status": aiImageResponse.Status,
}).Error; err != nil {
zap.L().Error("更新分镜图片失败", zap.Error(err), zap.String("shot_id", shotId))
response.GinFailed(c, "分镜更新失败", nil)
return
}

// 按前端要求格式返回响应（对应前端场景4）
response.GinSuccess(c, map[string]interface{}{
"shotId":  shotId,
"imageUrl": aiImageResponse.ImageUrl,
"status":   aiImageResponse.Status,
})
}

// 3. 生成视频（对应前端场景5-6）
func GenerateVideo(c *gin.Context, reqData interface{}) {
// 解析请求参数（前端场景5格式）
reqJson, _ := json.Marshal(reqData)
var videoReq struct {
ProjectId string `json:"projectId"`
Global    struct {
Resolution []int   `json:"resolution"`
Fps        int     `json:"fps"`
BgmUrl     string  `json:"bgmUrl"`
BgmVolume  float64 `json:"bgmVolume"`
} `json:"global"`
Track []struct {
Type       string `json:"type"`
Id         string `json:"id"`
Duration   int    `json:"duration"`
Transition string `json:"transition"`
Assets     struct {
Image string `json:"image"`
Audio string `json:"audio"`
Text  string `json:"text"`
} `json:"assets"`
} `json:"track"`
}
if err := json.Unmarshal(reqJson, &videoReq); err != nil {
zap.L().Error("解析前端视频请求参数失败", zap.Error(err))
response.GinFailed(c, "参数解析失败", nil)
return
}

// 验证项目是否存在
var project model.Project
if err := db.DB.Where("id = ?", videoReq.ProjectId).First(&project).Error; err != nil {
zap.L().Error("项目不存在", zap.Error(err), zap.String("project_id", videoReq.ProjectId))
response.GinFailed(c, "项目不存在", nil)
return
}

// 调用AI Server生成视频（转发前端参数，不做修改）
aiCfg := utils.GlobalConfig.AIServer
requestUrl := aiCfg.BaseUrl + aiCfg.GenerateVideoPath
jsonData, _ := json.Marshal(videoReq)

// 视频生成耗时较长，延长超时到60秒
client := &http.Client{Timeout: 60 * time.Second}
// 关键修复：变量名改为httpReq，避免和函数参数reqData重名
httpReq, _ := http.NewRequest("POST", requestUrl, bytes.NewBuffer(jsonData))
httpReq.Header.Set("Content-Type", "application/json; charset=utf-8")
resp, err := client.Do(httpReq)
if err != nil {
zap.L().Error("调用AI Server视频接口失败", zap.Error(err), zap.String("project_id", videoReq.ProjectId))
response.GinFailed(c, "视频生成请求失败", nil)
return
}
defer resp.Body.Close()

// 解析AI Server返回的视频URL
var aiVideoResponse struct {
ProjectId string `json:"project_id"`
VideoUrl  string `json:"video_url"`
Status    string `json:"status"`
}
if err := json.NewDecoder(resp.Body).Decode(&aiVideoResponse); err != nil {
zap.L().Error("解析AI Server视频数据失败", zap.Error(err), zap.String("project_id", videoReq.ProjectId))
response.GinFailed(c, "视频数据解析失败", nil)
return
}

// 更新项目状态和视频URL
if err := db.DB.Model(&project).Updates(map[string]interface{}{
"video_url":      aiVideoResponse.VideoUrl,
"project_status": aiVideoResponse.Status,
}).Error; err != nil {
zap.L().Error("更新项目视频信息失败", zap.Error(err), zap.String("project_id", videoReq.ProjectId))
response.GinFailed(c, "项目状态更新失败", nil)
return
}

// 按前端要求格式返回响应（对应前端场景6）
response.GinSuccess(c, map[string]interface{}{
"id":        videoReq.ProjectId,
"videoUrl":  aiVideoResponse.VideoUrl,
"status":    aiVideoResponse.Status,
"loading": map[string]bool{
"exportingVideo": false,
},
})
}
