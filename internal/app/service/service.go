package service

import "time"

type Service struct {
}

func New() *Service {
	return &Service{}
}

func (s *Service) TimeNow() time.Time {
	return time.Now()
}
