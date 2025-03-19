package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config 保存API网关的配置信息
type Config struct {
	Port     int       `yaml:"port"`
	Services []Service `yaml:"services"`
}

// Service 定义一个上游服务
type Service struct {
	Name      string            `yaml:"name"`
	Path      string            `yaml:"path"`
	Target    string            `yaml:"target"`
	StripPath bool              `yaml:"strip_path"`
	Headers   map[string]string `yaml:"headers"`
}

// Load 从文件加载配置
func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	// 设置默认值
	if config.Port == 0 {
		config.Port = 8080
	}

	return config, nil
}
