package model

// Shot 分镜模型（匹配shots表）
type Shot struct {
ShotID        string `json:"shot_id"`
ProjectID     string `json:"project_id"`
SceneTitle    string `json:"scene_title"`
Prompt        string `json:"prompt"`
Narration     string `json:"narration"`
Transition    string `json:"transition"`
ShotStatus    string `json:"shot_status"`
ImageURL      string `json:"image_url"`
VideoURL      string `json:"video_url"`
CreateTime    string `json:"create_time"`
UpdateTime    string `json:"update_time"`
}
