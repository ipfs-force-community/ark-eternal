package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/filecoin-project/go-commp-utils/nonffi"
	commcid "github.com/filecoin-project/go-fil-commcid"
	commp "github.com/filecoin-project/go-fil-commp-hashhash"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/ark-eternal/database"
)

const chunkSize = 10 << 20

type UploadRequest struct {
	UserAddress string `json:"user_address"`
	FileName    string `json:"file_name"`
	ResourceURL string `json:"resource_url"`
}

func (s *Service) uploadFile(c *gin.Context) error {
	ur := &UploadRequest{}
	if err := c.ShouldBindJSON(ur); err != nil {
		return fmt.Errorf("failed to bind JSON: %w", err)
	}

	content, err := downloadContent(ur.ResourceURL)
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

		if err := database.InsertData(s.db, ur.UserAddress, ur.FileName, s.proofSetID, root.String(), chunkCids); err != nil {
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

func downloadContent(resourceURL string) ([]byte, error) {
	slog.Info("Downloading content from resource URL", "url", resourceURL)
	url, err := launcher.New().
		Headless(true).
		NoSandbox(true). // 👈 关键
		Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(url).MustConnect()
	page := browser.MustPage(resourceURL).MustWaitLoad()

	html, err := page.Eval(`() => {
		const docClone = document.cloneNode(true);

		// 处理 <link rel="stylesheet">
		docClone.querySelectorAll('link[rel=stylesheet]').forEach(link => {
			const style = document.createElement('style');
			fetch(link.href)
				.then(resp => resp.text())
				.then(css => style.textContent = css)
				.catch(() => {});
			link.replaceWith(style);
		});

		// 处理 <script src=...>
		docClone.querySelectorAll('script[src]').forEach(script => {
			const newScript = document.createElement('script');
			fetch(script.src)
				.then(resp => resp.text())
				.then(js => newScript.textContent = js)
				.catch(() => {});
			script.replaceWith(newScript);
		});

		// 处理 <img src=...> 等
		const toDataURL = async url => {
			const blob = await fetch(url).then(r => r.blob());
			return new Promise(resolve => {
				const reader = new FileReader();
				reader.onload = () => resolve(reader.result);
				reader.readAsDataURL(blob);
			});
		};

		const promises = [];
		docClone.querySelectorAll('img').forEach(img => {
			if (img.src.startsWith('http')) {
				const p = toDataURL(img.src).then(dataUrl => {
					img.setAttribute('src', dataUrl);
				}).catch(() => {});
				promises.push(p);
			}
		});

		return Promise.all(promises).then(() => '<!DOCTYPE html>\n' + docClone.documentElement.outerHTML);
	}`)

	if err != nil {
		return nil, err
	}

	slog.Info("Content downloaded successfully", "length", len(html.Value.Str()))
	return []byte(html.Value.Str()), nil
}
