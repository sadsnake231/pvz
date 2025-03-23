# Получение логов через пагинацию с курсором по адресу


```sh
curl -X GET "http://localhost:9000/logs?limit=5&cursor=100" \
     -b cookies.txt
```

# Фильтр

Установить AUDIT_FILTER в .env

# Prometheus, Grafana

Сначала надо поднять docker-compose

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

