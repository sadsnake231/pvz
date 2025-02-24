# Пример json файла с заказми от курьера

```json
[
    {"id": "1500", "recipient_id": "15", "expiry": "2025-12-31"},
    {"id": "1601", "recipient_id": "16", "expiry": "2024-11-30"}
]
```
# Пример json файла - хранилища даннвх

```json
[
  {
    "id": "order1",
    "recipient_id": "user",
    "expiry": "2025-03-02T00:00:00Z",
    "status": "refunded",
    "UpdatedAt": "2025-02-22T13:39:23.886836111+03:00"
  },
  {
    "id": "order3",
    "recipient_id": "user",
    "expiry": "2025-03-02T00:00:00Z",
    "status": "refunded",
    "UpdatedAt": "2025-02-22T13:39:23.886836111+03:00"
  },
  {
    "id": "order2",
    "recipient_id": "user",
    "expiry": "2025-03-02T00:00:00Z",
    "status": "refunded",
    "UpdatedAt": "2025-02-22T13:39:28.804082217+03:00"
  },
  {
    "id": "order5",
    "recipient_id": "user",
    "expiry": "2025-03-02T00:00:00Z",
    "status": "issued",
    "UpdatedAt": "2025-02-22T13:39:23.886836111+03:00"
  },
  {
    "id": "order36",
    "recipient_id": "user",
    "expiry": "2024-03-02T00:00:00Z",
    "status": "stored",
    "UpdatedAt": "2025-02-22T13:39:23.886836111+03:00"
  }
]

```

# Пример файла .env

```
ORDER_STORAGE_PATH=./data/orders.json

```
