package demo

// 泛型函数
func Max[T int | int64 | float64](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// 变参函数
func SumAll(prefix string, nums ...int) (string, int) {
	total := 0
	for _, n := range nums {
		total += n
	}
	return prefix, total
}

// 通道操作
func Drain(ch <-chan int) []int {
	result := []int{}
	for v := range ch {
		result = append(result, v)
	}
	return result
}

// 带结构体字段的方法
type Stack struct {
	items []int
	max   int
}

func (s *Stack) Push(item int) {
	s.items = append(s.items, item)
	if item > s.max {
		s.max = item
	}
}

func (s *Stack) Pop() (int, bool) {
	if len(s.items) == 0 {
		return 0, false
	}
	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return item, true
}

// 接口定义
type Stringer interface {
	String() string
}
