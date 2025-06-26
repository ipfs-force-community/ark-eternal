package service

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/ark-eternal/database"
)

const (
	KB = 1 << 10
	MB = 1 << 20
	GB = 1 << 30
	TB = 1 << 40
)

type FileInfo struct {
	Name       string `json:"file_name"`
	Size       string `json:"size"`
	UploadTime string `json:"upload_time"`
	Status     string `json:"status"`
}

func (s *Service) listFiles(c *gin.Context) (any, error) {
	userAddress := c.Query("user_address")
	if userAddress == "" {
		return nil, fmt.Errorf("user_address is required")
	}

	files, err := database.ListFiles(s.db, userAddress)
	if err != nil {
		return nil, err
	}

	var fileInfos []FileInfo
	for _, file := range files {
		fileInfos = append(fileInfos, FileInfo{
			Name:       file.FileName,
			Size:       humanReadableSize(file.Size),
			UploadTime: file.CreatedAt.Format("2006-01-02 15:04"),
			Status:     string(file.Status),
		})

	}
	return fileInfos, nil
}

func humanReadableSize(size uint64) string {
	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/float64(TB))
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
