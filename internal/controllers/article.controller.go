package controllers

import (
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ArticleController struct {
	repo repository.ArticleRepository
}

func NewArticleController(repo repository.ArticleRepository) *ArticleController {
	return &ArticleController{repo: repo}
}

// CreateArticle godoc
// @Summary Create a new article
// @Description Create an article with the provided data
// @Tags article
// @Accept json
// @Produce json
// @Param article body models.Article true "Article data"
// @Success 201 {object} map[string]interface{} "Article created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 500 {object} map[string]interface{} "Failed to create article"
// @Router /article [post]
func (ac *ArticleController) CreateArticle(c *gin.Context) {
	var article models.Article

	if err := c.ShouldBindJSON(&article); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	if err := ac.repo.Create(&article); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to create article",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Article created successfully",
		"data":    article,
	})
}

// GetAllArticles godoc
// @Summary Get all articles
// @Description Retrieve all articles
// @Tags article
// @Produce json
// @Success 200 {object} map[string]interface{} "Articles retrieved successfully"
// @Failure 500 {object} map[string]interface{} "Failed to retrieve articles"
// @Router /article [get]
func (ac *ArticleController) GetAllArticles(c *gin.Context) {
	articles, err := ac.repo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve articles",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Articles retrieved successfully",
		"data":    articles,
	})
}

// GetArticleByID godoc
// @Summary Get an article by ID
// @Description Retrieve article information by ID
// @Tags article
// @Produce json
// @Param id path int true "Article ID"
// @Success 200 {object} map[string]interface{} "Article retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid article ID"
// @Failure 404 {object} map[string]interface{} "Article not found"
// @Router /article/{id} [get]
func (ac *ArticleController) GetArticleByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid article ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	article, err := ac.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Article not found",
			"error":   "No article exists with the provided ID",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Article retrieved successfully",
		"data":    article,
	})
}

// UpdateArticle godoc
// @Summary Update an article
// @Description Update article information
// @Tags article
// @Accept json
// @Produce json
// @Param id path int true "Article ID"
// @Param article body models.Article true "Article data"
// @Success 200 {object} map[string]interface{} "Article updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 404 {object} map[string]interface{} "Article not found"
// @Failure 500 {object} map[string]interface{} "Failed to update article"
// @Router /article/{id} [put]
func (ac *ArticleController) UpdateArticle(c *gin.Context) {
	var article models.Article

	if err := c.ShouldBindJSON(&article); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	// Ensure the ID in the path matches the ID in the body
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid article ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}
	article.ID = uint(id)

	_, err = ac.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Article not found",
			"error":   "No article exists with the provided ID",
		})
		return
	}

	if err := ac.repo.Update(&article); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update article",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Article updated successfully",
		"data":    article,
	})
}

// DeleteArticle godoc
// @Summary Delete an article
// @Description Delete article by ID
// @Tags article
// @Produce json
// @Param id path int true "Article ID"
// @Success 200 {object} map[string]interface{} "Article deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid article ID"
// @Failure 404 {object} map[string]interface{} "Article not found"
// @Failure 500 {object} map[string]interface{} "Failed to delete article"
// @Router /article/{id} [delete]
func (ac *ArticleController) DeleteArticle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid article ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	_, err = ac.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Article not found",
			"error":   "No article exists with the provided ID",
		})
		return
	}

	if err := ac.repo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to delete article",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Article deleted successfully",
		"data":    nil,
	})
}
