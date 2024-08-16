package repos

import (
	"github.com/google/uuid"
	"time"
)

type BaseModel struct {
	Id             uuid.UUID
	AuditCreatedAt time.Time
	AuditUpdatedAt time.Time
}
