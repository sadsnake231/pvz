# UML диаграмма работы кэша

![Кэш](https://cdn-0.plantuml.com/plantuml/png/tLLVQnD147_VJp7qAG7xftqyY3HjGXii9YQfh-NChlRWcWktbwA8e2bOmOfG1C6NAlW1eZQsjYP-XRrlvEouxLtJlPhuQE2IsztVpCxyvZUJPHcvOTJTyLdaHRu0TQGErNl6t5OKXA0-g7DrgWPg6FTO1u5Uo-kuxGXcYhKAhBjjxejm1bm9kBuAg8PSX0qHxdejGGZpRa7LHIUG7jxtX839yYgs5iZeKIWQzpO8LbgvUbmNt9EhRcjRq36zCUv6xWDN2g1JnWe5K9_YbpeZrO-VOsu_7DnH8tMYZbKVTp5Rm4LVdU63lr0ei3GDC7jR9-j0hTfemdoW4v2QBc_NZGJnW8z6cOGTABPQeXDHgZd2vnEy0J5cNj0mHO033NTNM2tJD2MlQ8x4E52vsoZNc_ZJh889iRG2isFEgtSO5pQ7wMY8I0n4s3CmHEjlWXFH22ytWP3EKuy_OxHBznDm_6hQjDswsU5ulYb5M6bpyJWtDpUNpuGLVR-eFOe9iEkUr9cca0dKLDZ4E5ufF298MXGEB2qimcee4CYpL7Q4AZkFz-9zsYUTdSOlufuQ4UGq9Jj4ViqKtbhvKdVpst_Ik5b_QwPq9kVxIpcB02rCePqn8VDAIUR-MlUrYxflNOJ6N3z2ik8tyIjG0vrmVC00FzSqWHVGWV6Pf90SsO8qklHu7jGnClmMYCJlk1YfRsg4napuKbm6yGkBW67OWsXtVJyh2dL4bbWnGjXRraeRuIsTHllNsxTHtYYlDW5rIG1BwwBNSLihOLnatJfHPw2RF1DjRkaEzOxcvJ1-hGPZqfscRHQmpR8wvX1z7-DwN_N7RfXeAizxKdR5pRrHqP0gmNDTgQjr2m9JjYd6TDeDLctICRrPAnMRPQ0u-PORJJUa_I3eO1biaa__WqtGE0d4FlExf0oUAOMFM8MJU0yZEOXxMB3UlodWCqO_rEN5Pm3uzSKipvEHUQPxcFwMapXIzQJ8w-9XAEIsRSmHPnW1aKNINLm0zHDz3x6tU_z1Nq_VtBPYBUD-TyV_3G00)


# Код UML диаграммы

```PlantUML
@startuml
title ПВЗ

actor Клиент
participant "API Handler" as API
participant "OrderService" as Service
participant "OrderRepository" as Repository
participant "PostgreSQL" as DB
participant "RedisCache" as Cache
database Redis

== Сценарий: Создание заказа ==
Клиент -> API: POST /orders
activate API
API -> Service: AcceptOrder(order)
activate Service

Service -> Repository: SaveOrder(order)
activate Repository
Repository -> DB: INSERT INTO orders
DB --> Repository: OK
deactivate Repository

Service -> Cache: SetOrder(order)
activate Cache
Cache -> Redis: SET order:{id}
Redis --> Cache: OK
deactivate Cache

Service -> Cache: UpdateUserIndex(...)
Service -> Cache: UpdateAllActiveIndex(...)
Service -> Cache: UpdateHistoryIndex(...)

Service --> API: OK
deactivate Service
API --> Клиент: 201 Created

== Сценарий: Получение истории заказов ==
Клиент -> API: GET /orders/history
activate API
API -> Service: GetOrderHistoryV2()
activate Service

Service -> Cache: GetHistoryOrderIDs()
activate Cache
Cache -> Redis: GET order_history

alt Кэш есть
    Redis --> Cache: IDs
    Cache --> Service: IDs
    Service -> Cache: GetOrder(id) для каждого ID
else Кэш пуст
    Redis --> Cache: null
    Cache --> Service: Пусто
    Service -> Repository: GetHistoryOrderIDs()
    activate Repository
    Repository -> DB: SELECT ...
    DB --> Repository: IDs
    Repository --> Service: IDs
    deactivate Repository
    Service -> Cache: UpdateHistoryIndex(IDs)
end

Service --> API: Данные заказов
deactivate Service
API --> Клиент: 200 OK

== Сценарий: Выдача заказов ==
Клиент -> API: POST /orders/issue
activate API
API -> Service: IssueOrders(userID, orderIDs)
activate Service

Service -> Repository: IssueOrders(...)
activate Repository
Repository -> DB: UPDATE issued_at
DB --> Repository: OK
deactivate Repository

loop Для каждого заказа
    Service -> Cache: GetOrder(id)
    Service -> Cache: SetOrder(updated)
end

Service --> API: Результат
deactivate Service
API --> Клиент: 200 OK

== Фоновое обновление кэша ==
Service -> Service: CacheRefresh()
activate Service
loop Каждые 10 минут
    Service -> Repository: GetHistoryOrderIDs()
    activate Repository
    Repository -> DB: SELECT ...
    DB --> Repository: IDs
    Repository --> Service: IDs
    deactivate Repository
    
    Service -> Cache: UpdateHistoryIndex(IDs)
end
deactivate Service
@enduml

```








# Получение логов через пагинацию с курсором по адресу


```sh
curl -X GET "http://localhost:9000/logs?limit=5&cursor=100" \
     -b cookies.txt
```

# Фильтр

Установить AUDIT_FILTER в .env

# Prometheus, Grafana

Сначала надо поднять docker-compose 

```sh
make docker-up
```

Вход в Grafana: http://localhost:3000, логин и пароль admin

В Grafana в Data Source надо прописать http://host.docker.internal:9090 

## Запросы в Grafana для графиков:

количество запросов в минуту:
```
sum(increase(http_requests_total[1m]))
```

количество запросов, разбитое по методам:
```
sum(rate(http_requests_total[5m])) by (method)
```

средняя загрузка процессора
```
rate(process_cpu_seconds_total[5m])
```

и т.д. ...

# Curl

Регистрация
```sh
curl -X POST http://localhost:9000/users/signup \
     -H "Content-Type: application/json" \
     -d '{
          "email": "user@example.com",
          "password": "securepassword"
         }'

```

Логин
```sh
curl -X POST http://localhost:9000/users/login \
     -H "Content-Type: application/json" \
     -c cookies.txt \
     -d '{
          "email": "user@example.com",
          "password": "securepassword"
         }'
```

Принять заказ
```sh
curl -X POST http://localhost:9000/orders \
     -H "Content-Type: application/json" \
     -b cookies.txt \
     -d '{
          "id": "order123",
          "recipient_id": "user1",
          "expiry": "2025-12-31",
          "base_price": 1000,
          "weight": 5,
          "packaging": "коробка"
         }'
```

Вернуть заказ курьеру
```sh
curl -X DELETE http://localhost:9000/orders/order123/return \
     -b cookies.txt
```

Выдать/вернуть заказы пользователя
```sh
curl -X PUT http://localhost:9000/actions/issues_refunds \
     -H "Content-Type: application/json" \
     -b cookies.txt \
     -d '{
          "command": "issue",
          "user_id": "user1",
          "order_ids": ["order1", "order2"]
         }'
```


Получить заказы пользователя. Выдает заказы и следующий курсор
```sh
curl -X GET "http://localhost:9000/reports/user1/orders?limit=2&cursor=10&status=stored" \
     -b cookies.txt
```

Получить возвращенные заказы. Выдает заказы и следующий курсор
```sh
curl -X GET "http://localhost:9000/reports/refunded?limit=5&cursor=20" \
     -b cookies.txt
```

Получить историю заказов. Выдает заказы и следующий курсор
Не написал курсор в url, так как там формат времени с наносекундами. Можно выполнить этот курл и дописать курсор
&cursor= то, что выдаст запрос
```sh
curl -X GET "http://localhost:9000/reports/history?limit=5" \
     -b cookies.txt
```

