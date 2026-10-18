package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHub GitHubConfig `yaml:"github"`
	GitLab GitLabConfig `yaml:"gitlab"`
}

type GitHubConfig struct {
	Username string           `yaml:"username"`
	Repos    []GitHubRepoSpec `yaml:"repos"`
}

type GitHubRepoSpec struct {
	Owner string `yaml:"owner"`
	Name  string `yaml:"name"`
}

type GitLabConfig struct {
	Username string              `yaml:"username"`
	Projects []GitLabProjectSpec `yaml:"projects"`
}

type GitLabProjectSpec struct {
	Path string `yaml:"path"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	if len(cfg.GitHub.Repos) == 0 && len(cfg.GitLab.Projects) == 0 {
		return nil, fmt.Errorf("config %s: no github.repos or gitlab.projects configured", path)
	}

	return &cfg, nil
}
