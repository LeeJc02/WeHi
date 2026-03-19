package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"awesomeproject/internal/app/repository"
	"awesomeproject/pkg/contracts"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type ConfigLoader interface {
	Load() (*contracts.AIConfig, error)
}

type Service struct {
	repo       *repository.Repository
	configs    ConfigLoader
	httpClient *http.Client
	replyFn    func(conversationID, botUserID uint64, messageType, content string) error
}

type BotReply struct {
	BotUserID uint64
	Provider  string
	Model     string
	Content   string
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

var aiRetryJobsGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "chat_ai_retry_jobs",
	Help: "Current number of AI retry jobs grouped by status.",
}, []string{"status"})

func NewService(repo *repository.Repository, configs ConfigLoader) *Service {
	return &Service{
		repo:    repo,
		configs: configs,
		httpClient: &http.Client{
			Timeout: 40 * time.Second,
		},
	}
}

func (s *Service) Start(ctx context.Context) {
	s.refreshRetryMetrics()
	go func() {
		s.cleanupExpiredAuditLogs()
		s.cleanupExpiredRetryJobs()
		s.processRetryJobs(context.Background())
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.cleanupExpiredAuditLogs()
				s.cleanupExpiredRetryJobs()
			}
		}
	}()
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.processRetryJobs(ctx)
			}
		}
	}()
}

func (s *Service) EnsureBotForUser(userID uint64) error {
	cfg, err := s.configs.Load()
	if err != nil {
		return err
	}
	if !cfg.Bot.Enabled {
		return nil
	}
	botUser, err := s.ensureBotUser(cfg)
	if err != nil {
		return err
	}
	if botUser.ID == userID {
		return nil
	}
	if err := s.repo.EnsureFriendship(userID, botUser.ID); err != nil {
		return err
	}
	if err := s.repo.EnsureFriendship(botUser.ID, userID); err != nil {
		return err
	}
	conversation, _, err := s.repo.EnsureDirectConversation(userID, botUser.ID)
	if err != nil {
		return err
	}
	return s.repo.EnsureConversationPinned(userID, conversation.ID)
}

func (s *Service) BotUserID() (uint64, error) {
	cfg, err := s.configs.Load()
	if err != nil {
		return 0, err
	}
	if !cfg.Bot.Enabled {
		return 0, nil
	}
	user, err := s.repo.FindUserByUsername(cfg.Bot.Username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}

func (s *Service) IsBotUser(userID uint64) (bool, error) {
	botUserID, err := s.BotUserID()
	if err != nil {
		return false, err
	}
	return botUserID > 0 && botUserID == userID, nil
}

func (s *Service) IsBotConversation(userID, conversationID uint64) (bool, error) {
	botUserID, err := s.BotUserID()
	if err != nil {
		return false, err
	}
	if botUserID == 0 {
		return false, nil
	}
	conversation, err := s.repo.FindConversationByID(conversationID)
	if err != nil {
		return false, err
	}
	if conversation.Type != "direct" {
		return false, nil
	}
	if _, err := s.repo.FindConversationMember(conversationID, userID); err != nil {
		return false, nil
	}
	if _, err := s.repo.FindConversationMember(conversationID, botUserID); err != nil {
		return false, nil
	}
	return true, nil
}

func (s *Service) SetReplyHandler(fn func(conversationID, botUserID uint64, messageType, content string) error) {
	s.replyFn = fn
}

func (s *Service) AsyncTimeout() (time.Duration, error) {
	cfg, err := s.configs.Load()
	if err != nil {
		return 0, err
	}
	if cfg.Bot.AsyncTimeoutSeconds <= 0 {
		return 30 * time.Second, nil
	}
	return time.Duration(cfg.Bot.AsyncTimeoutSeconds) * time.Second, nil
}

func (s *Service) GenerateReply(ctx context.Context, userID, conversationID uint64) (*BotReply, error) {
	cfg, err := s.configs.Load()
	if err != nil {
		return nil, err
	}
	if !cfg.Bot.Enabled {
		return nil, fmt.Errorf("ai bot disabled")
	}
	botUser, err := s.ensureBotUser(cfg)
	if err != nil {
		return nil, err
	}
	messages, err := s.buildContextMessages(cfg, conversationID, botUser.ID)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("empty ai conversation context")
	}

	providerName := strings.TrimSpace(cfg.Bot.DefaultProvider)
	modelName := strings.TrimSpace(cfg.Bot.DefaultModel)
	providerCfg, err := providerConfig(cfg, providerName)
	if err != nil {
		return nil, err
	}
	if !providerCfg.Enabled {
		return nil, fmt.Errorf("provider %s disabled", providerName)
	}
	if strings.TrimSpace(providerCfg.APIKey) == "" {
		return nil, fmt.Errorf("provider %s api key is empty", providerName)
	}

	requestBody, responseBody, usage, requestID, content, err := s.callProvider(ctx, providerName, providerCfg, modelName, strings.TrimSpace(cfg.Bot.SystemPrompt), messages)
	if err != nil {
		s.writeAuditLog(cfg, userID, conversationID, providerName, modelName, requestID, requestBody, responseBody, usage, "", err)
		return nil, err
	}
	s.writeAuditLog(cfg, userID, conversationID, providerName, modelName, requestID, requestBody, responseBody, usage, content, nil)
	return &BotReply{
		BotUserID: botUser.ID,
		Provider:  providerName,
		Model:     modelName,
		Content:   content,
	}, nil
}

func (s *Service) ensureBotUser(cfg *contracts.AIConfig) (*repository.User, error) {
	user, err := s.repo.FindUserByUsername(cfg.Bot.Username)
	if err == nil {
		if user.DisplayName != cfg.Bot.DisplayName {
			if updateErr := s.repo.UpdateUserProfile(user.ID, cfg.Bot.DisplayName, user.AvatarURL); updateErr == nil {
				user.DisplayName = cfg.Bot.DisplayName
			}
		}
		return user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(fmt.Sprintf("ai-bot:%s:%d", cfg.Bot.Username, time.Now().UnixNano())), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	row := &repository.User{
		Username:     cfg.Bot.Username,
		DisplayName:  cfg.Bot.DisplayName,
		PasswordHash: string(passwordHash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.CreateUser(row); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return nil, err
		}
		return s.repo.FindUserByUsername(cfg.Bot.Username)
	}
	return row, nil
}

func (s *Service) buildContextMessages(cfg *contracts.AIConfig, conversationID, botUserID uint64) ([]chatMessage, error) {
	rows, err := s.repo.ListMessages(conversationID, 0, cfg.Bot.ContextMessages)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	result := make([]chatMessage, 0, len(rows))
	for index := len(rows) - 1; index >= 0; index-- {
		row := rows[index]
		text := renderMessageContent(row.MessageType, row.Content, row.AttachmentJSON, row.RecalledAt != nil)
		if strings.TrimSpace(text) == "" {
			continue
		}
		role := "user"
		if row.SenderID == botUserID {
			role = "assistant"
		}
		result = append(result, chatMessage{Role: role, Content: text})
	}
	return result, nil
}

func providerConfig(cfg *contracts.AIConfig, provider string) (contracts.AIProviderConfig, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "zhipu":
		return cfg.Providers.Zhipu, nil
	case "openai":
		return cfg.Providers.OpenAI, nil
	case "anthropic":
		return cfg.Providers.Anthropic, nil
	default:
		return contracts.AIProviderConfig{}, fmt.Errorf("unsupported provider %s", provider)
	}
}

func renderMessageContent(messageType, content, attachmentJSON string, recalled bool) string {
	if recalled {
		return "[消息已撤回]"
	}
	switch messageType {
	case "image":
		return "[图片]"
	case "file":
		var attachment contracts.AttachmentDTO
		if attachmentJSON != "" && json.Unmarshal([]byte(attachmentJSON), &attachment) == nil && attachment.Filename != "" {
			return "[文件] " + attachment.Filename
		}
		return "[文件]"
	default:
		return strings.TrimSpace(content)
	}
}

func (s *Service) callProvider(ctx context.Context, providerName string, providerCfg contracts.AIProviderConfig, model, systemPrompt string, messages []chatMessage) (string, string, usageSummary, string, string, error) {
	switch strings.ToLower(providerName) {
	case "zhipu", "openai":
		return s.callOpenAICompatible(ctx, providerName, providerCfg, model, systemPrompt, messages)
	case "anthropic":
		return s.callAnthropic(ctx, providerCfg, model, systemPrompt, messages)
	default:
		return "", "", usageSummary{}, "", "", fmt.Errorf("unsupported provider %s", providerName)
	}
}

type usageSummary struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	DurationMS   int64
}

type openAIChatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type openAIChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (s *Service) callOpenAICompatible(ctx context.Context, providerName string, providerCfg contracts.AIProviderConfig, model, systemPrompt string, messages []chatMessage) (string, string, usageSummary, string, string, error) {
	payloadMessages := make([]chatMessage, 0, len(messages)+1)
	if systemPrompt != "" {
		payloadMessages = append(payloadMessages, chatMessage{Role: "system", Content: systemPrompt})
	}
	payloadMessages = append(payloadMessages, messages...)
	requestPayload := openAIChatRequest{
		Model:    model,
		Messages: payloadMessages,
		Stream:   false,
	}
	body, err := json.Marshal(requestPayload)
	if err != nil {
		return "", "", usageSummary{}, "", "", err
	}
	url := strings.TrimRight(providerCfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return string(body), "", usageSummary{}, "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+providerCfg.APIKey)
	startedAt := time.Now()
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return string(body), "", usageSummary{}, "", "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	usage := usageSummary{DurationMS: time.Since(startedAt).Milliseconds()}
	if resp.StatusCode >= http.StatusBadRequest {
		return string(body), string(raw), usage, resp.Header.Get("X-Request-Id"), "", fmt.Errorf("%s provider returned %d", providerName, resp.StatusCode)
	}
	var parsed openAIChatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return string(body), string(raw), usage, resp.Header.Get("X-Request-Id"), "", err
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return string(body), string(raw), usage, parsed.ID, "", fmt.Errorf("%s provider returned empty response", providerName)
	}
	usage.InputTokens = parsed.Usage.PromptTokens
	usage.OutputTokens = parsed.Usage.CompletionTokens
	usage.TotalTokens = parsed.Usage.TotalTokens
	return string(body), string(raw), usage, firstNonEmpty(resp.Header.Get("X-Request-Id"), parsed.ID), strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

type anthropicRequest struct {
	Model     string `json:"model"`
	System    string `json:"system,omitempty"`
	MaxTokens int    `json:"max_tokens"`
	Messages  []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type anthropicResponse struct {
	ID      string `json:"id"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (s *Service) callAnthropic(ctx context.Context, providerCfg contracts.AIProviderConfig, model, systemPrompt string, messages []chatMessage) (string, string, usageSummary, string, string, error) {
	payload := anthropicRequest{
		Model:     model,
		System:    systemPrompt,
		MaxTokens: 1024,
		Messages: make([]struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}, 0, len(messages)),
	}
	for _, message := range messages {
		payload.Messages = append(payload.Messages, struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			Role:    message.Role,
			Content: message.Content,
		})
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", usageSummary{}, "", "", err
	}
	url := strings.TrimRight(providerCfg.BaseURL, "/") + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return string(body), "", usageSummary{}, "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", providerCfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	startedAt := time.Now()
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return string(body), "", usageSummary{}, "", "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	usage := usageSummary{DurationMS: time.Since(startedAt).Milliseconds()}
	if resp.StatusCode >= http.StatusBadRequest {
		return string(body), string(raw), usage, resp.Header.Get("request-id"), "", fmt.Errorf("anthropic provider returned %d", resp.StatusCode)
	}
	var parsed anthropicResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return string(body), string(raw), usage, resp.Header.Get("request-id"), "", err
	}
	content := ""
	for _, block := range parsed.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			content = strings.TrimSpace(block.Text)
			break
		}
	}
	if content == "" {
		return string(body), string(raw), usage, parsed.ID, "", fmt.Errorf("anthropic provider returned empty response")
	}
	usage.InputTokens = parsed.Usage.InputTokens
	usage.OutputTokens = parsed.Usage.OutputTokens
	usage.TotalTokens = parsed.Usage.InputTokens + parsed.Usage.OutputTokens
	return string(body), string(raw), usage, firstNonEmpty(resp.Header.Get("request-id"), parsed.ID), content, nil
}

func (s *Service) writeAuditLog(cfg *contracts.AIConfig, userID, conversationID uint64, provider, model, requestID, requestPayload, responsePayload string, usage usageSummary, output string, callErr error) {
	if !cfg.Audit.Enabled {
		return
	}
	status := "success"
	errorCode := ""
	errorMessage := ""
	if callErr != nil {
		status = "error"
		errorCode = "AI_PROVIDER_ERROR"
		errorMessage = truncate(callErr.Error(), 2000)
	}
	_ = s.repo.CreateAIAuditLog(&repository.AIAuditLog{
		UserID:              userID,
		ConversationID:      conversationID,
		RequestID:           requestID,
		Provider:            provider,
		Model:               model,
		Status:              status,
		DurationMS:          usage.DurationMS,
		InputTokens:         usage.InputTokens,
		OutputTokens:        usage.OutputTokens,
		TotalTokens:         usage.TotalTokens,
		InputPreview:        truncate(requestPayload, 1000),
		OutputPreview:       truncate(output, 1000),
		RequestPayloadJSON:  defaultString(requestPayload, "{}"),
		ResponsePayloadJSON: defaultString(responsePayload, "{}"),
		ErrorCode:           errorCode,
		ErrorMessage:        errorMessage,
		CreatedAt:           time.Now(),
	})
}

func (s *Service) cleanupExpiredAuditLogs() {
	cfg, err := s.configs.Load()
	if err != nil || !cfg.Audit.Enabled || cfg.Audit.RetentionDays <= 0 {
		return
	}
	before := time.Now().AddDate(0, 0, -cfg.Audit.RetentionDays)
	_ = s.repo.DeleteAIAuditLogsBefore(before)
}

func (s *Service) cleanupExpiredRetryJobs() {
	cfg, err := s.configs.Load()
	if err != nil || cfg.Audit.RetentionDays <= 0 {
		return
	}
	before := time.Now().AddDate(0, 0, -cfg.Audit.RetentionDays)
	if err := s.repo.DeleteAIRetryJobsBefore([]string{"completed", "exhausted"}, before); err == nil {
		s.refreshRetryMetrics()
	}
}

func (s *Service) EnqueueRetryJob(userID, conversationID uint64, lastErr error) error {
	err := s.repo.CreateAIRetryJob(&repository.AIRetryJob{
		UserID:         userID,
		ConversationID: conversationID,
		Status:         "pending",
		AttemptCount:   0,
		NextAttemptAt:  time.Now().Add(2 * time.Second),
		LastError:      truncate(errorText(lastErr), 2000),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})
	if err == nil {
		s.refreshRetryMetrics()
	}
	return err
}

func (s *Service) processRetryJobs(ctx context.Context) {
	if s.replyFn == nil {
		return
	}
	jobs, err := s.repo.ListPendingAIRetryJobs(20, time.Now())
	if err != nil {
		return
	}
	for _, job := range jobs {
		reply, err := s.GenerateReply(ctx, job.UserID, job.ConversationID)
		if err == nil {
			emitErr := s.replyFn(job.ConversationID, reply.BotUserID, "text", reply.Content)
			if emitErr == nil {
				_ = s.repo.UpdateAIRetryJob(job.ID, map[string]any{
					"status": "completed",
				})
				continue
			}
			err = emitErr
		}
		nextAttempt := job.AttemptCount + 1
		if nextAttempt >= 3 {
			botUserID, botErr := s.BotUserID()
			if botErr == nil && botUserID > 0 {
				_ = s.replyFn(job.ConversationID, botUserID, "system", "AI 助手暂时不可用，请稍后再试")
			}
			_ = s.repo.UpdateAIRetryJob(job.ID, map[string]any{
				"status":        "exhausted",
				"attempt_count": nextAttempt,
				"last_error":    truncate(errorText(err), 2000),
			})
			continue
		}
		backoff := time.Duration(nextAttempt*nextAttempt) * time.Second
		_ = s.repo.UpdateAIRetryJob(job.ID, map[string]any{
			"attempt_count":   nextAttempt,
			"next_attempt_at": time.Now().Add(backoff),
			"last_error":      truncate(errorText(err), 2000),
		})
	}
	s.refreshRetryMetrics()
}

func (s *Service) ListAuditLogs(query contracts.ListAIAuditLogsQuery) ([]contracts.AIAuditLogDTO, error) {
	rows, err := s.repo.ListAIAuditLogs(query.Limit, strings.TrimSpace(query.Status), strings.TrimSpace(query.Provider), strings.TrimSpace(query.Model), query.UserID, query.ConversationID)
	if err != nil {
		return nil, err
	}
	result := make([]contracts.AIAuditLogDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, auditLogDTO(&row))
	}
	return result, nil
}

func (s *Service) GetAuditLog(id uint64) (*contracts.AIAuditLogDetailDTO, error) {
	row, err := s.repo.FindAIAuditLogByID(id)
	if err != nil {
		return nil, err
	}
	dto := &contracts.AIAuditLogDetailDTO{
		AIAuditLogDTO:       auditLogDTO(row),
		RequestPayloadJSON:  row.RequestPayloadJSON,
		ResponsePayloadJSON: row.ResponsePayloadJSON,
	}
	return dto, nil
}

func (s *Service) ListRetryJobs(query contracts.ListAIRetryJobsQuery) ([]contracts.AIRetryJobDTO, error) {
	rows, err := s.repo.ListAIRetryJobs(query.Limit, strings.TrimSpace(query.Status))
	if err != nil {
		return nil, err
	}
	result := make([]contracts.AIRetryJobDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, retryJobDTO(&row))
	}
	return result, nil
}

func (s *Service) GetRetryJob(id uint64) (*contracts.AIRetryJobDetailDTO, error) {
	row, err := s.repo.FindAIRetryJobByID(id)
	if err != nil {
		return nil, err
	}
	return &contracts.AIRetryJobDetailDTO{
		AIRetryJobDTO: retryJobDTO(row),
	}, nil
}

func (s *Service) RetryJobNow(id uint64) error {
	if _, err := s.repo.FindAIRetryJobByID(id); err != nil {
		return err
	}
	err := s.repo.UpdateAIRetryJob(id, retryResetUpdates(time.Now()))
	if err == nil {
		s.refreshRetryMetrics()
	}
	return err
}

func (s *Service) RetryJobs(ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	err := s.repo.UpdateAIRetryJobs(ids, retryResetUpdates(time.Now()))
	if err == nil {
		s.refreshRetryMetrics()
	}
	return err
}

func (s *Service) CleanupRetryJobs(statuses []string) error {
	err := s.repo.DeleteAIRetryJobsByStatuses(statuses)
	if err == nil {
		s.refreshRetryMetrics()
	}
	return err
}

func (s *Service) refreshRetryMetrics() {
	counts, err := s.repo.CountAIRetryJobsByStatus([]string{"pending", "completed", "exhausted"})
	if err != nil {
		return
	}
	for _, status := range []string{"pending", "completed", "exhausted"} {
		aiRetryJobsGauge.WithLabelValues(status).Set(float64(counts[status]))
	}
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func auditLogDTO(row *repository.AIAuditLog) contracts.AIAuditLogDTO {
	return contracts.AIAuditLogDTO{
		ID:             row.ID,
		UserID:         row.UserID,
		ConversationID: row.ConversationID,
		RequestID:      row.RequestID,
		Provider:       row.Provider,
		Model:          row.Model,
		Status:         row.Status,
		DurationMS:     row.DurationMS,
		InputTokens:    row.InputTokens,
		OutputTokens:   row.OutputTokens,
		TotalTokens:    row.TotalTokens,
		InputPreview:   row.InputPreview,
		OutputPreview:  row.OutputPreview,
		ErrorCode:      row.ErrorCode,
		ErrorMessage:   row.ErrorMessage,
		CreatedAt:      row.CreatedAt.Format(time.RFC3339),
	}
}

func retryJobDTO(row *repository.AIRetryJob) contracts.AIRetryJobDTO {
	return contracts.AIRetryJobDTO{
		ID:             row.ID,
		UserID:         row.UserID,
		ConversationID: row.ConversationID,
		Status:         row.Status,
		AttemptCount:   row.AttemptCount,
		NextAttemptAt:  row.NextAttemptAt.Format(time.RFC3339),
		LastError:      row.LastError,
		CreatedAt:      row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      row.UpdatedAt.Format(time.RFC3339),
	}
}

func retryResetUpdates(nextAttemptAt time.Time) map[string]any {
	return map[string]any{
		"status":          "pending",
		"attempt_count":   0,
		"next_attempt_at": nextAttemptAt,
		"last_error":      "",
	}
}
