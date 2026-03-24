package contracts

type AIProviderConfig struct {
	Enabled bool     `json:"enabled" yaml:"enabled"`
	APIKey  string   `json:"api_key" yaml:"api_key"`
	BaseURL string   `json:"base_url" yaml:"base_url"`
	Models  []string `json:"models" yaml:"models"`
}

type AIBotConfig struct {
	Enabled             bool   `json:"enabled" yaml:"enabled"`
	Username            string `json:"username" yaml:"username"`
	DisplayName         string `json:"display_name" yaml:"display_name"`
	SystemPrompt        string `json:"system_prompt" yaml:"system_prompt"`
	DefaultProvider     string `json:"default_provider" yaml:"default_provider"`
	DefaultModel        string `json:"default_model" yaml:"default_model"`
	ContextMessages     int    `json:"context_messages" yaml:"context_messages"`
	AsyncTimeoutSeconds int    `json:"async_timeout_seconds" yaml:"async_timeout_seconds"`
}

type AIAuditConfig struct {
	Enabled       bool `json:"enabled" yaml:"enabled"`
	RetentionDays int  `json:"retention_days" yaml:"retention_days"`
}

type AIProvidersConfig struct {
	Zhipu     AIProviderConfig `json:"zhipu" yaml:"zhipu"`
	OpenAI    AIProviderConfig `json:"openai" yaml:"openai"`
	Anthropic AIProviderConfig `json:"anthropic" yaml:"anthropic"`
}

type AIConfig struct {
	Bot       AIBotConfig       `json:"bot" yaml:"bot"`
	Providers AIProvidersConfig `json:"providers" yaml:"providers"`
	Audit     AIAuditConfig     `json:"audit" yaml:"audit"`
}

type ListAIAuditLogsQuery struct {
	Limit          int    `form:"limit"`
	Status         string `form:"status"`
	Provider       string `form:"provider"`
	Model          string `form:"model"`
	UserID         uint64 `form:"user_id"`
	ConversationID uint64 `form:"conversation_id"`
}

type AIAuditLogDTO struct {
	ID             uint64 `json:"id"`
	UserID         uint64 `json:"user_id"`
	ConversationID uint64 `json:"conversation_id"`
	RequestID      string `json:"request_id"`
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	Status         string `json:"status"`
	DurationMS     int64  `json:"duration_ms"`
	InputTokens    int    `json:"input_tokens"`
	OutputTokens   int    `json:"output_tokens"`
	TotalTokens    int    `json:"total_tokens"`
	InputPreview   string `json:"input_preview"`
	OutputPreview  string `json:"output_preview"`
	ErrorCode      string `json:"error_code"`
	ErrorMessage   string `json:"error_message"`
	CreatedAt      string `json:"created_at"`
}

type AIAuditLogDetailDTO struct {
	AIAuditLogDTO
	RequestPayloadJSON  string `json:"request_payload_json"`
	ResponsePayloadJSON string `json:"response_payload_json"`
}

type ListAIRetryJobsQuery struct {
	Limit  int    `form:"limit"`
	Status string `form:"status"`
}

type AIRetryJobDTO struct {
	ID             uint64 `json:"id"`
	UserID         uint64 `json:"user_id"`
	ConversationID uint64 `json:"conversation_id"`
	Status         string `json:"status"`
	AttemptCount   int    `json:"attempt_count"`
	NextAttemptAt  string `json:"next_attempt_at"`
	LastError      string `json:"last_error"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type AIRetryJobDetailDTO struct {
	AIRetryJobDTO
}

type RetryAIRetryJobsRequest struct {
	IDs []uint64 `json:"ids"`
}

type CleanupAIRetryJobsRequest struct {
	Statuses []string `json:"statuses"`
}
