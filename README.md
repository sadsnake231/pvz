# Паттерны

Используются паттерны **Стратегия** и **Компоновщик**. С помощью Стратегии мы инкапсулируем семейство алгоритмов. Каждый вид упаковки - свой отдельный алгоритм для расчета цены и проверки веса. Этот же паттерн помогает при необходимости быстро добавить еще один вид упаковки: добавить еще одну структуру Packaging4 и еще один case в функции parsePackaging. При выборе какого-нибудь другого паттерна (Декоратор, Фабрика, etc.) так легко это бы не получилось. Этим и обусловлен выбор Стратегии. 
С помощью Компоновщика мы эффективно работаем с комбинациями упаковок.

# Пример json файла с заказами от курьера

```json
[
    {
        "id": "1",
        "recipient_id": "user1",
        "expiry": "2025-12-31",
        "base_price": 100,
        "weight": 5,
        "packaging": "пакет"
    },
    {
        "id": "2",
        "recipient_id": "user2",
        "expiry": "2025-03-15",
        "base_price": 200,
        "weight": 15,
        "packaging": "коробка+пленка"
    }
]

```
# Пример ввода accept

```
accept 16 user1 2025-03-02 1000 5 коробка+пленка

```

# Пример файла .env

```
ORDER_STORAGE_PATH=./data/orders.json

```

# Диаграмма классов

Использовался [Plant UML](https://plantuml.com/ru/guide)

![Диаграмма](https://www.plantuml.com/plantuml/png/tLPDRvj04BtpA_O8YOdKfEe1LLLQMwvTjObbA-eXoc631xpA0jP-fBNS_lS2kmR3uE1AgG_rmTgTVVF1lZSCrr9HeNJ2dP1JASdmwtH2qoK7GROEoN--1F7CGWZ8GcM2nh0m-_Bm_0P-r1xk0QWNaBCQvVns79Og414DeOMqESy_XY6irQrOva6fY6L0xX-a4YoDyrYkMGq829493td8iSYIBulRcp7Zu4RvNqeJ2CZB0UQWj4Z_9kLKIWrpY7IwD7cFzFiCwaw2EEXp_r0U8IUJ2dR9E4kO2WXBrO1aKIJ1R5wAK5edJAfCRDo3m0dmjOkBptcp6f5TdFa2rbb0odZpV2bPaQLIIPDdjnFi8OqrfD92TsabA_vjN2ymXMDwsJ8WHr1hxrbB4DDHG7Qu8aTkw2RQs50yGzth14GQmX7tfI4LLLhMPrWgfnORqXFYr2-ln6h1qcbtvbpRPwynTorBhKcXciWvibKYhGjn3rSm8FtP1-Iup1vk62nvYztfVK6VeK_jOsRLzcap7RNqNd9uri7P23Wew6-HHedeqSdsw1PwxGxgZuVBnvoGxN_OOiKux7X8lvYECwfMk-ADDT7vVfE65z-qWfzLUxI3k6tlOj2tGSthdj7mYSOvTA44LH1NR_4XvQ7ckMacYldmIALP1IJY6LNltdhLoZgi87pw3YnUgH9jbllmnopZzGrKcT-SFkNbwLDUdIpbC4lAbnTKkNVYiXmpA4SRelwVrxVtBUlx7zhV5irjWK3lujQ-nbZVJW2QH1sWhY7KQC_tlouk26eb6xeFPeFt0xlk0DE4B2PQtL2zYCjUnDKX15QNurn3kjG9_mC0)

Исходный код на языке PlantUML

```
@startuml
class CLIHandler {
    -service: StorageService
    +NewCLIHandler(service: StorageService): *CLIHandler
}

interface StorageService {
    +AcceptOrder(args: []string): (string, error)
    +AcceptOrdersFromJSONFile(filename: string): (string, error)
    +ReturnOrder(args: []string): (string, error)
    +IssueRefundOrders(args: []string): (string, error)
    +GetUserOrders(args: []string): ([]Order, error)
    +GetRefundedOrders(limit: int, offset: int): ([]Order, error)
    +GetOrderHistory(): ([]Order, error)
    +Help(): (string, error)
}

class storageService {
    -repo: OrderRepository
    +NewStorageService(repo: OrderRepository): StorageService
}

interface OrderRepository {
    +AcceptOrder(order: Order): error
    +ReturnOrder(id: string): (string, error)
    +IssueOrders(userID: string, orderIDs: []string): (string, []string, error)
    +RefundOrders(userID: string, orderIDs: []string): (string, []string, error)
    +GetUserOrders(userID: string, limit: int, status: string, offset: int): ([]Order, error)
    +GetRefundedOrders(limit: int, offset: int): ([]Order, error)
    +GetOrderHistory(): ([]Order, error)
}

class Repository {
    -orderStorage: OrderStorage
    -userOrderStorage: UserOrderStorage
    -reportOrderStorage: ReportOrderStorage
    +NewRepository(orderStorage: OrderStorage, userOrderStorage: UserOrderStorage, reportOrderStorage: ReportOrderStorage): OrderRepository
}

interface OrderStorage {
    +SaveOrder(order: Order): error
    +FindOrderByID(id: string): (int, *Order, error)
    +DeleteOrder(id: string): (string, error)
}

interface UserOrderStorage {
    +IssueOrders(userID: string, orderID: []string): (string, []string, error)
    +RefundOrders(userID: string, orderID: []string): (string, []string, error)
}

interface ReportOrderStorage {
    +GetUserOrders(userID: string, limit: int, status: string, offset: int): ([]Order, error)
    +GetRefundedOrders(limit: int, offset: int): ([]Order, error)
    +GetOrderHistory(): ([]Order, error)
}

class JSONOrderStorage {
    -filePath: string
    -mu: sync.Mutex
    +NewJSONOrderStorage(filePath: string): *JSONOrderStorage
}

interface PackagingStrategy {
    +CalculatePrice(basePrice: float64): float64
    +CheckWeight(baseWeight: float64): bool
}

class Packaging1 {
    +CalculatePrice(basePrice: float64): float64
    +CheckWeight(baseWeight: float64): bool
}

class Packaging2 {
    +CalculatePrice(basePrice: float64): float64
    +CheckWeight(baseWeight: float64): bool
}

class Packaging3 {
    +CalculatePrice(basePrice: float64): float64
    +CheckWeight(baseWeight: float64): bool
}

class CompositePackaging {
    -Strategies: []PackagingStrategy
    +CalculatePrice(basePrice: float64): float64
    +CheckWeight(baseWeight: float64): bool
}

CLIHandler --> StorageService
StorageService --> OrderRepository
OrderRepository --> OrderStorage
OrderRepository --> UserOrderStorage
OrderRepository --> ReportOrderStorage
OrderStorage <|.. JSONOrderStorage
UserOrderStorage <|.. JSONOrderStorage
ReportOrderStorage <|.. JSONOrderStorage

StorageService --> PackagingStrategy
PackagingStrategy <|.. Packaging1
PackagingStrategy <|.. Packaging2
PackagingStrategy <|.. Packaging3
PackagingStrategy <|.. CompositePackaging
@enduml
```
# Makefile

Команды можно посмотреть через **make help**
