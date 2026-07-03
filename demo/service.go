package demo

import "errors"

// User 用户结构体
type User struct {
	ID    int
	Name  string
	Email string
}

// UserService 用户服务
type UserService struct {
	users map[int]User
}

// NewUserService 创建用户服务
func NewUserService() *UserService {
	return &UserService{
		users: make(map[int]User),
	}
}

// AddUser 添加用户
func (s *UserService) AddUser(user User) error {
	if user.ID == 0 {
		return errors.New("invalid user ID")
	}
	s.users[user.ID] = user
	return nil
}

// GetUser 获取用户
func (s *UserService) GetUser(id int) (User, error) {
	user, ok := s.users[id]
	if !ok {
		return User{}, errors.New("user not found")
	}
	return user, nil
}

// Calculate 计算两个整数之和
func Calculate(a, b int) int {
	return a + b
}
