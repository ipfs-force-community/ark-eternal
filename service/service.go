package service

import (
	"crypto/ecdsa"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Service struct {
	db          *sql.DB
	privateKey  *ecdsa.PrivateKey
	proofSetID  int
	serviceURL  string
	serviceName string
}

func NewService(
	db *sql.DB,
	privateKey *ecdsa.PrivateKey,
	proofSetID int,
	serviceURL string,
	serviceName string,
) *Service {
	return &Service{
		db:          db,
		privateKey:  privateKey,
		proofSetID:  proofSetID,
		serviceURL:  serviceURL,
		serviceName: serviceName,
	}
}

func (s *Service) Run(port int32) error {
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.POST("/upload", func(c *gin.Context) {
		if err := s.uploadFile(c); err != nil {
			slog.Error("failed to upload file", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "file uploaded successfully",
		})
	})

	r.GET("/download", func(c *gin.Context) {
		if err := s.downloadFile(c); err != nil {
			slog.Error("failed to download file", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
	})

	r.GET("/files", func(c *gin.Context) {
		files, err := s.listFiles(c)
		if err != nil {
			slog.Error("failed to list files", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, files)
	})

	r.Run(fmt.Sprintf(":%d", port))
	return nil
}
