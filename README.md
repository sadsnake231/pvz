# Что сделано

Все основные и доп. задания. Дополнительно приделал аутентификацию по jwt токену (он передается через куки при логине и проверяется в middleware) и логгирование ошибок базы данных в репозитории

# Curl

Еще надо накатить миграции (в мейкфайле прописал)

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

