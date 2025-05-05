package repository

import (
	"context"
	"diabetify/internal/models"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	articleCacheKeyPrefix = "article:"
	allArticlesCacheKey   = "articles:all"
	cacheExpiration       = 30 * time.Minute
)

type ArticleRepository interface {
	Create(article *models.Article) error
	FindAll() ([]models.Article, error)
	FindByID(id uint) (*models.Article, error)
	Update(article *models.Article) error
	Delete(id uint) error
	InvalidateCache(id uint) error
	InvalidateAllCache() error

	SaveImage(id uint, imageData []byte, mimeType string) error
	GetImage(id uint) ([]byte, string, error)
	DeleteImage(id uint) error
}

type articleRepository struct {
	db    *gorm.DB
	redis *redis.Client
	ctx   context.Context
}

func getCacheKey(id uint) string {
	return fmt.Sprintf("%s%d", articleCacheKeyPrefix, id)
}

func NewArticleRepository(db *gorm.DB) ArticleRepository {
	return &articleRepository{
		db:    db,
		redis: nil,
		ctx:   context.Background(),
	}
}

func NewCachedArticleRepository(db *gorm.DB, redisClient *redis.Client) ArticleRepository {
	return &articleRepository{
		db:    db,
		redis: redisClient,
		ctx:   context.Background(),
	}
}

func (r *articleRepository) Create(article *models.Article) error {
	result := r.db.Create(article)
	if result.Error != nil {
		log.Printf("Error creating article: %v", result.Error)
		return result.Error
	}
	return nil
}

func (r *articleRepository) FindAll() ([]models.Article, error) {
	if r.redis == nil {
		var articles []models.Article
		err := r.db.Find(&articles).Error
		return articles, err
	}

	cachedData, err := r.redis.Get(r.ctx, allArticlesCacheKey).Result()
	if err == nil {
		var articles []models.Article
		if err := json.Unmarshal([]byte(cachedData), &articles); err == nil {
			return articles, nil
		}
	}

	// Cache not found, find in database
	var articles []models.Article
	if err := r.db.Find(&articles).Error; err != nil {
		return nil, err
	}

	// Get articles information from database
	articlesJSON, err := json.Marshal(articles)
	if err == nil {
		if err := r.redis.Set(r.ctx, allArticlesCacheKey, articlesJSON, cacheExpiration).Err(); err != nil {
			log.Printf("Failed to cache all articles: %v", err)
		}
	}

	return articles, nil
}

func (r *articleRepository) FindByID(id uint) (*models.Article, error) {
	if r.redis == nil {
		var article models.Article
		err := r.db.First(&article, id).Error
		if err != nil {
			return nil, err
		}
		return &article, nil
	}

	cacheKey := getCacheKey(id)
	cachedData, err := r.redis.Get(r.ctx, cacheKey).Result()
	if err == nil {
		var article models.Article
		if err := json.Unmarshal([]byte(cachedData), &article); err == nil {
			log.Printf("Cache hit for article ID %d", id)
			return &article, nil
		}
		log.Printf("Failed to unmarshal cached article: %v", err)
	}

	// Cache miss or error, query database
	var article models.Article
	if err := r.db.First(&article, id).Error; err != nil {
		return nil, err
	}

	// Cache the result
	articleJSON, err := json.Marshal(article)
	if err == nil {
		if err := r.redis.Set(r.ctx, cacheKey, articleJSON, cacheExpiration).Err(); err != nil {
			log.Printf("Failed to cache article ID %d: %v", id, err)
		}
	}

	return &article, nil
}

func (r *articleRepository) Update(article *models.Article) error {
	if err := r.db.Save(article).Error; err != nil {
		return err
	}

	if r.redis == nil {
		return nil
	}

	// Update cache
	articleJSON, err := json.Marshal(article)
	if err == nil {
		if err := r.redis.Set(r.ctx, getCacheKey(article.ID), articleJSON, cacheExpiration).Err(); err != nil {
			log.Printf("Failed to update article cache: %v", err)
		}
	}
	_ = r.InvalidateAllCache()

	return nil
}

func (r *articleRepository) Delete(id uint) error {
	if err := r.db.Delete(&models.Article{}, id).Error; err != nil {
		return err
	}
	if r.redis == nil {
		return nil
	}
	_ = r.InvalidateCache(id)
	_ = r.InvalidateAllCache()
	return nil
}

func (r *articleRepository) InvalidateCache(id uint) error {
	if r.redis == nil {
		return nil
	}
	return r.redis.Del(r.ctx, getCacheKey(id)).Err()
}

func (r *articleRepository) InvalidateAllCache() error {
	if r.redis == nil {
		return nil
	}
	return r.redis.Del(r.ctx, allArticlesCacheKey).Err()
}

func (r *articleRepository) SaveImage(id uint, imageData []byte, mimeType string) error {
	var article models.Article
	if err := r.db.First(&article, id).Error; err != nil {
		return fmt.Errorf("article not found: %w", err)
	}

	article.ImageData = imageData
	article.ImageMimeType = mimeType
	article.HasImage = true

	if err := r.db.Save(&article).Error; err != nil {
		log.Printf("Failed to save image for article %d: %v", id, err)
		return fmt.Errorf("failed to save image: %w", err)
	}

	var checkArticle models.Article
	r.db.First(&checkArticle, id)
	// Invalidate cache
	if r.redis != nil {
		_ = r.InvalidateCache(id)
		_ = r.InvalidateAllCache()
	}

	return nil
}

func (r *articleRepository) GetImage(id uint) ([]byte, string, error) {
	log.Printf("GetImage called for article ID: %d", id)

	var article models.Article
	err := r.db.Select("id", "image_data", "image_mime_type", "has_image").
		First(&article, id).Error

	if err != nil {
		return nil, "", fmt.Errorf("article not found: %w", err)
	}
	if !article.HasImage || len(article.ImageData) == 0 {
		return nil, "", fmt.Errorf("no image found for article")
	}
	return article.ImageData, article.ImageMimeType, nil
}

func (r *articleRepository) DeleteImage(id uint) error {
	var article models.Article
	if err := r.db.First(&article, id).Error; err != nil {
		return fmt.Errorf("article not found: %w", err)
	}
	result := r.db.Model(&models.Article{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"image_data":      nil,
			"image_mime_type": "",
			"has_image":       false,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to delete image: %w", result.Error)
	}
	if r.redis != nil {
		_ = r.InvalidateCache(id)
		_ = r.InvalidateAllCache()
	}

	return nil
}
