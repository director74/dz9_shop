package idempotency

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ResourceGetter func(ctx *gin.Context, resourceID uint) (interface{}, error)

func ResponseHandler(c *gin.Context, resourceGetter ResourceGetter) (bool, interface{}) {
	// Получаем контекст проверки идемпотентности
	idempCtx, exists := GetRequestContext(c)
	if !exists {
		// Если контекст не найден, продолжаем обычное выполнение
		return true, nil
	}

	// Если есть ошибка идемпотентности, она уже должна была быть обработана в middleware
	if idempCtx.Error != nil {
		return false, nil
	}

	// Если запрос новый, продолжаем обычное выполнение
	if idempCtx.IsNew || idempCtx.ExistingResourceID == 0 {
		return true, nil
	}

	// Запрос повторный и у нас есть ID ресурса, получаем его
	resource, err := resourceGetter(c, idempCtx.ExistingResourceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("ошибка при получении существующего ресурса: %v", err)})
		return false, nil
	}

	c.JSON(http.StatusOK, gin.H{
		"resource": resource,
		"message":  "ресурс уже был создан ранее с этим идентификатором запроса",
	})
	return false, resource
}
