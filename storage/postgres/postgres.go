package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"gateway/model"
	"gateway/storage"
)

// Storage PostgreSQL存储实现
type Storage struct {
	db *sql.DB
}

// New 创建PostgreSQL存储
func New(cfg *storage.Config) (*Storage, error) {
	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	if cfg.MaxConns > 0 {
		db.SetMaxOpenConns(cfg.MaxConns)
	}
	if cfg.MaxIdle > 0 {
		db.SetMaxIdleConns(cfg.MaxIdle)
	}
	if cfg.Lifetime > 0 {
		db.SetConnMaxLifetime(cfg.Lifetime)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &Storage{db: db}, nil
}

// Migrate 执行数据库迁移
func (s *Storage) Migrate(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS request_records (
		id VARCHAR(64) PRIMARY KEY,
		request_id VARCHAR(64) NOT NULL,
		provider VARCHAR(32) NOT NULL,
		model VARCHAR(64) NOT NULL,
		user_id VARCHAR(64) DEFAULT '',
		api_key_id VARCHAR(64) DEFAULT '',
		endpoint VARCHAR(128) NOT NULL,
		method VARCHAR(16) NOT NULL,
		status_code INT NOT NULL,
		prompt_tokens INT DEFAULT 0,
		completion_tokens INT DEFAULT 0,
		total_tokens INT DEFAULT 0,
		latency_ms BIGINT DEFAULT 0,
		error_message TEXT,
		request_body TEXT,
		response_body TEXT,
		client_ip VARCHAR(64) DEFAULT '',
		user_agent VARCHAR(256) DEFAULT '',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_request_records_provider ON request_records(provider);
	CREATE INDEX IF NOT EXISTS idx_request_records_model ON request_records(model);
	CREATE INDEX IF NOT EXISTS idx_request_records_user_id ON request_records(user_id);
	CREATE INDEX IF NOT EXISTS idx_request_records_api_key_id ON request_records(api_key_id);
	CREATE INDEX IF NOT EXISTS idx_request_records_created_at ON request_records(created_at);
	CREATE INDEX IF NOT EXISTS idx_request_records_status_code ON request_records(status_code);

	CREATE TABLE IF NOT EXISTS api_keys (
		id VARCHAR(64) PRIMARY KEY,
		name VARCHAR(128) NOT NULL,
		key_hash VARCHAR(128) NOT NULL UNIQUE,
		key_prefix VARCHAR(16) NOT NULL,
		user_id VARCHAR(64) DEFAULT '',
		status SMALLINT DEFAULT 1,
		rate_limit INT DEFAULT 0,
		daily_limit INT DEFAULT 0,
		monthly_limit INT DEFAULT 0,
		allowed_models TEXT,
		allowed_providers TEXT,
		metadata JSONB,
		expires_at TIMESTAMP,
		last_used_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
	CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
	CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);

	CREATE TABLE IF NOT EXISTS api_key_usage (
		id BIGSERIAL PRIMARY KEY,
		api_key_id VARCHAR(64) NOT NULL,
		date DATE NOT NULL,
		request_count INT DEFAULT 0,
		token_count INT DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(api_key_id, date)
	);

	CREATE INDEX IF NOT EXISTS idx_api_key_usage_api_key_id ON api_key_usage(api_key_id);
	CREATE INDEX IF NOT EXISTS idx_api_key_usage_date ON api_key_usage(date);
	`

	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// SaveRequest 保存请求记录
func (s *Storage) SaveRequest(ctx context.Context, record *model.RequestRecord) error {
	query := `
		INSERT INTO request_records (
			id, request_id, provider, model, user_id, api_key_id,
			endpoint, method, status_code, prompt_tokens, completion_tokens,
			total_tokens, latency_ms, error_message, request_body, response_body,
			client_ip, user_agent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`

	_, err := s.db.ExecContext(ctx, query,
		record.ID,
		record.RequestID,
		record.Provider,
		record.Model,
		record.UserID,
		record.APIKeyID,
		record.Endpoint,
		record.Method,
		record.StatusCode,
		record.PromptTokens,
		record.CompletionTokens,
		record.TotalTokens,
		record.Latency.Milliseconds(),
		record.ErrorMessage,
		record.RequestBody,
		record.ResponseBody,
		record.ClientIP,
		record.UserAgent,
		record.CreatedAt,
	)

	return err
}

// QueryRequests 查询请求记录
func (s *Storage) QueryRequests(ctx context.Context, filter *storage.QueryFilter) ([]*model.RequestRecord, error) {
	query := `SELECT id, request_id, provider, model, user_id, api_key_id,
		endpoint, method, status_code, prompt_tokens, completion_tokens,
		total_tokens, latency_ms, error_message, client_ip, user_agent, created_at
		FROM request_records WHERE 1=1`

	args := []interface{}{}
	argIdx := 1

	if filter.Provider != "" {
		query += fmt.Sprintf(" AND provider = $%d", argIdx)
		args = append(args, filter.Provider)
		argIdx++
	}
	if filter.Model != "" {
		query += fmt.Sprintf(" AND model = $%d", argIdx)
		args = append(args, filter.Model)
		argIdx++
	}
	if filter.UserID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, filter.UserID)
		argIdx++
	}
	if !filter.StartTime.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, filter.StartTime)
		argIdx++
	}
	if !filter.EndTime.IsZero() {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, filter.EndTime)
		argIdx++
	}
	if filter.Status > 0 {
		query += fmt.Sprintf(" AND status_code = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
		argIdx++
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*model.RequestRecord
	for rows.Next() {
		var r model.RequestRecord
		var latencyMs int64
		err := rows.Scan(
			&r.ID, &r.RequestID, &r.Provider, &r.Model, &r.UserID, &r.APIKeyID,
			&r.Endpoint, &r.Method, &r.StatusCode, &r.PromptTokens, &r.CompletionTokens,
			&r.TotalTokens, &latencyMs, &r.ErrorMessage, &r.ClientIP, &r.UserAgent, &r.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		r.Latency = time.Duration(latencyMs) * time.Millisecond
		records = append(records, &r)
	}

	return records, rows.Err()
}

// GetStats 获取统计数据
func (s *Storage) GetStats(ctx context.Context, filter *storage.StatsFilter) (*storage.Stats, error) {
	query := `SELECT 
		COUNT(*) as total,
		SUM(CASE WHEN status_code = 200 THEN 1 ELSE 0 END) as success,
		SUM(CASE WHEN status_code != 200 THEN 1 ELSE 0 END) as failed,
		COALESCE(SUM(total_tokens), 0) as total_tokens,
		COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
		COALESCE(SUM(completion_tokens), 0) as completion_tokens,
		COALESCE(AVG(latency_ms), 0) as avg_latency
		FROM request_records WHERE 1=1`

	args := []interface{}{}
	argIdx := 1

	if filter.Provider != "" {
		query += fmt.Sprintf(" AND provider = $%d", argIdx)
		args = append(args, filter.Provider)
		argIdx++
	}
	if filter.Model != "" {
		query += fmt.Sprintf(" AND model = $%d", argIdx)
		args = append(args, filter.Model)
		argIdx++
	}
	if filter.UserID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, filter.UserID)
		argIdx++
	}
	if !filter.StartTime.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, filter.StartTime)
		argIdx++
	}
	if !filter.EndTime.IsZero() {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, filter.EndTime)
		argIdx++
	}

	var stats storage.Stats
	var avgLatencyMs float64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalRequests,
		&stats.SuccessRequests,
		&stats.FailedRequests,
		&stats.TotalTokens,
		&stats.PromptTokens,
		&stats.CompletionTokens,
		&avgLatencyMs,
	)
	if err != nil {
		return nil, err
	}
	stats.AvgLatency = time.Duration(avgLatencyMs) * time.Millisecond

	// 分组统计
	if filter.GroupBy != "" {
		stats.Groups, err = s.getGroupStats(ctx, filter)
		if err != nil {
			return nil, err
		}
	}

	return &stats, nil
}

func (s *Storage) getGroupStats(ctx context.Context, filter *storage.StatsFilter) ([]storage.StatsGroup, error) {
	var groupColumn string
	switch filter.GroupBy {
	case "provider":
		groupColumn = "provider"
	case "model":
		groupColumn = "model"
	case "user":
		groupColumn = "user_id"
	case "hour":
		groupColumn = "date_trunc('hour', created_at)"
	case "day":
		groupColumn = "date_trunc('day', created_at)"
	default:
		return nil, nil
	}

	query := fmt.Sprintf(`SELECT 
		%s::text as group_key,
		COUNT(*) as total,
		SUM(CASE WHEN status_code = 200 THEN 1 ELSE 0 END) as success,
		SUM(CASE WHEN status_code != 200 THEN 1 ELSE 0 END) as failed,
		COALESCE(SUM(total_tokens), 0) as total_tokens,
		COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
		COALESCE(SUM(completion_tokens), 0) as completion_tokens,
		COALESCE(AVG(latency_ms), 0) as avg_latency
		FROM request_records WHERE 1=1`, groupColumn)

	args := []interface{}{}
	argIdx := 1

	if filter.Provider != "" {
		query += fmt.Sprintf(" AND provider = $%d", argIdx)
		args = append(args, filter.Provider)
		argIdx++
	}
	if filter.Model != "" {
		query += fmt.Sprintf(" AND model = $%d", argIdx)
		args = append(args, filter.Model)
		argIdx++
	}
	if !filter.StartTime.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, filter.StartTime)
		argIdx++
	}
	if !filter.EndTime.IsZero() {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, filter.EndTime)
		argIdx++
	}

	query += fmt.Sprintf(" GROUP BY %s ORDER BY total DESC LIMIT 100", groupColumn)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []storage.StatsGroup
	for rows.Next() {
		var g storage.StatsGroup
		var avgLatencyMs float64
		err := rows.Scan(
			&g.Key,
			&g.TotalRequests,
			&g.SuccessRequests,
			&g.FailedRequests,
			&g.TotalTokens,
			&g.PromptTokens,
			&g.CompletionTokens,
			&avgLatencyMs,
		)
		if err != nil {
			return nil, err
		}
		g.AvgLatency = time.Duration(avgLatencyMs) * time.Millisecond
		groups = append(groups, g)
	}

	return groups, rows.Err()
}

// Close 关闭数据库连接
func (s *Storage) Close() error {
	return s.db.Close()
}

// HealthCheck 健康检查
func (s *Storage) HealthCheck(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// DB 返回原始数据库连接（用于高级操作）
func (s *Storage) DB() *sql.DB {
	return s.db
}
