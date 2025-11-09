package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/TIANLI0/LayerKit/config"
	"github.com/TIANLI0/LayerKit/model"
	"github.com/TIANLI0/LayerKit/utils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisService struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisService(cfg *config.RedisConfig) *RedisService {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return &RedisService{
		client: client,
		ttl:    cfg.TTL,
	}
}

func (s *RedisService) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// GetLayerResult 从缓存获取分层结果
func (s *RedisService) GetLayerResult(ctx context.Context, md5 string) (*model.LayerResult, error) {
	key := "layer:" + md5
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, err
	}

	var result model.LayerResult
	if err := json.Unmarshal(data, &result); err != nil {
		utils.Logger.Error("failed to unmarshal layer result",
			zap.String("md5", md5), zap.Error(err))
		return nil, err
	}

	return &result, nil
}

// SetLayerResult 设置分层结果到缓存
func (s *RedisService) SetLayerResult(ctx context.Context, md5 string, result *model.LayerResult) error {
	key := "layer:" + md5
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, data, s.ttl).Err()
}

func (s *RedisService) Close() error {
	return s.client.Close()
}
