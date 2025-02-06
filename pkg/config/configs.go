package configs

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
)

type GitLab struct {
	GitlabURL   string `json:"gitlab_url" env-required:"true" env:"GITLAB_URL"`
	AccessToken string `json:"access_token" env-required:"true" env:"ACCESS_TOKEN"`
	MainGroupID int    `json:"main_group_id" env-required:"true" env:"MAIN_GROUP_ID"`
}

func NewGitLabConfig() (*GitLab, error) {
	configPath := filepath.Join(".", "config.personal.json")
	cfg := &GitLab{}

	err := cleanenv.ReadConfig(configPath, cfg)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	err = cleanenv.ReadEnv(cfg)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(cfg.GitlabURL, "https://") {
		return nil, fmt.Errorf("gitlab_url must use HTTPS (e.g., https://gitlab.example.com)")
	}

	return cfg, nil
}
