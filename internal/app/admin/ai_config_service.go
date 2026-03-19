package admin

import (
	"os"
	"path/filepath"

	"awesomeproject/internal/platform/apperr"
	"awesomeproject/pkg/contracts"

	"github.com/goccy/go-yaml"
)

type AIConfigService struct {
	path string
}

func NewAIConfigService(path string) *AIConfigService {
	return &AIConfigService{path: path}
}

func (s *AIConfigService) EnsureFile() error {
	if _, err := os.Stat(s.path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	defaultConfig := contracts.AIConfig{
		Bot: contracts.AIBotConfig{
			Enabled:             true,
			Username:            "ai_bot",
			DisplayName:         "AI Bot",
			DefaultProvider:     "zhipu",
			DefaultModel:        "glm-4.5-air",
			ContextMessages:     20,
			AsyncTimeoutSeconds: 30,
		},
		Providers: contracts.AIProvidersConfig{
			Zhipu: contracts.AIProviderConfig{
				Enabled: true,
				BaseURL: "https://open.bigmodel.cn/api/coding/paas/v4",
				Models:  []string{"glm-4.5-air", "glm-4.5"},
			},
			OpenAI: contracts.AIProviderConfig{
				BaseURL: "https://api.openai.com/v1",
				Models:  []string{"gpt-4.1-mini", "gpt-4.1"},
			},
			Anthropic: contracts.AIProviderConfig{
				BaseURL: "https://api.anthropic.com/v1",
				Models:  []string{"claude-3-5-sonnet-latest"},
			},
		},
		Audit: contracts.AIAuditConfig{
			Enabled:       true,
			RetentionDays: 7,
		},
	}
	return s.Save(defaultConfig)
}

func (s *AIConfigService) Load() (*contracts.AIConfig, error) {
	body, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	var cfg contracts.AIConfig
	if err := yaml.Unmarshal(body, &cfg); err != nil {
		return nil, err
	}
	if err := s.Validate(cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *AIConfigService) Save(cfg contracts.AIConfig) error {
	if err := s.Validate(cfg); err != nil {
		return err
	}
	body, err := yaml.MarshalWithOptions(cfg, yaml.Indent(2))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(s.path), "ai-config-*.yaml")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(body); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func (s *AIConfigService) Validate(cfg contracts.AIConfig) error {
	if cfg.Bot.Username == "" || cfg.Bot.DisplayName == "" {
		return apperr.BadRequest("INVALID_AI_CONFIG", "bot username and display_name are required")
	}
	if cfg.Bot.DefaultProvider == "" || cfg.Bot.DefaultModel == "" {
		return apperr.BadRequest("INVALID_AI_CONFIG", "default provider and model are required")
	}
	if cfg.Bot.ContextMessages <= 0 {
		return apperr.BadRequest("INVALID_AI_CONFIG", "context_messages must be positive")
	}
	if cfg.Bot.AsyncTimeoutSeconds <= 0 {
		return apperr.BadRequest("INVALID_AI_CONFIG", "async_timeout_seconds must be positive")
	}
	if cfg.Audit.RetentionDays <= 0 {
		return apperr.BadRequest("INVALID_AI_CONFIG", "retention_days must be positive")
	}
	return nil
}
