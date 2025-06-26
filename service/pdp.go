package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

func CreateProofSet(recordKeeper, extraDataHexStr, serviceURL, jwtToken string) (string, error) {
	// Construct the request payload
	requestBody := map[string]string{
		"recordKeeper": recordKeeper,
	}
	if extraDataHexStr != "" {
		requestBody["extraData"] = extraDataHexStr
	}

	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %v", err)
	}

	// Append /pdp/proof-sets to the service URL
	postURL := serviceURL + "/pdp/proof-sets"

	// Create the POST request
	req, err := http.NewRequest("POST", postURL, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read and display the response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}
	bodyString := string(bodyBytes)

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("failed to create proof set, status code %d: %s", resp.StatusCode, bodyString)
	}

	location := resp.Header.Get("Location")
	fmt.Printf("Proof set creation initiated successfully.\n")
	fmt.Printf("Location: %s\n", location)
	fmt.Printf("Response: %s\n", bodyString)

	// Extract the transaction hash from the Location header
	parts := strings.Split(location, "/")
	if len(parts) > 0 {
		txHash := parts[len(parts)-1]
		fmt.Printf("Transaction Hash: %s\n", txHash)
		return location, nil
	} else {
		return "", fmt.Errorf("failed to extract transaction hash from Location header")
	}
}

func GetProofSetCreateStatus(txHash, serviceURL, jwtToken string) error {
	// Ensure txHash starts with '0x'
	if !strings.HasPrefix(txHash, "0x") {
		txHash = "0x" + txHash
	}
	txHash = strings.ToLower(txHash) // Ensure txHash is in lowercase

	// Construct the request URL
	getURL := fmt.Sprintf("%s/pdp/proof-sets/created/%s", serviceURL, txHash)

	// Create the GET request
	req, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read and process the response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode == http.StatusOK {
		// Decode the JSON response
		var response struct {
			CreateMessageHash string  `json:"createMessageHash"`
			ProofsetCreated   bool    `json:"proofsetCreated"`
			Service           string  `json:"service"`
			TxStatus          string  `json:"txStatus"`
			OK                *bool   `json:"ok"`
			ProofSetId        *uint64 `json:"proofSetId,omitempty"`
		}
		err = json.Unmarshal(bodyBytes, &response)
		if err != nil {
			return fmt.Errorf("failed to parse JSON response: %v", err)
		}

		// Display the status
		fmt.Printf("Proof Set Creation Status:\n")
		fmt.Printf("Transaction Hash: %s\n", response.CreateMessageHash)
		fmt.Printf("Transaction Status: %s\n", response.TxStatus)
		if response.OK != nil {
			fmt.Printf("Transaction Successful: %v\n", *response.OK)
		} else {
			fmt.Printf("Transaction Successful: Pending\n")
		}
		fmt.Printf("Proofset Created: %v\n", response.ProofsetCreated)
		if response.ProofSetId != nil {
			fmt.Printf("ProofSet ID: %d\n", *response.ProofSetId)
		}
	} else {
		return fmt.Errorf("failed to get proof set status, status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func AddRoots(extraDataHexStr, serviceURL, jwtToken string, proofSetID int, rootInputs []string) error {
	// Parse the root inputs to construct the request payload
	type SubrootEntry struct {
		SubrootCID string `json:"subrootCid"`
	}

	type AddRootRequest struct {
		RootCID  string         `json:"rootCid"`
		Subroots []SubrootEntry `json:"subroots"`
	}

	var addRootRequests []AddRootRequest

	for _, rootInput := range rootInputs {
		slog.Info("Processing root input", "input", rootInput, "proofSetID", proofSetID)
		// Expected format: rootCID:subrootCID1,subrootCID2,...
		parts := strings.SplitN(rootInput, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid root input format: %s (%d)", rootInput, len(parts))
		}
		rootCID := parts[0]
		subrootsStr := parts[1]
		subrootCIDStrs := strings.Split(subrootsStr, "+")

		if rootCID == "" || len(subrootCIDStrs) == 0 {
			return fmt.Errorf("rootCID and at least one subrootCID are required")
		}

		var subroots []SubrootEntry
		for _, subrootCID := range subrootCIDStrs {
			subroots = append(subroots, SubrootEntry{SubrootCID: subrootCID})
		}

		addRootRequests = append(addRootRequests, AddRootRequest{
			RootCID:  rootCID,
			Subroots: subroots,
		})
	}

	// Construct the full request payload including extraData
	type AddRootsPayload struct {
		Roots     []AddRootRequest `json:"roots"`
		ExtraData *string          `json:"extraData,omitempty"`
	}

	payload := AddRootsPayload{
		Roots: addRootRequests,
	}
	if extraDataHexStr != "" {
		// Pass the validated 0x-prefixed hex string directly
		payload.ExtraData = &extraDataHexStr
	}

	requestBodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %v", err)
	}

	// Construct the POST URL
	postURL := fmt.Sprintf("%s/pdp/proof-sets/%d/roots", serviceURL, proofSetID)

	// Create the POST request
	req, err := http.NewRequest("POST", postURL, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read and display the response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}
	bodyString := string(bodyBytes)

	if resp.StatusCode == http.StatusCreated {
		fmt.Printf("Roots added to proof set ID %d successfully.\n", proofSetID)
		fmt.Printf("Response: %s\n", bodyString)
	} else {
		return fmt.Errorf("failed to add roots, status code %d: %s", resp.StatusCode, bodyString)
	}

	return nil
}
