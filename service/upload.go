package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/filecoin-project/go-commp-utils/nonffi"
	commcid "github.com/filecoin-project/go-fil-commcid"
	commp "github.com/filecoin-project/go-fil-commp-hashhash"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/ark-eternal/database"
)

const chunkSize = 10 << 20

type uploadRequest struct {
	UserAddress string `json:"user_address"`
	FileName    string `json:"file_name"`
	ResourceURL string `json:"resource_url"`
}

func (s *Service) uploadFile(c *gin.Context) error {
	ur := &uploadRequest{}
	if err := c.ShouldBindJSON(ur); err != nil {
		return fmt.Errorf("failed to bind JSON: %w", err)
	}

	content, err := downloadContent(s.ctx, ur.ResourceURL)
	if err != nil {
		return fmt.Errorf("failed to download content: %w", err)
	}

	fileSize := int64(len(content))
	type rootSetInfo struct {
		pieces     []abi.PieceInfo
		subrootStr string
	}
	rootSets := []rootSetInfo{}
	rootSets = append(rootSets, rootSetInfo{
		pieces:     make([]abi.PieceInfo, 0),
		subrootStr: "",
	})
	rootSize := uint64(0)
	maxRootSize, err := abi.RegisteredSealProof_StackedDrg64GiBV1_1.SectorSize()
	if err != nil {
		return fmt.Errorf("failed to get sector size: %v", err)
	}

	jwtToken, err := createJWTToken(s.serviceName, s.privateKey)
	if err != nil {
		return fmt.Errorf("failed to create JWT token: %v", err)
	}

	client := &http.Client{}
	chunkCids := make([]string, 0)
	for idx := int64(0); idx < fileSize; idx += chunkSize {

		end := min(idx+chunkSize, fileSize)
		// Prepare the piece
		n := end - idx
		chunkReader := bytes.NewReader(content[idx:end])
		commP, paddedPieceSize, commpDigest, err := preparePiece(chunkReader)
		if err != nil {
			return fmt.Errorf("failed to prepare piece: %v", err)
		}

		// Prepare the request data

		checkData := map[string]any{
			"name": "sha2-256-trunc254-padded",
			"hash": hex.EncodeToString(commpDigest),
			"size": n,
		}

		reqData := map[string]any{
			"check": checkData,
		}

		reqBody, err := json.Marshal(reqData)
		if err != nil {
			return fmt.Errorf("failed to marshal request data: %v", err)
		}

		// Upload the piece
		err = uploadOnePiece(client, s.serviceURL, reqBody, jwtToken, chunkReader, int64(n))
		if err != nil {
			return fmt.Errorf("failed to upload piece: %v", err)
		}

		slog.Info("Piece uploaded successfully", "cid", commP.String())

		chunkCids = append(chunkCids, commP.String())

		if rootSize+paddedPieceSize > uint64(maxRootSize) {
			rootSets = append(rootSets, rootSetInfo{
				pieces:     make([]abi.PieceInfo, 0),
				subrootStr: "",
			})
			rootSize = 0
		}
		rootSize += paddedPieceSize
		rootSets[len(rootSets)-1].pieces = append(rootSets[len(rootSets)-1].pieces, abi.PieceInfo{Size: abi.PaddedPieceSize(paddedPieceSize), PieceCID: commP})
		rootSets[len(rootSets)-1].subrootStr = fmt.Sprintf("%s+%s", rootSets[len(rootSets)-1].subrootStr, commP)

	}

	for _, rootSet := range rootSets {
		pieceSize := uint64(0)
		for _, piece := range rootSet.pieces {
			pieceSize += uint64(piece.Size)
		}

		root, err := nonffi.GenerateUnsealedCID(abi.RegisteredSealProof_StackedDrg64GiBV1_1, rootSet.pieces)
		if err != nil {
			return fmt.Errorf("failed to generate unsealed CID: %v", err)
		}

		if err := database.InsertData(s.db, ur.UserAddress, ur.FileName, pieceSize, s.proofSetID, root.String(), chunkCids); err != nil {
			return fmt.Errorf("failed to insert data into database: %v", err)
		}
	}

	return nil
}

func uploadOnePiece(client *http.Client, serviceURL string, reqBody []byte, jwtToken string, r io.ReadSeeker, pieceSize int64) error {
	req, err := http.NewRequest("POST", serviceURL+"/pdp/piece", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Piece already exists, get the pieceCID from the response
		var respData map[string]string
		err = json.NewDecoder(resp.Body).Decode(&respData)
		if err != nil {
			return fmt.Errorf("failed to parse response: %v", err)
		}
		pieceCID := respData["pieceCID"]
		fmt.Printf("Piece already exists with CID: %s\n", pieceCID)

		return nil
	case http.StatusCreated:
		// Get the upload URL from the Location header
		uploadURL := resp.Header.Get("Location")
		if uploadURL == "" {
			return fmt.Errorf("server did not provide upload URL in Location header")
		}

		// Upload the piece data via PUT
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek file: %v", err)
		}
		uploadReq, err := http.NewRequest("PUT", serviceURL+uploadURL, r)
		if err != nil {
			return fmt.Errorf("failed to create upload request: %v", err)
		}
		// Set the Content-Length header
		uploadReq.ContentLength = pieceSize
		// Set the Content-Type header
		uploadReq.Header.Set("Content-Type", "application/octet-stream")

		uploadResp, err := client.Do(uploadReq)
		if err != nil {
			return fmt.Errorf("failed to upload piece data: %v", err)
		}
		defer uploadResp.Body.Close()

		if uploadResp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(uploadResp.Body)
			return fmt.Errorf("upload failed with status code %d: %s", uploadResp.StatusCode, string(body))
		}

		return nil
	default:
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status code %d: %s", resp.StatusCode, string(body))
	}
}

func preparePiece(r io.ReadSeeker) (cid.Cid, uint64, []byte, error) {
	// Create commp calculator
	cp := &commp.Calc{}

	// Copy data into commp calculator
	_, err := io.Copy(cp, r)
	if err != nil {
		return cid.Undef, 0, nil, fmt.Errorf("failed to read input file: %v", err)
	}

	// Finalize digest
	digest, paddedPieceSize, err := cp.Digest()
	if err != nil {
		return cid.Undef, 0, nil, fmt.Errorf("failed to compute digest: %v", err)
	}

	// Convert digest to CID
	pieceCIDComputed, err := commcid.DataCommitmentV1ToCID(digest)
	if err != nil {
		return cid.Undef, 0, nil, fmt.Errorf("failed to compute piece CID: %v", err)
	}

	// now compute sha256
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return cid.Undef, 0, nil, fmt.Errorf("failed to seek file: %v", err)
	}

	h := sha256.New()
	_, err = io.Copy(h, r)
	if err != nil {
		return cid.Undef, 0, nil, fmt.Errorf("failed to read input file: %v", err)
	}

	return pieceCIDComputed, paddedPieceSize, digest, nil
}

func downloadContent(ctx context.Context, resourceURL string) ([]byte, error) {
	slog.Info("Downloading content from resource URL", "url", resourceURL)
	// 1. Create Chromedp context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, chromedp.DefaultExecAllocatorOptions[:]...)
	defer allocCancel()

	chromeCtx, chromeCancel := chromedp.NewContext(allocCtx)
	defer chromeCancel()

	// 2. Use Chromedp to fetch rendered HTML
	var htmlContent string
	err := chromedp.Run(chromeCtx,
		chromedp.Navigate(resourceURL),
		chromedp.WaitVisible("body", chromedp.ByQuery), // Wait for page load
		chromedp.OuterHTML("html", &htmlContent),       // Get full HTML
	)
	if err != nil {
		return nil, fmt.Errorf("failed to render page with chromedp: %v", err)
	}

	// 4. Use goquery to parse HTML (since Colly doesn't have ParseHTML)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML with goquery: %v", err)
	}

	// 5. Save HTML to file
	html, err := doc.Html()
	if err != nil {
		return nil, fmt.Errorf("failed to get HTML content: %v", err)
	}

	slog.Info("Content downloaded successfully", "length", len(html))
	return []byte(html), nil
}
