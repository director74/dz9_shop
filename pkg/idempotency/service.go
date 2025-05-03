package idempotency

import (
	"context"
	"errors"
	"time"
)

// Ошибки сервиса
var (
	// ErrRequestIDRequired ошибка, когда идентификатор запроса отсутствует
	ErrRequestIDRequired = errors.New("требуется идентификатор запроса (X-Idempotency-Key)")
	// ErrDuplicateRequest ошибка, когда запрос является дубликатом
	ErrDuplicateRequest = errors.New("повторный запрос с тем же идентификатором")
)

type Service interface {
	// ProcessRequestID обрабатывает идентификатор запроса и возвращает true, если запрос новый, и false, если запрос уже обрабатывался
	// В случае, если запрос уже обрабатывался, также возвращает ID ресурса
	ProcessRequestID(ctx context.Context, requestID string, userID uint, resource string) (bool, uint, error)
	// RegisterResourceID регистрирует созданный ресурс для конкретного requestID
	RegisterResourceID(ctx context.Context, requestID string, resourceID uint) error
}

type ServiceImpl struct {
	repo Repository
	ttl  time.Duration // Время жизни записи идентификатора запроса
}

func NewService(repo Repository, ttl time.Duration) Service {
	return &ServiceImpl{
		repo: repo,
		ttl:  ttl,
	}
}

// ProcessRequestID обрабатывает идентификатор запроса
// Возвращает:
// - isNew: true, если запрос новый, false - если уже был обработан
// - resourceID: ID связанного ресурса (для повторных запросов)
// - error: ошибка, если произошла
func (s *ServiceImpl) ProcessRequestID(ctx context.Context, requestID string, userID uint, resource string) (bool, uint, error) {
	if requestID == "" {
		return false, 0, ErrRequestIDRequired
	}

	existingRequest, err := s.repo.GetByID(ctx, requestID)

	if err == nil {
		// Если это запрос по созданию другого ресурса или от другого пользователя - ошибка
		if existingRequest.Resource != resource || existingRequest.UserID != userID {
			return false, 0, ErrDuplicateRequest
		}

		if existingRequest.ResourceID > 0 {
			return false, existingRequest.ResourceID, nil
		}

		// Запись есть, но ресурс еще не создан - считаем как новый запрос
		return true, 0, nil
	}

	if !errors.Is(err, ErrRequestIDNotFound) {
		return false, 0, err
	}

	newRequest := &RequestID{
		ID:        requestID,
		Resource:  resource,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(s.ttl),
	}

	err = s.repo.Create(ctx, newRequest)
	if err != nil {
		// Если запись уже существует, это может быть признаком одновременных запросов
		if errors.Is(err, ErrRequestIDAlreadyExists) {
			return false, 0, ErrDuplicateRequest
		}
		return false, 0, err
	}

	return true, 0, nil
}

func (s *ServiceImpl) RegisterResourceID(ctx context.Context, requestID string, resourceID uint) error {
	if requestID == "" {
		return ErrRequestIDRequired
	}

	return s.repo.UpdateResourceID(ctx, requestID, resourceID)
}
