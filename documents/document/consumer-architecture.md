# Kiến Trúc Consumer Service - SMAP Dispatcher

## Tổng Quan

Consumer Service được thiết kế theo mô hình **Multi-Domain Consumer** với khả năng mở rộng linh hoạt. Service này có thể consume nhiều topic từ RabbitMQ và phân rã logic xử lý vào trong từng domain riêng biệt theo nguyên tắc Clean Architecture.

````
┌─────────────────────────────────────────────────────────────────┐
│                        cmd/consumer                              │
│                    (Entry Point - Main)                          │
│  - Khởi tạo dependencies (RabbitMQ, Redis, Logger, etc.)        │
│  - Tạo Consumer Server                                           │
│  - Khởi động tất cả consumers                                    │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    internal/consumer                             │
│                  (Consumer Server Core)                          │
│  - Orchestrator: Khởi tạo và quản lý tất cả domain consumers    │
│  - Dependency injection cho từng domain                          │
│  - Lifecycle management (Run, Close)                             │
└────────────────────────┬────────────────────────────────────────┘
                         │
         ┌───────────────┴───────────────┐
         ▼                               ▼
┌──────────────────────┐        ┌──────────────────────┐
│  Dispatcher Domain   │        │   Results Domain     │
│  ─────────────────   │        │  ──────────────────  │
│  Consumer:           │        │  Consumer:           │
│  • Inbound tasks     │        │  • TikTok results    │
│  • Project events    │        │  • YouTube results   │
│                      │        │                      │
│  Producer:           │        │  (No producer)       │
│  • TikTok tasks      │        │                      │
│  • YouTube tasks     │        │                      │
└──────────────────────┘        └──────────────────────┘


---

## 1. Vai Trò của `cmd/consumer`

### Mục Đích
`cmd/consumer` là **entry point** của toàn bộ Consumer Service. Đây là nơi khởi động ứng dụng và thiết lập môi trường.

### Trách Nhiệm

1. **Khởi tạo Dependencies**
   - Load configuration từ environment variables
   - Khởi tạo Logger (Zap)
   - Kết nối RabbitMQ (fail-fast nếu không kết nối được)
   - Kết nối Redis (cho state management)
   - Khởi tạo Discord webhook (cho error reporting)

2. **Tạo Consumer Server**
   - Inject tất cả dependencies vào `internal/consumer`
   - Cấu hình options cho từng domain (dispatcher, state, webhook)

3. **Lifecycle Management**
   - Khởi động server với context cancellation
   - Lắng nghe OS signals (SIGTERM, SIGINT) để graceful shutdown
   - Đóng tất cả connections khi shutdown

### Code Flow

```go
main()
  ├─ Load Config
  ├─ Init Logger
  ├─ Connect RabbitMQ (fail-fast)
  ├─ Connect Redis (fail-fast)
  ├─ Create Consumer Server
  │   └─ consumer.New(config)
  ├─ Run Server
  │   └─ srv.Run(ctx)
  └─ Wait for shutdown signal
      └─ srv.Close()
````

### Ví Dụ Code

```go
// cmd/consumer/main.go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    // 1. Load config
    cfg, err := config.Load()

    // 2. Init dependencies
    l := pkgLog.Init(...)
    conn, err := rabbitmq.Dial(cfg.RabbitMQConfig.URL, true)
    redisClient, err := pkgRedis.Connect(...)

    // 3. Create consumer server
    srv, err := consumer.New(consumer.Config{
        Logger:      l,
        AMQPConn:    conn,
        RedisClient: redisClient,
        // ... other configs
    })

    // 4. Run
    srv.Run(ctx)
}
```

---

## 2. Vai Trò của `internal/consumer`

### Mục Đích

`internal/consumer` là **orchestrator layer** - lớp điều phối trung tâm quản lý tất cả domain consumers và producers.

### Trách Nhiệm

1. **Domain Orchestration**
   - Khởi tạo UseCase cho từng domain (dispatcher, results, state, webhook)
   - Khởi tạo Consumer cho từng domain
   - Khởi tạo Producer (nếu domain cần publish messages)

2. **Dependency Injection**
   - Inject dependencies từ `cmd/consumer` xuống các domain
   - Tạo và inject cross-domain dependencies (ví dụ: stateUC được dùng bởi cả dispatcher và results)

3. **Consumer Lifecycle**
   - Start tất cả consumers trong goroutines riêng biệt
   - Quản lý shutdown gracefully

### Cấu Trúc

```
internal/consumer/
├── new.go       # Constructor, dependency validation
└── server.go    # Run() method - orchestration logic
```

### Code Flow trong `Run()`

```go
func (srv *Server) Run(ctx context.Context) error {
    // 1. Init Producers (nếu cần)
    prod := dispatcherProducer.New(srv.l, srv.conn)
    prod.Run()

    // 2. Init Microservice Clients
    projectClient := project.NewClient(...)

    // 3. Init UseCases (Business Logic Layer)
    stateRepo := stateRedis.NewRedisRepository(...)
    stateUC := stateUsecase.NewUseCase(...)
    webhookUC := webhookUsecase.NewUseCase(...)
    dispatcherUC := dispatcherUsecase.NewUseCaseWithDeps(...)
    resultsUC := resultsUsecase.NewUseCase(...)

    // 4. Init Domain Consumers
    dispatchC := dispatcherConsumer.NewConsumer(srv.l, srv.conn, dispatcherUC)
    resultsC := resultsConsumer.NewConsumer(srv.l, srv.conn, resultsUC)

    // 5. Start Consumers (non-blocking goroutines)
    dispatchC.Consume()                    // collector.inbound.tasks
    dispatchC.ConsumeProjectEvents()       // smap.events.project.created
    resultsC.Consume()                     // results từ TikTok & YouTube

    // 6. Block until shutdown signal
    <-ctx.Done()
    return nil
}
```

### Điểm Quan Trọng

- **Không chứa business logic**: Chỉ làm nhiệm vụ wiring/orchestration
- **Fail-fast**: Nếu bất kỳ dependency nào fail, service sẽ không start
- **Graceful shutdown**: Đợi context cancellation trước khi đóng connections

---

## 3. Cách Thêm Consumer Mới Trong Domain

### Kịch Bản

Bạn muốn thêm một consumer mới trong domain `dispatcher` để lắng nghe topic `project.updated`.

### Các Bước Thực Hiện

#### Bước 1: Định nghĩa Constants (Exchange, Queue, Routing Key)

File: `internal/dispatcher/delivery/rabbitmq/constants.go`

```go
const (
    // Thêm constants mới
    QueueProjectUpdated      = "collector.project.updated"
    RoutingKeyProjectUpdated = "project.updated"
)
```

#### Bước 2: Tạo Worker Function

File: `internal/dispatcher/delivery/rabbitmq/consumer/workers.go`

```go
// projectUpdatedWorker xử lý event project.updated
func (c Consumer) projectUpdatedWorker(d amqp.Delivery) {
    ctx := context.Background()

    // 1. Parse message
    var event models.ProjectUpdatedEvent
    if err := json.Unmarshal(d.Body, &event); err != nil {
        c.l.Errorf(ctx, "Failed to unmarshal project.updated event: %v", err)
        d.Nack(false, false) // Reject, không requeue
        return
    }

    // 2. Call UseCase
    if err := c.uc.HandleProjectUpdatedEvent(ctx, event); err != nil {
        c.l.Errorf(ctx, "Failed to handle project.updated: %v", err)
        d.Nack(false, true) // Reject và requeue để retry
        return
    }

    // 3. Acknowledge
    d.Ack(false)
}
```

#### Bước 3: Tạo Public Method để Start Consumer

File: `internal/dispatcher/delivery/rabbitmq/consumer/project_consumer.go`

```go
// ConsumeProjectUpdatedEvents starts consuming project.updated events
func (c Consumer) ConsumeProjectUpdatedEvents() {
    go c.consume(
        rabbitmq.SMAPEventsExchangeArgs,
        rabbitmq.QueueProjectUpdated,
        rabbitmq.RoutingKeyProjectUpdated,
        c.projectUpdatedWorker,
    )
}
```

#### Bước 4: Thêm Method vào UseCase Interface

File: `internal/dispatcher/interface.go`

```go
type UseCase interface {
    Dispatch(ctx context.Context, req models.CrawlRequest) ([]models.CollectorTask, error)
    HandleProjectCreatedEvent(ctx context.Context, event models.ProjectCreatedEvent) error
    HandleProjectUpdatedEvent(ctx context.Context, event models.ProjectUpdatedEvent) error // Thêm dòng này
    Producer
}
```

#### Bước 5: Implement Business Logic trong UseCase

File: `internal/dispatcher/usecase/project_event.go`

```go
func (uc *useCaseImpl) HandleProjectUpdatedEvent(ctx context.Context, event models.ProjectUpdatedEvent) error {
    // Business logic xử lý project updated
    // ...
    return nil
}
```

#### Bước 6: Đăng Ký Consumer trong `internal/consumer/server.go`

File: `internal/consumer/server.go`

```go
func (srv *Server) Run(ctx context.Context) error {
    // ... existing code ...

    dispatchC := dispatcherConsumer.NewConsumer(srv.l, srv.conn, dispatcherUC)

    dispatchC.Consume()
    dispatchC.ConsumeProjectEvents()
    dispatchC.ConsumeProjectUpdatedEvents() // Thêm dòng này
    srv.l.Info(ctx, "Dispatcher consumer started (project.updated)")

    // ... rest of code ...
}
```

### Tóm Tắt Flow

```
1. Constants (Exchange/Queue/RoutingKey)
   ↓
2. Worker Function (Parse → UseCase → Ack/Nack)
   ↓
3. Public Consume Method (Start goroutine)
   ↓
4. UseCase Interface (Define contract)
   ↓
5. UseCase Implementation (Business logic)
   ↓
6. Register in internal/consumer (Orchestration)
```

---

## 4. Cách Thêm Producer Mới Trong Domain

### Kịch Bản

Bạn muốn thêm producer để publish task tới platform mới (ví dụ: Instagram).

### Các Bước Thực Hiện

#### Bước 1: Định nghĩa Constants

File: `internal/dispatcher/delivery/rabbitmq/constants.go`

```go
const (
    // Instagram
    ExchangeInstagram   = "instagram_exchange"
    QueueInstagram      = "instagram_crawl_queue"
    RoutingKeyInstagram = "instagram.crawl"
)

var (
    InstagramExchangeArgs = pkgRabbit.ExchangeArgs{
        Name:       ExchangeInstagram,
        Type:       pkgRabbit.ExchangeTypeDirect,
        Durable:    true,
        AutoDelete: false,
        Internal:   false,
        NoWait:     false,
    }
)
```

#### Bước 2: Thêm Method vào Producer Interface

File: `internal/dispatcher/delivery/rabbitmq/producer/new.go`

```go
type Producer interface {
    PublishTikTokTask(ctx context.Context, task models.TikTokCollectorTask) error
    PublishYouTubeTask(ctx context.Context, task models.YouTubeCollectorTask) error
    PublishInstagramTask(ctx context.Context, task models.InstagramCollectorTask) error // Thêm
    Run() error
    Close()
}
```

#### Bước 3: Implement Producer Method

File: `internal/dispatcher/delivery/rabbitmq/producer/producer.go`

```go
func (p implProducer) PublishInstagramTask(ctx context.Context, task models.InstagramCollectorTask) error {
    if p.writer == nil {
        return errors.New("producer not started")
    }

    body, err := json.Marshal(task)
    if err != nil {
        return err
    }

    return p.writer.Publish(ctx, pkgRabbit.PublishArgs{
        Exchange:   rabb.ExchangeInstagram,
        RoutingKey: rabb.RoutingKeyInstagram,
        Msg: pkgRabbit.Publishing{
            Body:        body,
            ContentType: pkgRabbit.ContentTypeJSON,
        },
    })
}
```

#### Bước 4: Thêm vào Domain Interface

File: `internal/dispatcher/interface.go`

```go
type Producer interface {
    PublishTikTokTask(ctx context.Context, task models.TikTokCollectorTask) error
    PublishYouTubeTask(ctx context.Context, task models.YouTubeCollectorTask) error
    PublishInstagramTask(ctx context.Context, task models.InstagramCollectorTask) error // Thêm
}
```

#### Bước 5: Sử Dụng trong UseCase

File: `internal/dispatcher/usecase/dispatch.go`

```go
func (uc *useCaseImpl) Dispatch(ctx context.Context, req models.CrawlRequest) error {
    // ... logic ...

    switch platform {
    case models.PlatformTikTok:
        return uc.PublishTikTokTask(ctx, task)
    case models.PlatformYouTube:
        return uc.PublishYouTubeTask(ctx, task)
    case models.PlatformInstagram:
        return uc.PublishInstagramTask(ctx, instagramTask) // Sử dụng
    }
}
```

### Tóm Tắt Flow

```
1. Constants (Exchange/Queue/RoutingKey)
   ↓
2. Producer Interface (Define method signature)
   ↓
3. Producer Implementation (Marshal → Publish)
   ↓
4. Domain Interface (Expose to UseCase)
   ↓
5. UseCase Usage (Call producer method)
```

---

## 5. Liên Hệ giữa `internal/consumer` và Domain Consumers

### Dependency Flow

```
cmd/consumer (main.go)
    │
    │ Creates & injects dependencies
    ▼
internal/consumer (server.go)
    │
    │ Orchestrates domains
    ├─────────────────┬─────────────────┐
    ▼                 ▼                 ▼
Dispatcher        Results           State
Domain            Domain            Domain
    │                 │                 │
    ├─ Consumer       ├─ Consumer       ├─ Repository
    ├─ Producer       └─ UseCase        └─ UseCase
    └─ UseCase
```

### Ví Dụ Cụ Thể: Dispatcher Domain

#### 1. `internal/consumer` khởi tạo Producer

```go
// internal/consumer/server.go
prod := dispatcherProducer.New(srv.l, srv.conn)
prod.Run() // Chuẩn bị RabbitMQ channel
```

#### 2. `internal/consumer` khởi tạo UseCase và inject Producer

```go
dispatcherUC := dispatcherUsecase.NewUseCaseWithDeps(
    srv.l,
    prod,        // Producer được inject vào UseCase
    // ... other deps
)
```

#### 3. `internal/consumer` khởi tạo Consumer và inject UseCase

```go
dispatchC := dispatcherConsumer.NewConsumer(
    srv.l,
    srv.conn,
    dispatcherUC, // UseCase được inject vào Consumer
)
```

#### 4. `internal/consumer` start Consumer

```go
dispatchC.Consume()
dispatchC.ConsumeProjectEvents()
```

### Luồng Xử Lý Message

```
RabbitMQ Message
    ↓
Consumer (delivery layer)
    ├─ Parse message
    ├─ Validate
    └─ Call UseCase
        ↓
UseCase (business logic)
    ├─ Process business rules
    ├─ Call Producer (if needed)
    └─ Return result
        ↓
Consumer
    ├─ Ack (success)
    └─ Nack (failure)
```

### Cross-Domain Dependencies

Một số UseCase được share giữa nhiều domain:

```go
// stateUC được dùng bởi cả dispatcher và results
stateUC := stateUsecase.NewUseCase(...)

dispatcherUC := dispatcherUsecase.NewUseCaseWithDeps(
    ...,
    stateUC, // Inject stateUC
)

resultsUC := resultsUsecase.NewUseCase(
    ...,
    stateUC, // Inject stateUC
)
```

---

## 6. Best Practices

### 6.1. Tổ Chức Code trong Domain

```
internal/{domain}/
├── delivery/
│   └── rabbitmq/
│       ├── constants.go          # Exchange, Queue, RoutingKey
│       ├── consumer/
│       │   ├── new.go            # Constructor
│       │   ├── consumer.go       # Public Consume methods
│       │   ├── common.go         # Shared consume logic
│       │   └── workers.go        # Worker functions
│       └── producer/
│           ├── new.go            # Constructor
│           └── producer.go       # Publish methods
├── usecase/
│   ├── new.go                    # Constructor
│   └── {feature}.go              # Business logic
├── interface.go                  # Domain interfaces
└── types.go                      # Domain types
```

### 6.2. Naming Conventions

- **Queue**: `{service}.{domain}.{event}` (ví dụ: `collector.project.created`)
- **Routing Key**: `{domain}.{event}` (ví dụ: `project.created`)
- **Exchange**: `{platform}_exchange` hoặc `{service}.events`
- **Worker Function**: `{event}Worker` (ví dụ: `projectEventWorker`)
- **Consume Method**: `Consume{Feature}` (ví dụ: `ConsumeProjectEvents`)

### 6.3. Error Handling

```go
func (c Consumer) worker(d amqp.Delivery) {
    // 1. Parse - Nack without requeue nếu invalid format
    if err := json.Unmarshal(d.Body, &msg); err != nil {
        c.l.Errorf(ctx, "Invalid message format: %v", err)
        d.Nack(false, false) // Không requeue
        return
    }

    // 2. Process - Nack with requeue nếu transient error
    if err := c.uc.Process(ctx, msg); err != nil {
        if isTransientError(err) {
            d.Nack(false, true) // Requeue để retry
        } else {
            d.Nack(false, false) // Permanent error, không retry
        }
        return
    }

    // 3. Success - Ack
    d.Ack(false)
}
```

### 6.4. Testing

- **Unit Test UseCase**: Mock Producer interface
- **Integration Test Consumer**: Sử dụng test RabbitMQ instance
- **Mock Generation**: Sử dụng `//go:generate mockery` comments

```go
//go:generate mockery --name=UseCase
type UseCase interface {
    // ...
}
```

---

## 7. Tóm Tắt

### Vai Trò Các Layer

| Layer                | Vai Trò                            | Ví Dụ                          |
| -------------------- | ---------------------------------- | ------------------------------ |
| `cmd/consumer`       | Entry point, khởi tạo dependencies | Load config, connect DB/MQ     |
| `internal/consumer`  | Orchestrator, wiring domains       | Init UseCases, start consumers |
| `{domain}/delivery`  | Transport layer (RabbitMQ)         | Parse messages, Ack/Nack       |
| `{domain}/usecase`   | Business logic                     | Validate, transform, dispatch  |
| `{domain}/interface` | Contracts giữa layers              | UseCase, Producer interfaces   |

### Checklist Thêm Consumer Mới

- [ ] Định nghĩa constants (Exchange, Queue, RoutingKey)
- [ ] Tạo worker function (parse → usecase → ack/nack)
- [ ] Tạo public Consume method
- [ ] Thêm method vào UseCase interface
- [ ] Implement business logic trong UseCase
- [ ] Đăng ký consumer trong `internal/consumer/server.go`
- [ ] Test với RabbitMQ

### Checklist Thêm Producer Mới

- [ ] Định nghĩa constants (Exchange, Queue, RoutingKey)
- [ ] Thêm method vào Producer interface
- [ ] Implement publish logic
- [ ] Thêm vào domain interface
- [ ] Sử dụng trong UseCase
- [ ] Test publish flow

---

## 8. Ví Dụ Thực Tế: Flow Hoàn Chỉnh

### Scenario: Xử lý Project Created Event

```
1. Project Service publish event
   ↓
2. RabbitMQ: smap.events exchange
   ↓ (routing key: project.created)
3. Queue: collector.project.created
   ↓
4. Dispatcher Consumer (delivery layer)
   ├─ projectEventWorker() parse message
   └─ Call dispatcherUC.HandleProjectCreatedEvent()
       ↓
5. Dispatcher UseCase (business logic)
   ├─ Validate event
   ├─ Generate tasks (YouTube, TikTok)
   ├─ Call prod.PublishYouTubeTask()
   └─ Call prod.PublishTikTokTask()
       ↓
6. Dispatcher Producer (delivery layer)
   ├─ Marshal task to JSON
   └─ Publish to youtube_exchange / tiktok_exchange
       ↓
7. Worker Services consume tasks
   ↓
8. Workers publish results
   ↓
9. Results Consumer (delivery layer)
   ├─ resultWorker() parse result
   └─ Call resultsUC.HandleResult()
       ↓
10. Results UseCase (business logic)
    ├─ Update state in Redis
    ├─ Call webhookUC.NotifyProgress()
    └─ Ack message
```

---

**Tài liệu này mô tả đầy đủ kiến trúc Consumer Service với khả năng mở rộng linh hoạt theo từng domain.**
