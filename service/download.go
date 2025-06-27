package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/ark-eternal/database"
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

	cids, err := database.QueryFileInfo(s.db, userAddress, fileName, database.StatusCompleted)
	if err != nil {
		return err
	}

	return s.fetchFileByCIDS(c, cids)
}

func (s *Service) fetchFileByRootCID(c *gin.Context) error {
	rootCID := c.Param("cid")
	if rootCID == "" {
		return fmt.Errorf("root CID is required")
	}

	cids, err := database.QueryCIDsByRoot(s.db, rootCID, database.StatusCompleted)
	if err != nil {
		return fmt.Errorf("failed to get file data for CID %s: %v", rootCID, err)
	}

	return s.fetchFileByCIDS(c, cids)
}

func (s *Service) fetchFileByCIDS(c *gin.Context, cids []string) error {
	// Create HTTP client
	client := &http.Client{}

	// Create a context to manage cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel to collect results
	type downloadResult struct {
		index int
		data  []byte
		err   error
	}
	results := make(chan downloadResult, len(cids))

	// WaitGroup to wait for all downloads to complete
	var wg sync.WaitGroup

	// Semaphore to limit concurrency
	concurrencyLimit := 10
	semaphore := make(chan struct{}, concurrencyLimit)

	for i, cidString := range cids {
		wg.Add(1)
		go func(ctx context.Context, index int, cid string) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				// Exit if context is canceled
				return
			}

			downloadURL := fmt.Sprintf("%s/piece/%s", s.serviceURL, cid)

			// Create the GET request
			req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
			if err != nil {
				results <- downloadResult{index: index, err: fmt.Errorf("failed to create request for CID %s: %v", cid, err)}
				cancel() // Cancel all other tasks
				return
			}

			// Send the request
			resp, err := client.Do(req)
			if err != nil {
				results <- downloadResult{index: index, err: fmt.Errorf("failed to download piece %s: %v", cid, err)}
				cancel() // Cancel all other tasks
				return
			}
			defer resp.Body.Close()

			// Check response status
			if resp.StatusCode != http.StatusOK {
				results <- downloadResult{index: index, err: fmt.Errorf("failed to download piece %s: status code %d", cid, resp.StatusCode)}
				cancel() // Cancel all other tasks
				return
			}

			// Read the response body
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				results <- downloadResult{index: index, err: fmt.Errorf("failed to read piece %s: %v", cid, err)}
				cancel() // Cancel all other tasks
				return
			}

			results <- downloadResult{index: index, data: data}
		}(ctx, i, cidString)
	}

	// Close the results channel once all downloads are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and write results in order
	downloadedPieces := make([][]byte, len(cids))
	for result := range results {
		if result.err != nil {
			return result.err
		}
		downloadedPieces[result.index] = result.data
	}
	c.Header("Content-Type", "text/html")

	// Write all pieces to the client in order
	for _, piece := range downloadedPieces {
		_, err := c.Writer.Write(piece)
		if err != nil {
			return fmt.Errorf("failed to write piece to client: %v", err)
		}
	}

	return nil
}
