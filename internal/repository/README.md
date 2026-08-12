# internal/repository

Слой персистентности сокращённых URL. Интерфейс `Repository` и фабрика `New` выбирают бэкенд по приоритету:

1. **PostgreSQL**
2. **Файл**
3. **In-memory**

Реализации: `PostgresRepository`, `FileRepository`, `MemoryRepository`.
