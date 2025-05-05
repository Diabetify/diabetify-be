package models

import (
	"time"

	"gorm.io/gorm"
)

type Article struct {
	ID           uint           `gorm:"primaryKey" json:"id" form:"id" example:"1"`
	CreatedAt    time.Time      `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt    time.Time      `json:"updated_at" example:"2023-01-01T00:00:00Z"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-" swaggerignore:"true"`
	Title        string         `json:"title" form:"title" example:"Sample Article Title"`
	Description  string         `json:"description" form:"description" example:"This is a sample article description."`
	Content      string         `gorm:"type:text" json:"content" form:"content" example:"Full article content goes here..."`
	Author       string         `json:"author,omitempty" form:"author" example:"John Doe"`
	Category     string         `json:"category,omitempty" form:"category" example:"Health"`
	Tags         string         `json:"tags,omitempty" form:"tags" example:"diabetes,health,nutrition"`
	ReadCount    *int           `json:"read_count,omitempty" gorm:"default:0" example:"42"`
	IsPublished  bool           `json:"is_published" form:"is_published" gorm:"default:true" example:"true"`
	PublishedAt  *time.Time     `json:"published_at,omitempty" example:"2023-01-01T00:00:00Z"`
	ThumbnailURL string         `json:"thumbnail_url,omitempty" example:"https://example.com/image.jpg"`

	ImageData     []byte `gorm:"type:bytea" json:"-"`
	ImageMimeType string `gorm:"size:50" json:"image_mime_type,omitempty" example:"image/jpeg"`
	HasImage      bool   `gorm:"default:false" json:"has_image" example:"true"`
}
