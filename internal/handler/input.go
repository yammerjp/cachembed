package handler

import (
	"fmt"
	"strings"
)

// processInput は入力を文字列のスライスに変換します
func processInput(input interface{}) ([]string, error) {
	switch v := input.(type) {
	case string:
		if v == "" {
			return nil, fmt.Errorf("empty string input")
		}
		return []string{v}, nil
	case []interface{}:
		// 配列が数値のみで構成されているかチェック
		isNumberArray := true
		for _, item := range v {
			switch item.(type) {
			case float64, int:
				continue
			default:
				isNumberArray = false
				break
			}
		}

		if isNumberArray {
			// 数値配列の場合は1つの入力として扱う
			numbers := make([]string, len(v))
			for i, num := range v {
				switch n := num.(type) {
				case float64:
					numbers[i] = fmt.Sprintf("%.10g", n)
				case int:
					numbers[i] = fmt.Sprintf("%d", n)
				}
			}
			return []string{"[" + strings.Join(numbers, ",") + "]"}, nil
		}

		// それ以外の場合は各要素を個別の入力として処理
		return processArrayInput(v)
	default:
		return nil, fmt.Errorf("invalid input type: got %T", input)
	}
}

// processArrayInput は配列入力を処理します
func processArrayInput(arr []interface{}) ([]string, error) {
	result := make([]string, len(arr))
	for i, item := range arr {
		switch v := item.(type) {
		case string:
			if v == "" {
				return nil, fmt.Errorf("empty string in input array at index %d", i)
			}
			result[i] = v
		case float64:
			result[i] = fmt.Sprintf("%.10g", v)
		case int:
			result[i] = fmt.Sprintf("%d", v)
		case []interface{}:
			// ネストされた数値配列の場合
			numbers := make([]string, len(v))
			for j, num := range v {
				switch n := num.(type) {
				case float64:
					numbers[j] = fmt.Sprintf("%.10g", n)
				case int:
					numbers[j] = fmt.Sprintf("%d", n)
				default:
					return nil, fmt.Errorf("invalid number type in array at index %d,%d: got %T", i, j, num)
				}
			}
			result[i] = "[" + strings.Join(numbers, ",") + "]"
		default:
			return nil, fmt.Errorf("invalid input type at index %d: got %T", i, item)
		}
	}
	return result, nil
}
