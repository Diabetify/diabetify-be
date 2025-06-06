package controllers

import (
	"bytes"
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"io"
	"net/http"
	"strconv"
	"strings"

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
// @Accept json,multipart/form-data
// @Produce json
// @Param article body models.Article true "Article data"
// @Param file formData file false "Image file (optional)"
// @Success 201 {object} map[string]interface{} "Article created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 500 {object} map[string]interface{} "Failed to create article"
// @Router /article [post]
func (ac *ArticleController) CreateArticle(c *gin.Context) {
	contentType := c.GetHeader("Content-Type")
	var article models.Article

	if contentType == "application/json" {
		if err := c.ShouldBindJSON(&article); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Invalid request data",
				"error":   err.Error(),
			})
			return
		}
	} else if strings.Contains(contentType, "multipart/form-data") {
		if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Failed to parse form data",
				"error":   err.Error(),
			})
			return
		}

		article.Title = c.PostForm("title")
		article.Description = c.PostForm("description")
		article.Content = c.PostForm("content")
		article.Author = c.PostForm("author")
		article.Category = c.PostForm("category")
		article.Tags = c.PostForm("tags")
		isPublishedStr := c.PostForm("is_published")
		article.IsPublished = isPublishedStr == "true"

		file, header, err := c.Request.FormFile("file")

		if err == nil && file != nil {
			defer file.Close()

			mimeType := header.Header.Get("Content-Type")
			if !isValidImageMimeType(mimeType) {
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  "error",
					"message": "Invalid image type",
					"error":   "Only image files are allowed (JPEG, PNG, GIF, WebP)",
				})
				return
			}

			buffer := bytes.NewBuffer(nil)
			if _, err := io.Copy(buffer, file); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"status":  "error",
					"message": "Failed to read image data",
					"error":   err.Error(),
				})
				return
			}

			article.ImageData = buffer.Bytes()
			article.ImageMimeType = mimeType
			article.HasImage = true

		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Unsupported content type",
			"error":   "Use application/json or multipart/form-data",
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
	// articles, err := ac.repo.FindAll()
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{
	// 		"status":  "error",
	// 		"message": "Failed to retrieve articles",
	// 		"error":   err.Error(),
	// 	})
	// 	return
	// }

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Articles retrieved successfully",
		"data":    []models.Article{}, // Replace with actual articles
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

// GetArticleImage godoc
// @Summary Get article image
// @Description Retrieve the image for an article
// @Tags article
// @Produce image/jpeg,image/png,image/gif,image/webp
// @Param id path int true "Article ID"
// @Success 200 {file} binary "The image file"
// @Failure 400 {object} map[string]interface{} "Invalid article ID"
// @Failure 404 {object} map[string]interface{} "Article or image not found"
// @Router /article/{id}/image [get]
func (ac *ArticleController) GetArticleImage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid article ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	imageData, mimeType, err := ac.repo.GetImage(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Image not found",
			"error":   err.Error(),
		})
		return
	}

	c.Data(http.StatusOK, mimeType, imageData)
}

// UpdateArticle godoc
// @Summary Update an article
// @Description Update article information
// @Tags article
// @Accept json,multipart/form-data
// @Produce json
// @Param id path int true "Article ID"
// @Param article body models.Article false "Article data (JSON)"
// @Param article formData object false "Article data (form)"
// @Param file formData file false "Image file (optional)"
// @Param delete_image formData boolean false "Delete existing image if true"
// @Success 200 {object} map[string]interface{} "Article updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 404 {object} map[string]interface{} "Article not found"
// @Failure 500 {object} map[string]interface{} "Failed to update article"
// @Router /article/{id} [put]
func (ac *ArticleController) UpdateArticle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid article ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	existingArticle, err := ac.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Article not found",
			"error":   "No article exists with the provided ID",
		})
		return
	}

	contentType := c.GetHeader("Content-Type")
	var article models.Article

	if contentType == "application/json" {
		if err := c.ShouldBindJSON(&article); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Invalid request data",
				"error":   err.Error(),
			})
			return
		}
	} else if c.ContentType() == "multipart/form-data" {
		if err := c.ShouldBind(&article); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Invalid form data",
				"error":   err.Error(),
			})
			return
		}

		deleteImage := c.PostForm("delete_image") == "true"
		if deleteImage && existingArticle.HasImage {
			if err := ac.repo.DeleteImage(uint(id)); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"status":  "error",
					"message": "Failed to delete existing image",
					"error":   err.Error(),
				})
				return
			}
			article.HasImage = false
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Unsupported content type",
			"error":   "Use application/json or multipart/form-data",
		})
		return
	}

	article.ID = uint(id)

	if existingArticle.HasImage && !article.HasImage {
		article.HasImage = existingArticle.HasImage
	}

	if err := ac.repo.Update(&article); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update article",
			"error":   err.Error(),
		})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err == nil && file != nil {
		defer file.Close()

		mimeType := header.Header.Get("Content-Type")
		if !isValidImageMimeType(mimeType) {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Invalid image type",
				"error":   "Only image files are allowed (JPEG, PNG, GIF, WebP)",
			})
			return
		}

		buffer := bytes.NewBuffer(nil)
		if _, err := io.Copy(buffer, file); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to read image data",
				"error":   err.Error(),
			})
			return
		}

		if err := ac.repo.SaveImage(article.ID, buffer.Bytes(), mimeType); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Article updated but failed to save image",
				"error":   err.Error(),
				"data":    article,
			})
			return
		}

		article.HasImage = true
		if err := ac.repo.Update(&article); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to update article image status",
				"error":   err.Error(),
			})
			return
		}
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

func isValidImageMimeType(mimeType string) bool {
	validMimeTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	return validMimeTypes[mimeType]
}
