package service

import (
	"crypto/ecdsa"
	"database/sql"
	"fmt"
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
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.POST("/upload", func(c *gin.Context) {
		if err := s.uploadFile(c); err != nil {
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
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
	})

	r.Run(fmt.Sprintf(":%d", port))
	return nil
}
