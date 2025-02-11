package upstream

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
)

// Client はOpenAI APIクライアントの構造体
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient は新しいClientを作成します
func NewClient(httpClient *http.Client, baseURL string) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

// CreateEmbedding は埋め込みを作成します
func (c *Client) CreateEmbedding(req *EmbeddingRequest, authHeader string) (*EmbeddingResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("failed to decode error response: %w", err)
		}
		return nil, &UpstreamError{
			StatusCode: resp.StatusCode,
			ErrorInfo: struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			}{
				Message: errResp.Error.Message,
				Type:    errResp.Error.Type,
			},
			Usage: errResp.Usage,
		}
	}

	var embedResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &embedResp, nil
}

func (r *EmbeddingRequest) InputHashes() ([]string, error) {
	hashes, err := r.inputHashBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get input hashes: %w", err)
	}

	hashesStr := make([]string, len(hashes))
	for i, hash := range hashes {
		hashesStr[i] = hex.EncodeToString(hash[:])
	}
	return hashesStr, nil
}

func (r *EmbeddingRequest) inputHashBytes() ([][20]byte, error) {
	if r.Input == nil {
		return nil, fmt.Errorf("input is nil")
	}

	if str, ok := r.Input.(string); ok {
		return [][20]byte{sha1.Sum([]byte(str))}, nil
	}

	if nums, ok := r.Input.([]float64); ok {
		if len(nums) == 0 {
			return nil, fmt.Errorf("input array is empty")
		}
		return [][20]byte{numArrSha1(nums)}, nil
	}

	if strs, ok := r.Input.([]string); ok {
		hashes := make([][20]byte, len(strs))
		for i, str := range strs {
			hashes[i] = sha1.Sum([]byte(str))
		}
		return hashes, nil
	}

	if nums, ok := r.Input.([][]float64); ok {
		hashes := make([][20]byte, len(nums))
		for i, num := range nums {
			hashes[i] = numArrSha1(num)
		}
		return hashes, nil
	}

	return nil, fmt.Errorf("unsupported input type: %T", r.Input)
}

func numArrSha1(nums []float64) [20]byte {
	numsBytes := make([]byte, len(nums)*8)
	for i, num := range nums {
		binary.BigEndian.PutUint64(numsBytes[i*8:], math.Float64bits(num))
	}
	return sha1.Sum(numsBytes)
}

func (r *EmbeddingRequest) PickInput(target int) (interface{}, error) {
	if r.Input == nil {
		return nil, fmt.Errorf("input is nil")
	}
	if target < 0 {
		return nil, fmt.Errorf("invalid target: %d", target)
	}
	if nums, ok := r.Input.([]float64); ok {
		if target != 0 {
			return nil, fmt.Errorf("invalid target: %d", target)
		}
		return nums, nil
	}

	if str, ok := r.Input.(string); ok {
		if target != 0 {
			return nil, fmt.Errorf("invalid target: %d", target)
		}
		return str, nil
	}

	if strs, ok := r.Input.([]string); ok {
		if target >= len(strs) {
			return nil, fmt.Errorf("invalid target: %d", target)
		}
		return strs[target], nil
	}

	if nums, ok := r.Input.([][]float64); ok {
		if target >= len(nums) {
			return nil, fmt.Errorf("invalid target: %d", target)
		}
		return nums[target], nil
	}

	return nil, fmt.Errorf("unsupported input type: %T", r.Input)
}

func (r *EmbeddingRequest) PickInputs(targets []int) (interface{}, error) {
	if r.Input == nil {
		return nil, fmt.Errorf("input is nil")
	}

	if strs, ok := r.Input.(string); ok {
		if len(targets) != 1 || targets[0] != 0 {
			return nil, fmt.Errorf("invalid targets: %v", targets)
		}
		return strs, nil
	}

	if nums, ok := r.Input.([]float64); ok {
		if len(targets) != 1 || targets[0] != 0 {
			return nil, fmt.Errorf("invalid targets: %v", targets)
		}
		return nums, nil
	}

	if strs, ok := r.Input.([]string); ok {
		rets := make([]string, len(targets))
		for i, target := range targets {
			if target >= len(strs) {
				return nil, fmt.Errorf("invalid target: %d", target)
			}
			rets[i] = strs[target]
		}
		return rets, nil
	}

	if nums, ok := r.Input.([][]float64); ok {
		rets := make([][]float64, len(targets))
		for i, target := range targets {
			if target >= len(nums) {
				return nil, fmt.Errorf("invalid target: %d", target)
			}
			rets[i] = nums[target]
		}
		return rets, nil
	}

	return nil, fmt.Errorf("unsupported input type: %T", r.Input)
}
