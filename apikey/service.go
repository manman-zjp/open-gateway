package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"gateway/model"
)

// Store API Key 存储接口
type Store interface {
	Create(ctx context.Context, key *model.APIKey) error
	Get(ctx context.Context, id string) (*model.APIKey, error)
	GetByKeyHash(ctx context.Context, keyHash string) (*model.APIKey, error)
	Update(ctx context.Context, key *model.APIKey) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter *model.APIKeyListRequest) ([]*model.APIKey, int, error)
	UpdateLastUsed(ctx context.Context, id string) error
	IncrementUsage(ctx context.Context, id string, tokens int64) error
	ConsumeTokens(ctx context.Context, id string, tokens int64) error
	ResetDailyTokens(ctx context.Context, id string) error
	GetUsage(ctx context.Context, id string, startDate, endDate time.Time) ([]*model.APIKeyUsage, error)
}

// Service API Key 服务
type Service struct {
	store Store
	cache sync.Map // 简单内存缓存
}

// NewService 创建 API Key 服务
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create 创建 API Key
func (s *Service) Create(ctx context.Context, req *model.APIKeyCreateRequest) (*model.APIKeyCreateResponse, error) {
	// 生成随机 key
	rawKey, err := generateKey(32)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	// 格式化 key: sk-xxxxxxxxxxxxxx
	fullKey := "sk-" + rawKey
	keyPrefix := fullKey[:10] + "..."
	keyHash := hashKey(fullKey)

	now := time.Now()
	apiKey := &model.APIKey{
		ID:               generateID(),
		Name:             req.Name,
		KeyHash:          keyHash,
		KeyPrefix:        keyPrefix,
		UserID:           req.UserID,
		Status:           model.APIKeyStatusActive,
		RateLimit:        req.RateLimit,
		DailyLimit:       req.DailyLimit,
		MonthlyLimit:     req.MonthlyLimit,
		TokenQuota:       req.TokenQuota,
		DailyTokenQuota:  req.DailyTokenQuota,
		UsedTokens:       0,
		UsedDailyTokens:  0,
		AllowedModels:    req.AllowedModels,
		AllowedProviders: req.AllowedProviders,
		Metadata:         req.Metadata,
		ExpiresAt:        req.ExpiresAt,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.store.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}

	return &model.APIKeyCreateResponse{
		ID:        apiKey.ID,
		Name:      apiKey.Name,
		Key:       fullKey, // 只在创建时返回
		KeyPrefix: keyPrefix,
		CreatedAt: apiKey.CreatedAt,
	}, nil
}

// Get 获取 API Key
func (s *Service) Get(ctx context.Context, id string) (*model.APIKey, error) {
	return s.store.Get(ctx, id)
}

// Validate 验证 API Key
func (s *Service) Validate(ctx context.Context, rawKey string) (*model.APIKey, error) {
	keyHash := hashKey(rawKey)

	// 尝试从缓存获取
	if cached, ok := s.cache.Load(keyHash); ok {
		apiKey := cached.(*model.APIKey)
		if apiKey.IsActive() {
			// 异步更新最后使用时间
			go func() { _ = s.store.UpdateLastUsed(context.Background(), apiKey.ID) }()
			return apiKey, nil
		}
		s.cache.Delete(keyHash)
	}

	// 从存储获取
	apiKey, err := s.store.GetByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, fmt.Errorf("invalid api key")
	}

	if !apiKey.IsActive() {
		return nil, fmt.Errorf("api key is not active")
	}

	// 缓存有效的 key
	s.cache.Store(keyHash, apiKey)

	// 异步更新最后使用时间
	go func() { _ = s.store.UpdateLastUsed(context.Background(), apiKey.ID) }()

	return apiKey, nil
}

// Update 更新 API Key
func (s *Service) Update(ctx context.Context, id string, req *model.APIKeyUpdateRequest) (*model.APIKey, error) {
	apiKey, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		apiKey.Name = *req.Name
	}
	if req.Status != nil {
		apiKey.Status = *req.Status
	}
	if req.RateLimit != nil {
		apiKey.RateLimit = *req.RateLimit
	}
	if req.DailyLimit != nil {
		apiKey.DailyLimit = *req.DailyLimit
	}
	if req.MonthlyLimit != nil {
		apiKey.MonthlyLimit = *req.MonthlyLimit
	}
	if req.TokenQuota != nil {
		apiKey.TokenQuota = *req.TokenQuota
	}
	if req.DailyTokenQuota != nil {
		apiKey.DailyTokenQuota = *req.DailyTokenQuota
	}
	if req.ResetUsedTokens {
		apiKey.UsedTokens = 0
		apiKey.UsedDailyTokens = 0
	}
	if req.AllowedModels != nil {
		apiKey.AllowedModels = req.AllowedModels
	}
	if req.AllowedProviders != nil {
		apiKey.AllowedProviders = req.AllowedProviders
	}
	if req.Metadata != nil {
		apiKey.Metadata = req.Metadata
	}
	if req.ExpiresAt != nil {
		apiKey.ExpiresAt = req.ExpiresAt
	}
	apiKey.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, apiKey); err != nil {
		return nil, err
	}

	// 清除缓存
	s.cache.Delete(apiKey.KeyHash)

	return apiKey, nil
}

// Delete 删除 API Key
func (s *Service) Delete(ctx context.Context, id string) error {
	apiKey, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}

	// 清除缓存
	s.cache.Delete(apiKey.KeyHash)

	return nil
}

// List 列出 API Keys
func (s *Service) List(ctx context.Context, req *model.APIKeyListRequest) (*model.APIKeyListResponse, error) {
	keys, total, err := s.store.List(ctx, req)
	if err != nil {
		return nil, err
	}

	return &model.APIKeyListResponse{
		Total: total,
		Data:  keys,
	}, nil
}

// IncrementUsage 增加使用量
func (s *Service) IncrementUsage(ctx context.Context, id string, tokens int64) error {
	return s.store.IncrementUsage(ctx, id, tokens)
}

// ConsumeTokens 消耗 Token 额度
func (s *Service) ConsumeTokens(ctx context.Context, id string, tokens int64) error {
	// 先消耗 Token
	if err := s.store.ConsumeTokens(ctx, id, tokens); err != nil {
		return err
	}
	// 记录使用量
	return s.store.IncrementUsage(ctx, id, tokens)
}

// CheckAndResetDailyQuota 检查并重置日额度
func (s *Service) CheckAndResetDailyQuota(ctx context.Context, apiKey *model.APIKey) {
	// 检查是否需要重置日额度
	if apiKey.QuotaResetAt == nil {
		return
	}
	now := time.Now()
	if now.After(*apiKey.QuotaResetAt) {
		// 重置日额度
		_ = s.store.ResetDailyTokens(ctx, apiKey.ID)
		// 设置下一次重置时间（第二天凌晨）
		nextReset := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		apiKey.QuotaResetAt = &nextReset
		apiKey.UsedDailyTokens = 0
	}
}

// GetUsage 获取使用统计
func (s *Service) GetUsage(ctx context.Context, id string, days int) ([]*model.APIKeyUsage, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)
	return s.store.GetUsage(ctx, id, startDate, endDate)
}

// InvalidateCache 使缓存失效
func (s *Service) InvalidateCache(keyHash string) {
	s.cache.Delete(keyHash)
}

// 辅助函数

func generateKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func generateID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// MemoryStore 内存存储实现（用于测试/开发）
type MemoryStore struct {
	keys  map[string]*model.APIKey
	mu    sync.RWMutex
	usage map[string]map[string]*model.APIKeyUsage // keyID -> date -> usage
}

// NewMemoryStore 创建内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		keys:  make(map[string]*model.APIKey),
		usage: make(map[string]map[string]*model.APIKeyUsage),
	}
}

func (s *MemoryStore) Create(ctx context.Context, key *model.APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[key.ID] = key
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (*model.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if key, ok := s.keys[id]; ok {
		return key, nil
	}
	return nil, fmt.Errorf("api key not found")
}

func (s *MemoryStore) GetByKeyHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, key := range s.keys {
		if key.KeyHash == keyHash {
			return key, nil
		}
	}
	return nil, fmt.Errorf("api key not found")
}

func (s *MemoryStore) Update(ctx context.Context, key *model.APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[key.ID] = key
	return nil
}

func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keys, id)
	return nil
}

func (s *MemoryStore) List(ctx context.Context, filter *model.APIKeyListRequest) ([]*model.APIKey, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.APIKey
	for _, key := range s.keys {
		if filter.UserID != "" && key.UserID != filter.UserID {
			continue
		}
		if filter.Status != nil && int(key.Status) != *filter.Status {
			continue
		}
		result = append(result, key)
	}

	total := len(result)

	// 分页
	start := filter.Offset
	if start > len(result) {
		start = len(result)
	}
	end := start + filter.Limit
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], total, nil
}

func (s *MemoryStore) UpdateLastUsed(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key, ok := s.keys[id]; ok {
		now := time.Now()
		key.LastUsedAt = &now
	}
	return nil
}

func (s *MemoryStore) IncrementUsage(ctx context.Context, id string, tokens int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.usage[id] == nil {
		s.usage[id] = make(map[string]*model.APIKeyUsage)
	}

	today := time.Now().Format("2006-01-02")
	if s.usage[id][today] == nil {
		s.usage[id][today] = &model.APIKeyUsage{
			KeyID: id,
			Date:  time.Now().Truncate(24 * time.Hour),
		}
	}

	s.usage[id][today].RequestCount++
	s.usage[id][today].TokenCount += tokens

	return nil
}

func (s *MemoryStore) ConsumeTokens(ctx context.Context, id string, tokens int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, ok := s.keys[id]
	if !ok {
		return fmt.Errorf("api key not found")
	}

	key.UsedTokens += tokens
	key.UsedDailyTokens += tokens
	return nil
}

func (s *MemoryStore) ResetDailyTokens(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if key, ok := s.keys[id]; ok {
		key.UsedDailyTokens = 0
		now := time.Now()
		nextReset := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		key.QuotaResetAt = &nextReset
	}
	return nil
}

func (s *MemoryStore) GetUsage(ctx context.Context, id string, startDate, endDate time.Time) ([]*model.APIKeyUsage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.APIKeyUsage
	if usageMap, ok := s.usage[id]; ok {
		for _, usage := range usageMap {
			if !usage.Date.Before(startDate) && !usage.Date.After(endDate) {
				result = append(result, usage)
			}
		}
	}

	return result, nil
}

// SQLStore SQL 存储实现
type SQLStore struct {
	db *sql.DB
}

// NewSQLStore 创建 SQL 存储
func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) Create(ctx context.Context, key *model.APIKey) error {
	allowedModels, _ := json.Marshal(key.AllowedModels)
	allowedProviders, _ := json.Marshal(key.AllowedProviders)
	metadata, _ := json.Marshal(key.Metadata)

	query := `INSERT INTO api_keys (
		id, name, key_hash, key_prefix, user_id, status,
		rate_limit, daily_limit, monthly_limit,
		allowed_models, allowed_providers, metadata,
		expires_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
		key.ID, key.Name, key.KeyHash, key.KeyPrefix, key.UserID, key.Status,
		key.RateLimit, key.DailyLimit, key.MonthlyLimit,
		string(allowedModels), string(allowedProviders), string(metadata),
		key.ExpiresAt, key.CreatedAt, key.UpdatedAt,
	)
	return err
}

func (s *SQLStore) Get(ctx context.Context, id string) (*model.APIKey, error) {
	query := `SELECT id, name, key_hash, key_prefix, user_id, status,
		rate_limit, daily_limit, monthly_limit,
		allowed_models, allowed_providers, metadata,
		expires_at, last_used_at, created_at, updated_at
		FROM api_keys WHERE id = ?`

	var key model.APIKey
	var allowedModels, allowedProviders, metadata string
	var expiresAt, lastUsedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&key.ID, &key.Name, &key.KeyHash, &key.KeyPrefix, &key.UserID, &key.Status,
		&key.RateLimit, &key.DailyLimit, &key.MonthlyLimit,
		&allowedModels, &allowedProviders, &metadata,
		&expiresAt, &lastUsedAt, &key.CreatedAt, &key.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(allowedModels), &key.AllowedModels)
	_ = json.Unmarshal([]byte(allowedProviders), &key.AllowedProviders)
	_ = json.Unmarshal([]byte(metadata), &key.Metadata)

	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}

	return &key, nil
}

func (s *SQLStore) GetByKeyHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	query := `SELECT id, name, key_hash, key_prefix, user_id, status,
		rate_limit, daily_limit, monthly_limit,
		allowed_models, allowed_providers, metadata,
		expires_at, last_used_at, created_at, updated_at
		FROM api_keys WHERE key_hash = ?`

	var key model.APIKey
	var allowedModels, allowedProviders, metadata string
	var expiresAt, lastUsedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, keyHash).Scan(
		&key.ID, &key.Name, &key.KeyHash, &key.KeyPrefix, &key.UserID, &key.Status,
		&key.RateLimit, &key.DailyLimit, &key.MonthlyLimit,
		&allowedModels, &allowedProviders, &metadata,
		&expiresAt, &lastUsedAt, &key.CreatedAt, &key.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(allowedModels), &key.AllowedModels)
	_ = json.Unmarshal([]byte(allowedProviders), &key.AllowedProviders)
	_ = json.Unmarshal([]byte(metadata), &key.Metadata)

	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}

	return &key, nil
}

func (s *SQLStore) Update(ctx context.Context, key *model.APIKey) error {
	allowedModels, _ := json.Marshal(key.AllowedModels)
	allowedProviders, _ := json.Marshal(key.AllowedProviders)
	metadata, _ := json.Marshal(key.Metadata)

	query := `UPDATE api_keys SET 
		name = ?, status = ?, rate_limit = ?, daily_limit = ?, monthly_limit = ?,
		allowed_models = ?, allowed_providers = ?, metadata = ?,
		expires_at = ?, updated_at = ?
		WHERE id = ?`

	_, err := s.db.ExecContext(ctx, query,
		key.Name, key.Status, key.RateLimit, key.DailyLimit, key.MonthlyLimit,
		string(allowedModels), string(allowedProviders), string(metadata),
		key.ExpiresAt, key.UpdatedAt, key.ID,
	)
	return err
}

func (s *SQLStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM api_keys WHERE id = ?", id)
	return err
}

func (s *SQLStore) List(ctx context.Context, filter *model.APIKeyListRequest) ([]*model.APIKey, int, error) {
	countQuery := "SELECT COUNT(*) FROM api_keys WHERE 1=1"
	query := `SELECT id, name, key_hash, key_prefix, user_id, status,
		rate_limit, daily_limit, monthly_limit,
		allowed_models, allowed_providers, metadata,
		expires_at, last_used_at, created_at, updated_at
		FROM api_keys WHERE 1=1`

	args := []interface{}{}

	if filter.UserID != "" {
		countQuery += " AND user_id = ?"
		query += " AND user_id = ?"
		args = append(args, filter.UserID)
	}
	if filter.Status != nil {
		countQuery += " AND status = ?"
		query += " AND status = ?"
		args = append(args, *filter.Status)
	}

	// 获取总数
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, filter.Limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var keys []*model.APIKey
	for rows.Next() {
		var key model.APIKey
		var allowedModels, allowedProviders, metadata string
		var expiresAt, lastUsedAt sql.NullTime

		err := rows.Scan(
			&key.ID, &key.Name, &key.KeyHash, &key.KeyPrefix, &key.UserID, &key.Status,
			&key.RateLimit, &key.DailyLimit, &key.MonthlyLimit,
			&allowedModels, &allowedProviders, &metadata,
			&expiresAt, &lastUsedAt, &key.CreatedAt, &key.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		_ = json.Unmarshal([]byte(allowedModels), &key.AllowedModels)
		_ = json.Unmarshal([]byte(allowedProviders), &key.AllowedProviders)
		_ = json.Unmarshal([]byte(metadata), &key.Metadata)

		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}
		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Time
		}

		keys = append(keys, &key)
	}

	return keys, total, nil
}

func (s *SQLStore) UpdateLastUsed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE api_keys SET last_used_at = ? WHERE id = ?", time.Now(), id)
	return err
}

func (s *SQLStore) IncrementUsage(ctx context.Context, id string, tokens int64) error {
	today := time.Now().Format("2006-01-02")
	query := `INSERT INTO api_key_usage (api_key_id, date, request_count, token_count)
		VALUES (?, ?, 1, ?)
		ON DUPLICATE KEY UPDATE request_count = request_count + 1, token_count = token_count + ?`
	_, err := s.db.ExecContext(ctx, query, id, today, tokens, tokens)
	return err
}

func (s *SQLStore) ConsumeTokens(ctx context.Context, id string, tokens int64) error {
	query := `UPDATE api_keys SET 
		used_tokens = used_tokens + ?,
		used_daily_tokens = used_daily_tokens + ?
		WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, tokens, tokens, id)
	return err
}

func (s *SQLStore) ResetDailyTokens(ctx context.Context, id string) error {
	now := time.Now()
	nextReset := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	query := `UPDATE api_keys SET used_daily_tokens = 0, quota_reset_at = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, nextReset, id)
	return err
}

func (s *SQLStore) GetUsage(ctx context.Context, id string, startDate, endDate time.Time) ([]*model.APIKeyUsage, error) {
	query := `SELECT api_key_id, date, request_count, token_count 
		FROM api_key_usage 
		WHERE api_key_id = ? AND date >= ? AND date <= ?
		ORDER BY date DESC`

	rows, err := s.db.QueryContext(ctx, query, id, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usages []*model.APIKeyUsage
	for rows.Next() {
		var u model.APIKeyUsage
		if err := rows.Scan(&u.KeyID, &u.Date, &u.RequestCount, &u.TokenCount); err != nil {
			return nil, err
		}
		usages = append(usages, &u)
	}

	return usages, nil
}

// ExtractAPIKey 从请求中提取 API Key
func ExtractAPIKey(authHeader string) string {
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return authHeader
}
