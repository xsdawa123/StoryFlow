package handler

import (
"database/sql"
"fmt"
"math/rand"
"net/http"
"time"
"story2video_server/db"
"story2video_server/model"
"story2video_server/utils"
"github.com/gin-gonic/gin"
"go.uber.org/zap"
)

// -------------------------- 客户端接口：生成故事 --------------------------
func GenerateStory(c *gin.Context) {
type GenerateStoryReq struct {
ProjectID string `json:"project_id" binding:"required"`
RawStory  string `json:"raw_story" binding:"required"`
Style     string `json:"style" binding:"required"`
UserID    string `json:"user_id"`
}

var req GenerateStoryReq
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "参数错误: " + err.Error(),
})
return
}

// 写入 projects 表
now := time.Now().Format("2006-01-02 15:04:05")
_, err := db.DB.Exec(`
INSERT INTO projects (project_id, raw_story, style, project_status, create_time, update_time)
VALUES (?, ?, ?, 'generating', ?, ?)
ON DUPLICATE KEY UPDATE 
raw_story=VALUES(raw_story), style=VALUES(style), project_status='generating', update_time=?
`, req.ProjectID, req.RawStory, req.Style, now, now, now)
if err != nil {
utils.Logger.Error("写入项目数据失败", zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  "写入项目数据失败",
})
return
}

// 模拟调用AI Server（替换为真实AI地址）
storyTaskID := fmt.Sprintf("story_task_%s_%d", req.ProjectID, rand.Intn(10000))
c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": gin.H{
"story_task_id": storyTaskID,
"project_id":    req.ProjectID,
},
"msg": "故事分镜生成任务已提交",
})
}

// -------------------------- 客户端接口：生成分镜图片 --------------------------
func GenerateImages(c *gin.Context) {
type ShotReq struct {
ShotID   string `json:"shot_id" binding:"required"`
Prompt   string `json:"prompt" binding:"required"`
Style    string `json:"style" binding:"required"`
FrameNum int    `json:"frame_num" binding:"required"`
}

type GenerateImagesReq struct {
ProjectID string            `json:"project_id" binding:"required"`
StoryID   string            `json:"story_id" binding:"required"`
Shots     map[string]ShotReq `json:"shots" binding:"required"`
}

var req GenerateImagesReq
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "参数错误: " + err.Error(),
})
return
}

// 写入 image_tasks 表
now := time.Now().Format("2006-01-02 15:04:05")
tx, err := db.DB.Begin()
if err != nil {
utils.Logger.Error("开启事务失败", zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  "开启事务失败",
})
return
}

imageTaskIDs := make(map[string]string)
for shotKey, shot := range req.Shots {
imageTaskID := fmt.Sprintf("image_task_%s_%s", req.ProjectID, shot.ShotID)
imageTaskIDs[shot.ShotID] = imageTaskID

_, err := tx.Exec(`
INSERT INTO image_tasks (image_task_id, project_id, story_id, style, prompt, frame_num, status, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?, 'generating', ?, ?)
ON DUPLICATE KEY UPDATE 
style=VALUES(style), prompt=VALUES(prompt), frame_num=VALUES(frame_num), status='generating', update_time=?
`, imageTaskID, req.ProjectID, req.StoryID, shot.Style, shot.Prompt, shot.FrameNum, now, now, now)
if err != nil {
tx.Rollback()
utils.Logger.Error(fmt.Sprintf("写入图片任务%s失败", shotKey), zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  fmt.Sprintf("写入图片任务%s失败", shotKey),
})
return
}
}

tx.Commit()
imageTaskID := fmt.Sprintf("image_task_%s_%d", req.ProjectID, rand.Intn(10000))
c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": gin.H{
"image_task_id": imageTaskID,
"project_id":    req.ProjectID,
"shot_ids":      imageTaskIDs,
},
"msg": "图片生成任务已提交",
})
}

// -------------------------- 客户端接口：生成视频 --------------------------
func GenerateVideos(c *gin.Context) {
type VideoShotReq struct {
ShotID      string  `json:"shot_id" binding:"required"`
ImageTaskID string  `json:"image_task_id" binding:"required"`
Style       string  `json:"style" binding:"required"`
Duration    float64 `json:"duration" binding:"required"`
}

type GenerateVideosReq struct {
ProjectID string              `json:"project_id" binding:"required"`
Shots     map[string]VideoShotReq `json:"shots" binding:"required"`
}

var req GenerateVideosReq
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "参数错误: " + err.Error(),
})
return
}

// 写入 video_tasks 表
now := time.Now().Format("2006-01-02 15:04:05")
tx, err := db.DB.Begin()
if err != nil {
utils.Logger.Error("开启事务失败", zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  "开启事务失败",
})
return
}

videoTaskIDs := make(map[string]string)
for shotKey, shot := range req.Shots {
videoTaskID := fmt.Sprintf("video_task_%s_%s", req.ProjectID, shot.ShotID)
videoTaskIDs[shot.ShotID] = videoTaskID

_, err := tx.Exec(`
INSERT INTO video_tasks (video_task_id, project_id, image_task_id, style, duration, status, create_time, update_time)
VALUES (?, ?, ?, ?, ?, 'generating', ?, ?)
ON DUPLICATE KEY UPDATE 
style=VALUES(style), duration=VALUES(duration), status='generating', update_time=?
`, videoTaskID, req.ProjectID, shot.ImageTaskID, shot.Style, shot.Duration, now, now, now)
if err != nil {
tx.Rollback()
utils.Logger.Error(fmt.Sprintf("写入视频任务%s失败", shotKey), zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  fmt.Sprintf("写入视频任务%s失败", shotKey),
})
return
}
}

tx.Commit()
videoTaskID := fmt.Sprintf("video_task_%s_%d", req.ProjectID, rand.Intn(10000))
c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": gin.H{
"video_task_id": videoTaskID,
"project_id":    req.ProjectID,
"shot_ids":      videoTaskIDs,
},
"msg": "视频生成任务已提交",
})
}

// -------------------------- 客户端接口：查询分镜列表（匹配前端期望格式） --------------------------
func ListShots(c *gin.Context) {
projectID := c.Query("project_id")
if projectID == "" {
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "project_id不能为空",
})
return
}

// 1. 查询项目基本信息（获取status）
var projectStatus string
err := db.DB.QueryRow(`
SELECT project_status FROM projects WHERE project_id=?
`, projectID).Scan(&projectStatus)
if err != nil {
utils.Logger.Error("查询项目状态失败", zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  "查询项目状态失败",
})
return
}

// 2. 查询分镜数据
rows, err := db.DB.Query(`
SELECT shot_id, scene_title, prompt, transition, image_url
FROM shots WHERE project_id=?
`, projectID)
if err != nil {
utils.Logger.Error("查询分镜失败", zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  "查询分镜失败",
})
return
}
defer rows.Close()

// 3. 转换为前端期望的Storyboard数组格式
type Storyboard struct {
ShotId          string `json:"shotId"`
SceneTitle      string `json:"sceneTitle"`
Prompt          string `json:"prompt"`
Narration       string `json:"narration"` // 固定字符串
LocalImagePath  string `json:"localImagePath"`
Status          string `json:"status"` // 固定为generated
Transition      string `json:"transition"`
}
var storyboards []Storyboard
for rows.Next() {
var shotId, sceneTitle, prompt, transition, imageUrl string
err := rows.Scan(&shotId, &sceneTitle, &prompt, &transition, &imageUrl)
if err != nil {
utils.Logger.Error("解析分镜数据失败", zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  "解析分镜数据失败",
})
return
}
storyboards = append(storyboards, Storyboard{
ShotId:         shotId,
SceneTitle:     sceneTitle,
Prompt:         prompt,
Narration:      "这是分镜的默认旁白文字", // 固定字符串
LocalImagePath: imageUrl, // 映射原image_url
Status:         "generated", // 固定状态
Transition:     transition,
})
}

// 4. 组装前端期望的完整响应格式
type ProjectData struct {
Id           string        `json:"id"`
Name         string        `json:"name"` // 固定项目名称（若表中有则替换为表字段）
Storyboards  []Storyboard  `json:"storyboards"`
}
type ListShotsResp struct {
Code int `json:"code"`
Data struct {
Status      string      `json:"status"`
ProjectData ProjectData `json:"project_data"`
} `json:"data"`
Msg string `json:"msg"`
}

resp := ListShotsResp{
Code: 0,
Msg:  "分镜查询成功",
}
resp.Data.Status = projectStatus // 项目状态（来自projects表）
resp.Data.ProjectData = ProjectData{
Id:          projectID,
Name:        "项目名称", // 若projects表有name字段，可替换为表中数据
Storyboards: storyboards,
}

c.JSON(http.StatusOK, resp)
}

// -------------------------- 客户端接口：查询项目列表 --------------------------
func ListProjects(c *gin.Context) {
userID := c.Query("user_id")

query := `
SELECT project_id, raw_story, style, project_status, create_time, update_time
FROM projects
`
var args []interface{}
if userID != "" {
query += " WHERE user_id=?"
args = append(args, userID)
}

rows, err := db.DB.Query(query, args...)
if err != nil {
utils.Logger.Error("查询项目失败", zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  "查询项目失败",
})
return
}
defer rows.Close()

var projects []model.Project
for rows.Next() {
var project model.Project
err := rows.Scan(
&project.ProjectID, &project.RawStory, &project.Style,
&project.ProjectStatus, &project.CreateTime, &project.UpdateTime,
)
if err != nil {
utils.Logger.Error("解析项目数据失败", zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  "解析项目数据失败",
})
return
}
projects = append(projects, project)
}

c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": gin.H{
"projects": projects,
"total":    len(projects),
},
"msg": "项目列表查询成功",
})
}
