#!/bin/bash

# Скрипт для применения тестовых миграций ко всем сервисам
# Для запуска: chmod +x apply_test_migrations.sh && ./apply_test_migrations.sh

# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Получаем абсолютный путь к текущему скрипту
SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Корень проекта - две директории вверх от каталога скрипта
PROJECT_ROOT="$(cd "$SCRIPT_PATH/../../.." && pwd)"
MIGRATIONS_PATH="$PROJECT_ROOT/src/migrations"

echo -e "${YELLOW}Подготовка баз данных и применение миграций для всех сервисов...${NC}"
echo -e "${YELLOW}Путь к миграциям: $MIGRATIONS_PATH${NC}"

# Соответствие сервисов и имен баз данных
declare -A DB_NAMES
DB_NAMES["delivery_service"]="delivery"
DB_NAMES["warehouse_service"]="warehouse"
DB_NAMES["billing_service"]="billing"
DB_NAMES["order_service"]="orders"
DB_NAMES["notification_service"]="notifications"

# Функция для создания базы данных, если она не существует
create_database() {
    service=$1
    container=$2
    db_name=${DB_NAMES[$service]}
    echo -e "${YELLOW}Проверяем/создаем базу данных для $service (имя БД: $db_name) в контейнере $container...${NC}"
    
    # Проверяем существование базы данных
    if docker exec -i $container psql -U postgres -c "SELECT 1 FROM pg_database WHERE datname='$db_name'" | grep -q 1; then
        echo -e "${GREEN}База данных $db_name уже существует${NC}"
    else
        echo -e "${YELLOW}Создаем базу данных $db_name...${NC}"
        docker exec -i $container psql -U postgres -c "CREATE DATABASE $db_name;"
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}База данных $db_name успешно создана${NC}"
        else
            echo -e "${RED}Ошибка при создании базы данных $db_name${NC}"
            exit 1
        fi
    fi
}

# Функция для выполнения миграций
apply_migrations() {
    service=$1
    port=$2
    db_name=${DB_NAMES[$service]}
    echo -e "${YELLOW}Применяем миграции для $service в базу $db_name...${NC}"
    
    # Проверяем наличие директории с миграциями
    if [ ! -d "$MIGRATIONS_PATH/$service" ]; then
        echo -e "${RED}Ошибка: директория с миграциями для $service не найдена: $MIGRATIONS_PATH/$service${NC}"
        ls -la $MIGRATIONS_PATH
        exit 1
    fi
    
    # Применяем схему (если она еще не применена)
    migrate -path $MIGRATIONS_PATH/$service -database "postgres://postgres:postgres@localhost:$port/$db_name?sslmode=disable" up
    
    migration_result=$?
    if [ $migration_result -eq 0 ]; then
        echo -e "${GREEN}Миграции схемы для $service применены успешно${NC}"
    elif [ $migration_result -eq 1 ]; then
        # Код возврата 1 обычно означает, что нет новых миграций для применения
        echo -e "${GREEN}Для $service нет новых миграций или они уже применены${NC}"
    else
        # Проверяем, не связана ли ошибка с уже существующими таблицами
        if migrate -path $MIGRATIONS_PATH/$service -database "postgres://postgres:postgres@localhost:$port/$db_name?sslmode=disable" version 2>&1 | grep -q "Dirty database version"; then
            echo -e "${YELLOW}База данных $db_name для $service помечена как 'dirty'. Попытка исправления...${NC}"
            migrate -path $MIGRATIONS_PATH/$service -database "postgres://postgres:postgres@localhost:$port/$db_name?sslmode=disable" force $(migrate -path $MIGRATIONS_PATH/$service -database "postgres://postgres:postgres@localhost:$port/$db_name?sslmode=disable" version | grep -o "[0-9]*$")
            echo -e "${GREEN}Состояние миграций для $service исправлено${NC}"
        else
            echo -e "${RED}Ошибка при применении миграций схемы для $service${NC}"
            # Продолжаем выполнение, не останавливаем скрипт
            # exit 1
        fi
    fi
}

# Создаем базы данных для всех сервисов
create_database "delivery_service" "delivery-db"
create_database "warehouse_service" "warehouse-db"
create_database "billing_service" "postgres"
create_database "order_service" "postgres"
create_database "notification_service" "postgres"

# Применяем миграции для всех сервисов
apply_migrations "delivery_service" 5435
apply_migrations "warehouse_service" 5434
apply_migrations "billing_service" 5432
apply_migrations "order_service" 5432
apply_migrations "notification_service" 5432

echo -e "${GREEN}Все миграции успешно применены!${NC}"
echo -e "${YELLOW}Теперь можно использовать Postman-коллекцию для тестирования саги${NC}" 