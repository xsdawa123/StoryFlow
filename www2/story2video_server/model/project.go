package model

// Project 项目模型（匹配projects表）
type Project struct {
ProjectID     string `json:"project_id"`
RawStory      string `json:"raw_story"`
Style         string `json:"style"`
ProjectStatus string `json:"project_status"`
CreateTime    string `json:"create_time"`
UpdateTime    string `json:"update_time"`
}
