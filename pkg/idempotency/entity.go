package idempotency

import (
	"time"
)

// RequestID структура для хранения идентификаторов запросов
type RequestID struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	Resource   string    `json:"resource" gorm:"index"`    // Тип ресурса (например, "order")
	ResourceID uint      `json:"resource_id" gorm:"index"` // ID ресурса, если есть
	UserID     uint      `json:"user_id" gorm:"index"`     // ID пользователя, который сделал запрос
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at" gorm:"index"` // Время истечения хранения ID запроса
}
