package handler

import (
"database/sql"
"fmt"
"net/http"
"time"
"story2video_server/db"
"story2video_server/utils"
"github.com/gin-gonic/gin"
"go.uber.org/zap"
)

// -------------------------- AI回调：分镜结果 --------------------------
func StoryResultCallback(c *gin.Context) {
type ShotResult struct {
SceneTitle  string `json:"scene_title"`
Prompt      string `json:"prompt"`
Narration   string `json:"narration"`
Transition  string `json:"transition"`
ShotStatus  string `json:"shot_status"`
ShotID      string `json:"shot_id"`
}

type StoryCallbackReq struct {
ProjectID string            `json:"project_id" binding:"required"`
Shots     map[string]ShotResult `json:"shots" binding:"required"`
}

var req StoryCallbackReq
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "回调参数错误: " + err.Error(),
})
return
}

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

for shotKey, shot := range req.Shots {
_, err := tx.Exec(`
INSERT INTO shots (shot_id, project_id, scene_title, prompt, narration, transition, shot_status, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE 
scene_title=VALUES(scene_title), prompt=VALUES(prompt), narration=VALUES(narration), 
transition=VALUES(transition), shot_status=VALUES(shot_status), update_time=?
`, shot.ShotID, req.ProjectID, shot.SceneTitle, shot.Prompt, shot.Narration, shot.Transition, shot.ShotStatus, now, now, now)
if err != nil {
tx.Rollback()
utils.Logger.Error(fmt.Sprintf("写入分镜%s数据失败", shotKey), zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  fmt.Sprintf("写入分镜%s数据失败", shotKey),
})
return
}
}

_, err = tx.Exec(`
UPDATE projects SET project_status='completed', update_time=? WHERE project_id=?
`, now, req.ProjectID)
if err != nil {
tx.Rollback()
utils.Logger.Error("更新项目状态失败", zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  "更新项目状态失败",
})
return
}

tx.Commit()
c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": nil,
"msg":  "分镜结果已写入RDS",
})
}

// -------------------------- AI回调：图片结果 --------------------------
func ImageResultCallback(c *gin.Context) {
type ImageShotResult struct {
ShotID     string `json:"shot_id" binding:"required"`
ImageURL   string `json:"image_url" binding:"required"`
ShotStatus string `json:"shot_status" binding:"required"`
}

type ImageCallbackReq struct {
ProjectID string                  `json:"project_id" binding:"required"`
Shots     map[string]ImageShotResult `json:"shots" binding:"required"`
}

var req ImageCallbackReq
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "回调参数错误: " + err.Error(),
})
return
}

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

for shotKey, shot := range req.Shots {
var imageTaskID string
err := tx.QueryRow(`
SELECT image_task_id FROM image_tasks WHERE project_id=? AND EXISTS (
SELECT 1 FROM shots WHERE shot_id=? AND project_id=?
)
`, req.ProjectID, shot.ShotID, req.ProjectID).Scan(&imageTaskID)
if err != nil && err != sql.ErrNoRows {
tx.Rollback()
utils.Logger.Error(fmt.Sprintf("查询图片任务%s失败", shotKey), zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  fmt.Sprintf("查询图片任务%s失败", shotKey),
})
return
}

_, err = tx.Exec(`
UPDATE image_tasks SET image_url=?, status=?, update_time=? WHERE image_task_id=?
`, shot.ImageURL, shot.ShotStatus, now, imageTaskID)
if err != nil {
tx.Rollback()
utils.Logger.Error(fmt.Sprintf("更新图片任务%s失败", shotKey), zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  fmt.Sprintf("更新图片任务%s失败", shotKey),
})
return
}

_, err = tx.Exec(`
UPDATE shots SET image_url=?, shot_status=?, update_time=? WHERE shot_id=? AND project_id=?
`, shot.ImageURL, shot.ShotStatus, now, shot.ShotID, req.ProjectID)
if err != nil {
tx.Rollback()
utils.Logger.Error(fmt.Sprintf("更新分镜%s图片URL失败", shotKey), zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  fmt.Sprintf("更新分镜%s图片URL失败", shotKey),
})
return
}
}

tx.Commit()
c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": nil,
"msg":  "图片结果已写入RDS",
})
}

// -------------------------- AI回调：视频结果 --------------------------
func VideoResultCallback(c *gin.Context) {
type VideoShotResult struct {
ShotID     string `json:"shot_id" binding:"required"`
VideoURL   string `json:"video_url" binding:"required"`
ShotStatus string `json:"shot_status" binding:"required"`
}

type VideoCallbackReq struct {
ProjectID string                  `json:"project_id" binding:"required"`
Shots     map[string]VideoShotResult `json:"shots" binding:"required"`
}

var req VideoCallbackReq
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "回调参数错误: " + err.Error(),
})
return
}

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

for shotKey, shot := range req.Shots {
var videoTaskID string
err := tx.QueryRow(`
SELECT video_task_id FROM video_tasks WHERE project_id=? AND EXISTS (
SELECT 1 FROM shots WHERE shot_id=? AND project_id=?
)
`, req.ProjectID, shot.ShotID, req.ProjectID).Scan(&videoTaskID)
if err != nil && err != sql.ErrNoRows {
tx.Rollback()
utils.Logger.Error(fmt.Sprintf("查询视频任务%s失败", shotKey), zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  fmt.Sprintf("查询视频任务%s失败", shotKey),
})
return
}

_, err = tx.Exec(`
UPDATE video_tasks SET video_url=?, status=?, update_time=? WHERE video_task_id=?
`, shot.VideoURL, shot.ShotStatus, now, videoTaskID)
if err != nil {
tx.Rollback()
utils.Logger.Error(fmt.Sprintf("更新视频任务%s失败", shotKey), zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  fmt.Sprintf("更新视频任务%s失败", shotKey),
})
return
}

_, err = tx.Exec(`
UPDATE shots SET video_url=?, shot_status=?, update_time=? WHERE shot_id=? AND project_id=?
`, shot.VideoURL, shot.ShotStatus, now, shot.ShotID, req.ProjectID)
if err != nil {
tx.Rollback()
utils.Logger.Error(fmt.Sprintf("更新分镜%s视频URL失败", shotKey), zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  fmt.Sprintf("更新分镜%s视频URL失败", shotKey),
})
return
}
}

tx.Commit()
c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": nil,
"msg":  "视频结果已写入RDS",
})
}
