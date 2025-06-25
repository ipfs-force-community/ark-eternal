package service

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/asamuj/ark-eternal/database"
)

type DownloadFileRequest struct {
	UserAddress string `json:"user_address"`
	FileName    string `json:"file_name"`
}

func (s *Service) downloadFile(c *gin.Context) error {
	userAddress := c.Query("user_address")
	if userAddress == "" {
		return fmt.Errorf("user_address is required")
	}

	fileName := c.Query("file_name")
	if fileName == "" {
		return fmt.Errorf("file_name is required")
	}

	cids, err := database.QueryData(s.db, userAddress, fileName)
	if err != nil {
		return err
	}

	// Create HTTP client
	client := &http.Client{}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))

	// Download each piece and write it to the output file
	for _, cidString := range cids {
		// Create the download URL
		downloadURL := fmt.Sprintf("%s/piece/%s", s.serviceURL, cidString)

		// Create the GET request
		req, err := http.NewRequest("GET", downloadURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request for CID %s: %v", cidString, err)
		}

		// Send the request
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to download piece %s: %v", cidString, err)
		}

		// Check response status
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("failed to download piece %s: status code %d", cidString, resp.StatusCode)
		}
		// Stream the response body to the client
		_, err = io.Copy(c.Writer, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to stream piece %s to client: %v", cidString, err)
		}
	}
	return nil
}
