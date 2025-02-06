package configs

type GitLab struct {
	GitlabURL   string `json:"gitlab_url"`
	AccessToken string `json:"access_token"`
	MainGroupID int    `json:"main_group_id"`
}
