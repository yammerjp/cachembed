package handler

import (
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
)

// EmbeddingInput は埋め込み対象の入力を表すインターフェース
type EmbeddingInput interface {
	// ToAPIInput はupstreamに送信する形式に変換する
	ToAPIInput() interface{}
	// Length は入力の要素数を返す
	Length() int
	// PickInputs は指定されたインデックスの入力のみを抽出する
	PickInputs(indexes []int) (EmbeddingInput, error)
	// InputHashes は入力のハッシュを返す
	InputHashes() ([]string, error)
}

// SingleStringInput は単一の文字列入力
type SingleStringInput struct {
	Value string
}

func (s SingleStringInput) ToAPIInput() interface{} {
	return s.Value
}

func (s SingleStringInput) Length() int {
	return 1
}

func (s SingleStringInput) PickInputs(indexes []int) (EmbeddingInput, error) {
	if len(indexes) != 1 || indexes[0] != 0 {
		return nil, fmt.Errorf("invalid indexes for single string input")
	}
	return s, nil
}

func (s SingleStringInput) InputHashes() ([]string, error) {
	hash := sha1.Sum([]byte(s.Value))
	return []string{hex.EncodeToString(hash[:])}, nil
}

// MultiStringInput は複数の文字列入力
type MultiStringInput struct {
	Values []string
}

func (m MultiStringInput) ToAPIInput() interface{} {
	return m.Values
}

func (m MultiStringInput) Length() int {
	return len(m.Values)
}

func (m MultiStringInput) PickInputs(indexes []int) (EmbeddingInput, error) {
	picked := make([]string, len(indexes))
	for i, idx := range indexes {
		if idx < 0 || idx >= len(m.Values) {
			return nil, fmt.Errorf("index out of range: %d", idx)
		}
		picked[i] = m.Values[idx]
	}
	return MultiStringInput{Values: picked}, nil
}

func (m MultiStringInput) InputHashes() ([]string, error) {
	hashes := make([]string, len(m.Values))
	for i, str := range m.Values {
		hash := sha1.Sum([]byte(str))
		hashes[i] = hex.EncodeToString(hash[:])
	}
	return hashes, nil
}

// SingleNumberArrayInput は単一の数値配列入力
type SingleNumberArrayInput struct {
	Values []float64
}

func (s SingleNumberArrayInput) ToAPIInput() interface{} {
	return s.Values
}

func (s SingleNumberArrayInput) Length() int {
	return 1
}

func (s SingleNumberArrayInput) PickInputs(indexes []int) (EmbeddingInput, error) {
	if len(indexes) != 1 || indexes[0] != 0 {
		return nil, fmt.Errorf("invalid indexes for single number array input")
	}
	return s, nil
}

func (s SingleNumberArrayInput) InputHashes() ([]string, error) {
	hash := numArrSha1(s.Values)
	return []string{hex.EncodeToString(hash[:])}, nil
}

// MultiNumberArrayInput は複数の数値配列入力
type MultiNumberArrayInput struct {
	Values [][]float64
}

func (m MultiNumberArrayInput) ToAPIInput() interface{} {
	return m.Values
}

func (m MultiNumberArrayInput) Length() int {
	return len(m.Values)
}

func (m MultiNumberArrayInput) PickInputs(indexes []int) (EmbeddingInput, error) {
	picked := make([][]float64, len(indexes))
	for i, idx := range indexes {
		if idx < 0 || idx >= len(m.Values) {
			return nil, fmt.Errorf("index out of range: %d", idx)
		}
		picked[i] = m.Values[idx]
	}
	return MultiNumberArrayInput{Values: picked}, nil
}

func (m MultiNumberArrayInput) InputHashes() ([]string, error) {
	hashes := make([]string, len(m.Values))
	for i, nums := range m.Values {
		hash := numArrSha1(nums)
		hashes[i] = hex.EncodeToString(hash[:])
	}
	return hashes, nil
}

// numArrSha1をembedding_input.goに移動
func numArrSha1(nums []float64) [20]byte {
	numsBytes := make([]byte, len(nums)*8)
	for i, num := range nums {
		binary.BigEndian.PutUint64(numsBytes[i*8:], math.Float64bits(num))
	}
	return sha1.Sum(numsBytes)
}

// ParseEmbeddingInput はJSONデータから適切なEmbeddingInput実装を生成する
func ParseEmbeddingInput(data json.RawMessage) (EmbeddingInput, error) {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	switch v := raw.(type) {
	case string:
		return SingleStringInput{Value: v}, nil

	case []interface{}:
		if len(v) == 0 {
			return nil, fmt.Errorf("input array must not be empty")
		}

		switch first := v[0].(type) {
		case string:
			// 文字列配列
			strings := make([]string, len(v))
			for i, item := range v {
				str, ok := item.(string)
				if !ok {
					return nil, fmt.Errorf("invalid element type in string array at index %d: got %T", i, item)
				}
				strings[i] = str
			}
			return MultiStringInput{Values: strings}, nil

		case float64:
			// 単一の数値配列
			numbers := make([]float64, len(v))
			for i, item := range v {
				num, ok := item.(float64)
				if !ok {
					return nil, fmt.Errorf("invalid element type in number array at index %d: got %T", i, item)
				}
				numbers[i] = num
			}
			return SingleNumberArrayInput{Values: numbers}, nil

		case []interface{}:
			// 複数の数値配列
			arrays := make([][]float64, len(v))
			for i, arr := range v {
				subArr, ok := arr.([]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid element type at index %d: got %T", i, arr)
				}
				numbers := make([]float64, len(subArr))
				for j, item := range subArr {
					num, ok := item.(float64)
					if !ok {
						return nil, fmt.Errorf("invalid element type at index [%d][%d]: got %T", i, j, item)
					}
					numbers[j] = num
				}
				arrays[i] = numbers
			}
			return MultiNumberArrayInput{Values: arrays}, nil

		default:
			return nil, fmt.Errorf("invalid array element type: got %T", first)
		}

	default:
		return nil, fmt.Errorf("invalid input type: got %T", raw)
	}
}
