package idempotency

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Repository интерфейс репозитория для работы с идентификаторами запросов
type Repository interface {
	Create(ctx context.Context, requestID *RequestID) error
	GetByID(ctx context.Context, id string) (*RequestID, error)
	DeleteExpired(ctx context.Context) error
	UpdateResourceID(ctx context.Context, requestID string, resourceID uint) error
}

// Ошибки репозитория
var (
	// ErrRequestIDNotFound ошибка, когда идентификатор запроса не найден
	ErrRequestIDNotFound = errors.New("идентификатор запроса не найден")
	// ErrRequestIDAlreadyExists ошибка, когда идентификатор запроса уже существует
	ErrRequestIDAlreadyExists = errors.New("идентификатор запроса уже существует")
)

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) Repository {
	return &GormRepository{
		db: db,
	}
}

func (r *GormRepository) Create(ctx context.Context, requestID *RequestID) error {
	var count int64
	if err := r.db.WithContext(ctx).Model(&RequestID{}).
		Where("id = ?", requestID.ID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrRequestIDAlreadyExists
	}

	return r.db.WithContext(ctx).Create(requestID).Error
}

func (r *GormRepository) GetByID(ctx context.Context, id string) (*RequestID, error) {
	var requestID RequestID
	result := r.db.WithContext(ctx).First(&requestID, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRequestIDNotFound
		}
		return nil, result.Error
	}
	return &requestID, nil
}

func (r *GormRepository) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&RequestID{}).Error
}

func (r *GormRepository) UpdateResourceID(ctx context.Context, requestID string, resourceID uint) error {
	result := r.db.WithContext(ctx).Model(&RequestID{}).
		Where("id = ?", requestID).
		Update("resource_id", resourceID)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrRequestIDNotFound
	}

	return nil
}
