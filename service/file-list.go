package service

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/ark-eternal/database"
)

func (s *Service) listFiles(c *gin.Context) (any, error) {
	userAddress := c.Query("user_address")
	if userAddress == "" {
		return nil, fmt.Errorf("user_address is required")
	}

	files, err := database.ListFiles(s.db, userAddress)
	if err != nil {
		return nil, err
	}

	return files, nil
}
