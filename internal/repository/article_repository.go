package repository

import (
	"diabetify/internal/models"

	"gorm.io/gorm"
)

type ArticleRepository interface {
	Create(article *models.Article) error
	FindAll() ([]models.Article, error)
	FindByID(id uint) (*models.Article, error)
	Update(article *models.Article) error
	Delete(id uint) error
}

type articleRepository struct {
	db *gorm.DB
}

func NewArticleRepository(db *gorm.DB) ArticleRepository {
	return &articleRepository{db}
}

func (r *articleRepository) Create(article *models.Article) error {
	return r.db.Create(article).Error
}

func (r *articleRepository) FindAll() ([]models.Article, error) {
	var articles []models.Article
	err := r.db.Find(&articles).Error
	return articles, err
}

func (r *articleRepository) FindByID(id uint) (*models.Article, error) {
	var article models.Article
	err := r.db.First(&article, id).Error
	if err != nil {
		return nil, err
	}
	return &article, nil
}

func (r *articleRepository) Update(article *models.Article) error {
	return r.db.Save(article).Error
}

func (r *articleRepository) Delete(id uint) error {
	return r.db.Delete(&models.Article{}, id).Error
}
