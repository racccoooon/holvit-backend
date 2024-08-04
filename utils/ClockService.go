package utils

import "time"

type ClockService interface {
	Now() time.Time
}

func NewClockService() ClockService {
	return &ClockServiceImpl{}
}

type ClockServiceImpl struct{}

func (c *ClockServiceImpl) Now() time.Time {
	return time.Now()
}
