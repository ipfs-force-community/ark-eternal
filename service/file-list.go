package service

import (
	"fmt"

	"github.com/asamuj/ark-eternal/database"
	"github.com/gin-gonic/gin"
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
