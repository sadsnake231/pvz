# UML диаграмма работы кэша

![Кэш](https://cdn-0.plantuml.com/plantuml/png/jLPHQnD147w_Np7qAGsKreelWKYJDhJ1rjGczNszMyt1cOJRDL14i2bMGKGGGH6X2Fw0M1kR6bl_mku_SdPoSNDnhze72zVSxNupD_Dzysso8YIBvQE7aIT36N5GH-gDUkTvp9Vj6DG7DL93DL3dkkTr41ZwOOUr9CoLjgkmmLf1nECvO0BmEGsenG6FwppkXZudH7BlGEtmJbk4-Buz0jh7DDQint86R1TdmK4eLfdVv0IZEozWajrZWiDsMyW7CQ_VsJIRQsMxGt6ULoo2-gGFJUUghvyY1sS5N54NwbITg3wk8Yk03ttr7I_yX6BHcwOB5cuQKWgmNQitOB5j6XM6rh3B92U-y4BKX9W2b7oVToHHaYFylCLFsFEI6nDegIX0dNMvEtAAJdkBuTXs0QgtwvMTXhTMXeMPXBLRNi0TIL8L4AtuSfwg6l9v9lPQVZxlm9Q2eD7U28N9DSPNrowFsEsWEdnNyBY4vIHjYjnqa8rAMBVbfzZ3BO9C2rM0vQLhi1fp10PPoX-XyYDze9_JEca_rFkwfou8CinKVxtoM_p5vhZhnUxovNEmdj2Pi7HtEUSQ11P9x4E_qY_0JA7kwGFy79dtUu1_iqHItYHbRSmxaaXGsZT0cdxKSwkoLQxir26fc42q1mis7SPFHziEx5Ps8MCHoqpwpHoYtNLi6VQ8lCBZC11Ft7LGAdSfTfT7Wa_eGXrHEb9te4QpDTeJnQa4mIrcOgHwPREosW2z3-edol1L3Mb-3caWpYY8J9REhzxIMNAkNwh17ubj5mwWcrcClnzpqKpgJNc4jmA7JrA9h-2O9NHObba-erdEyJh6Qm3uxqloE9gunzZ55c_Rf2wI7fn37xNR7LDBNtrJ9wPck07gXo4RHUtzHSfI5DaJn8v_ffg9hp3xiI58FyEvpb8IirqYeDtFb2nrjf4ZK31NmzUN_hrZjzsp-qsTQV-jrQGgswLhy6Ystfjn8HFrQVnEnvRIhm65GIkhVjOaLLkXfBUwnkXg8t85UZE_lT9vBJAR7IT4VLHzb6X_OYxxpJllud05oSMVThy0)


# Код UML диаграммы

```PlantUML
@startuml
title ПВЗ

actor Клиент
participant "API Handler" as API
participant "OrderService" as Service
participant "OrderRepository" as Repository
participant "ReportRepository" as ReportRepo
participant "RedisCache" as Cache
database PostgreSQL as DB
database Redis

== Сценарий: Создание заказа ==
Клиент -> API: POST /orders
activate API
API -> Service: AcceptOrder(order)
activate Service

Service -> Repository: SaveOrder(order)
activate Repository
Repository -> DB: INSERT
DB --> Repository: OK
deactivate Repository

Service -> Cache: SetOrder(order) **async**
Service -> Cache: AddToHistory(orderID) **async**
Service -> Cache: UpdateUserActiveOrders() **async**

Service --> API: OK
deactivate Service
API --> Клиент: 201 Created

== Сценарий: Получение истории ==
Клиент -> API: GET /orders/history/v2
activate API
API -> Service: GetOrderHistoryV2()
activate Service

alt Кэш актуален
    Service -> Cache: GetHistoryOrderIDs()
    Cache -> Redis: SMEMBERS history
    Redis --> Cache: IDs
    Cache --> Service: IDs
    
    Service -> Cache: GetOrdersBatch(IDs)
    Cache -> Redis: MGET order:{ids}
    Redis --> Cache: Orders
    Cache --> Service: Orders
else Кэш устарел
    Service -> ReportRepo: GetOrderHistoryV2()
    ReportRepo -> DB: SELECT с пагинацией
    DB --> ReportRepo: Данные
    ReportRepo --> Service: Данные
    
    Service -> Cache: RefreshHistory() **async**
end

Service --> API: Данные
deactivate Service
API --> Клиент: 200 OK

== Сценарий: Фоновое обновление ==
Service -> Service: CacheRefresh()
activate Service

loop Каждые 5 минут
    Service -> ReportRepo: GetAllActiveOrderIDs()
    ReportRepo -> DB: SELECT активных ID
    DB --> ReportRepo: IDs
    ReportRepo --> Service: IDs
    Service -> Cache: RefreshActiveOrders(IDs)
end

loop Каждые 30 минут
    Service -> ReportRepo: GetHistoryOrderIDs()
    ReportRepo -> DB: SELECT истории
    DB --> ReportRepo: IDs
    ReportRepo --> Service: IDs
    Service -> Cache: RefreshHistory(IDs)
end

deactivate Service

== Инициализация кэша ==
Service -> Service: InitCache()
activate Service
Service -> ReportRepo: GetAllActiveOrderIDs()
Service -> ReportRepo: GetHistoryOrderIDs()
Service -> Cache: UpdateAllActiveOrders()
Service -> Cache: RefreshHistory()
Service -> Cache: Массовое SetOrder()
deactivate Service
@enduml

```
# Принцип инвалидации кэша

Инвалидация по событию + периодическое обновление + TTL в зависимости от статуса заказа

# Новые curl, работающие с кэшем

Активные заказы пользователя

```sh
curl -X get "http://localhost:9000/user0/orders/active" \
     -b cookies.txt
```

Все активные заказы

```sh
curl -X get "http://localhost:9000/active" \
     -b cookies.txt
```

История заказов

```sh
curl -X get "http://localhost:9000/history/v2" \
     -b cookies.txt
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

