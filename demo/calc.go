package demo

import "fmt"

// Add 两个整数相加
func Add(a, b int) int {
	return a + b
}

// Subtract 两个整数相减
func Subtract(a, b int) int {
	return a - b
}

// Divide 两个数相除，除数为0时返回错误
func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil
}
