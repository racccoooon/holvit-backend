package repositories

import "github.com/google/uuid"

type BaseFilter struct {
	Id         uuid.UUID
	PagingInfo PagingInfo
}

type PagingInfo struct {
	PageSize   int
	PageNumber int
}
