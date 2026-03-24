package admin

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LeeJc02/WeHi/backend/internal/config"
	"github.com/LeeJc02/WeHi/backend/internal/platform/httpx"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"
)

type monitorSnapshot struct {
	Overview contracts.MonitorOverview
	Point    contracts.MonitorPoint
}

type MonitorService struct {
	client    *http.Client
	targets   map[string]string
	mu        sync.RWMutex
	snapshots []monitorSnapshot
}

func NewMonitorService(cfg config.Config) *MonitorService {
	return &MonitorService{
		client: &http.Client{Timeout: 2 * time.Second},
		targets: map[string]string{
			"auth":     cfg.AuthServiceURL,
			"api":      cfg.APIServiceURL,
			"realtime": cfg.RealtimeServiceURL,
		},
		snapshots: make([]monitorSnapshot, 0, 720),
	}
}

func (s *MonitorService) Start(ctx context.Context) {
	s.pollOnce()
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.pollOnce()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *MonitorService) Overview() *contracts.MonitorOverview {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.snapshots) == 0 {
		return &contracts.MonitorOverview{}
	}
	copyValue := s.snapshots[len(s.snapshots)-1].Overview
	return &copyValue
}

func (s *MonitorService) Timeseries() *contracts.MonitorTimeseries {
	s.mu.RLock()
	defer s.mu.RUnlock()
	points := make([]contracts.MonitorPoint, 0, len(s.snapshots))
	for _, snapshot := range s.snapshots {
		points = append(points, snapshot.Point)
	}
	return &contracts.MonitorTimeseries{Points: points}
}

func (s *MonitorService) pollOnce() {
	now := time.Now().UTC()
	statuses := make([]contracts.MonitorServiceStatus, 0, len(s.targets))
	var totalRequests float64
	var clientErrors float64
	var serverErrors float64
	var latencySum float64
	var latencyCount float64
	var wsConnections float64
	var aiRetryPending float64
	var aiRetryCompleted float64
	var aiRetryExhausted float64

	for name, baseURL := range s.targets {
		healthy, errMessage := true, ""
		if err := s.checkHealth(baseURL); err != nil {
			healthy = false
			errMessage = err.Error()
		}
		status := contracts.MonitorServiceStatus{
			Name:      name,
			Healthy:   healthy,
			Status:    map[bool]string{true: "up", false: "down"}[healthy],
			Error:     errMessage,
			CheckedAt: now.Format(time.RFC3339),
		}
		statuses = append(statuses, status)

		metricsBody, err := s.fetchText(baseURL + "/metrics")
		if err != nil {
			continue
		}
		totalRequests += parseMetricSum(metricsBody, httpx.HTTPRequestsTotalMetricName(), nil)
		clientErrors += parseMetricSum(metricsBody, httpx.HTTPRequestsTotalMetricName(), func(labels string) bool {
			return strings.Contains(labels, `status="4`)
		})
		serverErrors += parseMetricSum(metricsBody, httpx.HTTPRequestsTotalMetricName(), func(labels string) bool {
			return strings.Contains(labels, `status="5`)
		})
		latencySum += parseMetricSum(metricsBody, httpx.HTTPRequestDurationMetricName()+"_sum", nil)
		latencyCount += parseMetricSum(metricsBody, httpx.HTTPRequestDurationMetricName()+"_count", nil)
		wsConnections += parseMetricSum(metricsBody, "chat_ws_connections", nil)
		aiRetryPending += parseMetricSum(metricsBody, "chat_ai_retry_jobs", func(labels string) bool {
			return strings.Contains(labels, `status="pending"`)
		})
		aiRetryCompleted += parseMetricSum(metricsBody, "chat_ai_retry_jobs", func(labels string) bool {
			return strings.Contains(labels, `status="completed"`)
		})
		aiRetryExhausted += parseMetricSum(metricsBody, "chat_ai_retry_jobs", func(labels string) bool {
			return strings.Contains(labels, `status="exhausted"`)
		})
	}

	averageLatencyMs := 0.0
	if latencyCount > 0 {
		averageLatencyMs = (latencySum / latencyCount) * 1000
	}
	overview := contracts.MonitorOverview{
		Services:             statuses,
		TotalRequests:        totalRequests,
		ClientErrors:         clientErrors,
		ServerErrors:         serverErrors,
		AverageLatencyMs:     averageLatencyMs,
		WebSocketConnections: wsConnections,
		AIRetryPending:       aiRetryPending,
		AIRetryCompleted:     aiRetryCompleted,
		AIRetryExhausted:     aiRetryExhausted,
		SnapshotAt:           now.Format(time.RFC3339),
	}
	point := contracts.MonitorPoint{
		Timestamp:            now.Format(time.RFC3339),
		TotalRequests:        totalRequests,
		ClientErrors:         clientErrors,
		ServerErrors:         serverErrors,
		AverageLatencyMs:     averageLatencyMs,
		WebSocketConnections: wsConnections,
		AIRetryPending:       aiRetryPending,
		AIRetryCompleted:     aiRetryCompleted,
		AIRetryExhausted:     aiRetryExhausted,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshots = append(s.snapshots, monitorSnapshot{Overview: overview, Point: point})
	if len(s.snapshots) > 720 {
		s.snapshots = s.snapshots[len(s.snapshots)-720:]
	}
}

func (s *MonitorService) checkHealth(baseURL string) error {
	url := baseURL + "/health"
	if strings.Contains(baseURL, ":8082") {
		url = baseURL + "/api/v1/system/ready"
	}
	resp, err := s.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return io.EOF
	}
	return nil
}

func (s *MonitorService) fetchText(url string) (string, error) {
	resp, err := s.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func parseMetricSum(body, metricName string, match func(labels string) bool) float64 {
	lines := strings.Split(body, "\n")
	total := 0.0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.HasPrefix(line, metricName) {
			continue
		}
		labels := ""
		valuePart := ""
		if idx := strings.Index(line, "{"); idx >= 0 {
			end := strings.Index(line, "}")
			if end < 0 {
				continue
			}
			labels = line[idx+1 : end]
			if match != nil && !match(labels) {
				continue
			}
			valuePart = strings.TrimSpace(line[end+1:])
		} else {
			if match != nil && !match("") {
				continue
			}
			valuePart = strings.TrimSpace(strings.TrimPrefix(line, metricName))
		}
		value, err := strconv.ParseFloat(strings.Fields(valuePart)[0], 64)
		if err != nil {
			continue
		}
		total += value
	}
	return total
}
