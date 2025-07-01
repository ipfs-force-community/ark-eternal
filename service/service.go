package service

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ipfs-force-community/ark-eternal/database"
)

type Service struct {
	ctx         context.Context
	srv         *http.Server
	db          *gorm.DB
	privateKey  *ecdsa.PrivateKey
	proofSetID  int
	serviceURL  string
	serviceName string
}

func NewService(
	ctx context.Context,
	db *gorm.DB,
	privateKey *ecdsa.PrivateKey,
	proofSetID int,
	serviceURL string,
	serviceName string,
) *Service {
	s := &Service{
		ctx:         ctx,
		db:          db,
		privateKey:  privateKey,
		proofSetID:  proofSetID,
		serviceURL:  serviceURL,
		serviceName: serviceName,
	}

	r := s.registerRoutes()
	s.srv = &http.Server{
		Addr:    ":8080", // Default address, can be overridden in Start method
		Handler: r,
	}

	return s
}

func (s *Service) registerRoutes() *gin.Engine {
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

	r.GET("/:cid", func(c *gin.Context) {
		if err := s.fetchFileByRootCID(c); err != nil {
			slog.Error("failed to fetch file by root CID", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
	})

	return r
}

func (s *Service) Start(port int32) error {
	s.srv.Addr = fmt.Sprintf(":%d", port)
	slog.Info("starting server", "address", s.srv.Addr)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %v", err)
	}

	return nil
}

func (s *Service) Schedule() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	slog.Info("scheduler started, running every 10 seconds")
	for {
		select {
		case <-s.ctx.Done():
			slog.Info("service context done, stopping scheduler")
			return
		case <-ticker.C:
			if err := s.performScheduledTask(); err != nil {
				slog.Error("failed to perform scheduled task", "error", err)
			}
		}
	}
}

func (s *Service) performScheduledTask() error {
	fileInfos, err := database.QueryPendingInfo(s.db)
	if err != nil {
		return fmt.Errorf("failed to query pending data: %v", err)
	}

	// TODO: If the status remains pending for a long time, it should be marked as failed.

	jwtToken, err := createJWTToken(s.serviceName, s.privateKey)
	if err != nil {
		return fmt.Errorf("failed to create JWT token: %v", err)
	}

	for _, fileInfo := range fileInfos {
		cids := strings.ReplaceAll(fileInfo.CIDs, " ", "+")
		root := fmt.Sprintf("%s:%s", fileInfo.Root, cids)
		if err := AddRoots("", s.serviceURL, jwtToken, fileInfo.ProofSetID, []string{root}); err != nil {
			return fmt.Errorf("failed to add roots to proof set: %v", err)
		}

		if err := database.UpdateFileStatus(s.db, fileInfo.ID, database.StatusCompleted); err != nil {
			return fmt.Errorf("failed to update file info status: %v", err)
		}

		slog.Info("updated file info status to completed", "file_id", fileInfo.ID, "file_name", fileInfo.FileName)
	}

	return nil
}

func (s *Service) Close() error {
	if s.srv != nil {
		if err := s.srv.Shutdown(s.ctx); err != nil {
			return fmt.Errorf("failed to shutdown server: %v", err)
		}
		slog.Info("server shutdown gracefully")
	}

	return nil
}
