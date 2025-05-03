package idempotency

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequestIDHeader Название заголовка для X-Idempotency-Key
const RequestIDHeader = "X-Idempotency-Key"

// RequestContext название ключа, под которым информация о результате проверки
// идемпотентности будет храниться в контексте запроса
const RequestContext = "idempotency_context"

// ResourceType тип обрабатываемого ресурса
type ResourceType string

// Response Ответ с данными для определения идемпотентности
type Response struct {
	// IsNew true, если запрос новый
	IsNew bool
	// ExistingResourceID ID существующего ресурса, если запрос повторный
	ExistingResourceID uint
	// RequestID идентификатор запроса из заголовка
	RequestID string
	// Error ошибка при обработке идемпотентности (если есть)
	Error error
}

// GetUserIDFunc функция для получения ID пользователя из контекста запроса
type GetUserIDFunc func(c *gin.Context) uint

// Middleware создает middleware для обработки идемпотентности запросов
func Middleware(service Service, resourceType ResourceType, getUserID GetUserIDFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем RequestID из заголовка
		requestID := c.GetHeader(RequestIDHeader)

		// Получаем ID пользователя
		userID := getUserID(c)
		if userID == 0 {
			// Если ID пользователя не найден, пропускаем обработку идемпотентности
			c.Next()
			return
		}

		// Проверяем идемпотентность запроса
		isNew, existingResourceID, err := service.ProcessRequestID(c.Request.Context(), requestID, userID, string(resourceType))

		// Сохраняем результаты проверки в контексте запроса
		c.Set(RequestContext, Response{
			IsNew:              isNew,
			ExistingResourceID: existingResourceID,
			RequestID:          requestID,
			Error:              err,
		})

		// Если запрос новый, или существующий ресурс уже найден - продолжаем выполнение
		if err == nil {
			c.Next()
			return
		}

		// Обрабатываем ошибки
		switch {
		case errors.Is(err, ErrRequestIDRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "требуется заголовок X-Idempotency-Key"})
			c.Abort()
			return
		case errors.Is(err, ErrDuplicateRequest):
			c.JSON(http.StatusConflict, gin.H{"error": "повторный запрос с тем же идентификатором (X-Idempotency-Key)"})
			c.Abort()
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка при обработке идемпотентности: " + err.Error()})
			c.Abort()
			return
		}
	}
}

// GetRequestContext получает контекст проверки идемпотентности из контекста Gin
func GetRequestContext(c *gin.Context) (Response, bool) {
	value, exists := c.Get(RequestContext)
	if !exists {
		return Response{}, false
	}

	result, ok := value.(Response)
	return result, ok
}

// RegisterResource регистрирует ID созданного ресурса для текущего запроса
func RegisterResource(ctx context.Context, service Service, requestID string, resourceID uint) error {
	if requestID == "" {
		return nil // Если нет requestID, то и регистрировать нечего
	}

	return service.RegisterResourceID(ctx, requestID, resourceID)
}
