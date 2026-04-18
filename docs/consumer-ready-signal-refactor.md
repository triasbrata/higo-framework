# Consumer Ready Signal Refactor — Discussion

**Status:** All semantic questions resolved — ready for implementation.
**File under refactor:** `higo/messagebroker/consumer/amqp/cosumer.go`
**Related:** `higo/server/consumer/server.go` (the caller that blocks on `c.Status()`)

---

## Context

Saat running 3 server (HTTP, gRPC, consumer) dalam 1 binary, kita sudah implement `fx.Shutdowner` supaya kalau salah satu server mati, yang lain ikut graceful shutdown. Juga sudah tambah `running atomic.Bool` di HTTP dan gRPC server untuk skip `ShutdownWithContext` / `GracefulStop` kalau server gagal start.

Saat telusuri consumer server, ditemukan bug struktural pada `chanOk` — sinyal "consumer ready" dikirim terlalu dini (TCP-level), bukan saat consumer benar-benar subscribe ke queue.

---

## Current Structure (Buggy)

```
Start(ctx)
  │
  ├─── errgroup.WithContext(ctx) ──► eg, gctx
  │
  ├─── chanStack := make(chan amqpStack, len(c.stack))
  │
  ├─── eg.Go [Goroutine-A: Connection Listener]
  │      │
  │      │   for range c.man.Ready() {           ← infinite loop
  │      │     ┌──────────────────────────────┐       (readyChan never closed)
  │      │     │ if len(stack)==0 return err  │
  │      │     │ chanOk ← struct{}{}  ❌BUG 1 │  signal "ready" di TCP-level
  │      │     │ for _, s := range stacks {   │       (belum ada subscribe apapun)
  │      │     │   chanStack ← s              │
  │      │     │ }                            │
  │      │     └──────────────────────────────┘
  │      │   }
  │
  ├─── for stack := range chanStack {           ← main goroutine blocks
  │      │
  │      └─── eg.Go [Goroutine-B: Per-Queue Consumer]
  │             │
  │             │  buildAndConsume(gctx, stack)
  │             │    ├─ buildTopology()          QueueDeclare, QueueBind
  │             │    └─ startConsuming()
  │             │         ├─ ch := Channel()
  │             │         ├─ ch.Qos()
  │             │         ├─ del := ConsumeWithContext()  ← subscribe to queue
  │             │         │                                 (baru disini consumer
  │             │         │                                  benar-benar siap)
  │             │         └─ for {
  │             │              select {         ← line 182
  │             │                case <-chNotif: return err
  │             │                case msg := <-del:
  │             │                  go processMessage(...)
  │             │              }
  │             │            }
  │             │
  │             ├─ err == nil → time.Sleep + chanStack ← stack (restart)
  │             └─ err != nil → return err
  │
  └─── eg.Wait() ──► never returns (Goroutine-A infinite loop)
```

---

## Bugs Identified

### 🔴 BUG #1: Ready signal di level yang salah
- `chanOk` dikirim saat `man.Ready()` yield (= TCP connection established)
- Pada titik itu: topology belum dibuat, `QueueDeclare` belum jalan, `ConsumeWithContext` belum dipanggil
- Seharusnya dikirim setelah `ConsumeWithContext` sukses → masuk `for { select }` loop

### 🔴 BUG #2: Multi-queue tidak di-coordinate
- Kalau ada 3 queue, `chanOk` dikirim 1x di level connection — bukan setelah 3 queue semua berhasil subscribe
- Tidak ada mekanisme "tunggu semua queue ready"

### 🔴 BUG #3: Separation of concerns tercampur
- Goroutine-A merangkap 2 tugas: (a) dengar reconnection event, (b) dispatch initial stacks + ready signal
- Reconnection path meng-overlap dengan initial setup path

### 🔴 BUG #4: Reconnection kirim `chanOk` palsu
- Saat reconnect, `man.Ready()` yield lagi → goroutine-A coba kirim `chanOk` lagi
- `select { case chanOk <- struct{}{}: default: }` tidak blocking, tapi semantiknya salah — OnStart sudah lama return, tidak ada yang peduli

---

## Proposed Structure (Full Coordinator — Multi-Queue Aware)

```
Start(ctx)
  │
  ├─── eg.Go [Dispatcher]
  │      └─ for range man.Ready() { dispatch stacks to chanStack }
  │         (TANPA chanOk disini — cuma dispatch)
  │
  ├─── for stack := range chanStack
  │      └─ eg.Go [Per-Queue Consumer]
  │            └─ buildAndConsume
  │                 └─ startConsuming
  │                      ├─ ConsumeWithContext() sukses
  │                      ├─ signal "queue-X ready"      ◄── PER-QUEUE signal
  │                      └─ for { select { ... } }
  │
  └─── (ready coordinator — lihat bagian berikutnya)
```

---

## Coordinator — Pilihan Mekanisme

### Alternatif yang Dipertimbangkan

| # | Mekanisme | Pros | Cons |
|---|-----------|------|------|
| 1 | `sync.WaitGroup` | Idiom standar | ❌ Kalau 1 queue gagal subscribe, `wg.Done()` tidak pernah dipanggil → `wg.Wait()` hang |
| 2 | Channel per-queue (`readyCh chan queueReady`) | Bisa carry error info | ❌ Butuh coordinator goroutine terpisah; buffer sizing tricky saat reconnect |
| 3 | `atomic.Int32` counter + `sync.Once` | ✅ Error path bersih via errgroup; reconnect-safe; lock-free | ⚠️ Semantik 1-shot — tidak continuous monitoring |
| 4 | `sync.Map` untuk per-queue status + transition detection | ✅ Continuous monitoring; bisa flip false↔true per reconnect/disconnect | ⚠️ `sync.Map` tidak ideal untuk pattern kita (fixed keys) |

### Pilihan User: Alternatif #4 — per-queue status dengan `map[string]*atomic.Bool`

**Alasan awal user:**
> "mending pakai sync.Map jadi kalau pun consumer mengalami re-connect kita bisa flag dari false ke true, dan kalau dia putus dari maka di flaging ke false. ini akan membuat c.Status() bisa di pakai untuk monitor status secara terus menerus"

**Keputusan final (Q4 resolved):** gunakan `map[string]*atomic.Bool` yang di-populate sekali di awal `Start()` dari `c.stack`. Map structure **immutable** setelah init — hanya `atomic.Bool` value yang flip true↔false. Tidak butuh locking, snapshot konsisten, lebih cocok dengan pattern "fixed keys" kita.

### Draft Design (perlu klarifikasi Q1–Q3)

```go
type csmr struct {
    // ...
    queueStatus map[string]*atomic.Bool  // populated once in Start(); values flip
    readyOrder  []string                 // queue names in stable order for iteration
    chanOk      chan struct{}            // fires on every transition → "all ready"
    chanErr     chan error               // fires on every transition → "any not ready"
}

// Di awal Start() — populate sekali, tidak pernah mutasi struktur lagi:
c.queueStatus = make(map[string]*atomic.Bool, len(c.stack))
c.readyOrder = make([]string, 0, len(c.stack))
for _, s := range c.stack {
    c.queueStatus[s.queueName] = &atomic.Bool{}  // default false
    c.readyOrder = append(c.readyOrder, s.queueName)
}

// Aggregate state tracker untuk detect transitions
var aggregateState atomic.Bool  // current aggregate "all ready?" state

setQueueReady := func(name string, ready bool) {
    flag, ok := c.queueStatus[name]
    if !ok { return } // defensive — queue tidak terdaftar
    flag.Store(ready)

    // Recompute aggregate (lock-free, konsisten karena keys fixed)
    allReady := true
    for _, n := range c.readyOrder {
        if !c.queueStatus[n].Load() {
            allReady = false
            break
        }
    }

    // Fire hanya saat transition, bukan setiap kali
    prev := aggregateState.Swap(allReady)
    if prev != allReady {
        if allReady {
            select { case c.chanOk <- struct{}{}: default: }
        }
        // optional: signal "not ready" transition (lihat Q2)
    }
}
```

**Call sites untuk `setQueueReady`:**
- `startConsuming()` setelah `ConsumeWithContext` sukses → `setQueueReady(stack.queueName, true)`
- `startConsuming()` saat `chNotif` fire atau `del` closed → `setQueueReady(stack.queueName, false)` via `defer`
- (opsional) `buildTopology()` pada error → tidak perlu set false karena initial state sudah false

**Keuntungan pendekatan ini vs `sync.Map`:**
- Tidak ada lock contention — setiap slot punya `*atomic.Bool` independen
- Snapshot konsisten — iterate `readyOrder` selalu lihat semua keys yang sama
- Tidak ada cost dari `sync.Map`'s internal read/dirty tracking
- `readyOrder` slice memastikan iteration order stable (berguna untuk logging/debugging)

---

## ✅ All Questions Resolved — Ready to Implement

### ✅ Q1. Semantik `chanOk` dan `chanErr` — RESOLVED

**Keputusan:** continuous trigger — `chanOk` fires setiap transition `any-down → all-up`, `chanErr` fires setiap transition `all-up → any-down`.

**Konsekuensi:**
- OnStart di `server/consumer/server.go` tetap valid — first `<-ok` akan fire saat semua queue initial-subscribed sukses
- Setiap reconnect yang berhasil restore semua queue akan re-fire `chanOk` — caller yang masih listening bisa react
- Setiap loss of readiness akan fire `chanErr` dengan error deskriptif `"consumer queue X lost readiness"`
- Pattern buffer-1 + non-blocking send → kalau signal consecutive datang sebelum caller consume, yang lama di-drop (acceptable karena state real-time bisa di-query via `IsQueueReady`)

**Implikasi untuk server/consumer/server.go:**
- OnStart tetap bekerja correctly: blocks di `<-ok` → unblocks saat semua queue ready pertama kali → return nil
- Setelah OnStart return, caller **boleh** terus dengarkan `chanOk`/`chanErr` untuk monitoring runtime — tapi untuk sekarang tidak dipakai (server.go tidak listen setelah OnStart)
- Kalau nanti mau monitoring runtime, tinggal tambah goroutine yang listen channels setelah OnStart

**Pseudo-code lengkap `setQueueReady`:**
```go
setQueueReady := func(name string, ready bool) {
    c.queueStatus[name].Store(ready)

    allReady := true
    for _, n := range c.readyOrder {
        if !c.queueStatus[n].Load() {
            allReady = false
            break
        }
    }

    prev := aggregateState.Swap(allReady)
    if prev == allReady {
        return  // no transition, nothing to signal
    }
    if allReady {
        // any-down → all-up (re-fires on every reconnect recovery)
        select { case c.chanOk <- struct{}{}: default: }
    } else {
        // all-up → any-down
        err := fmt.Errorf("consumer queue %q lost readiness", name)
        select { case c.chanErr <- err: default: }
    }
}
```

### ✅ Q2. Signal saat disconnect — RESOLVED

**Keputusan:** opsi (a) — `chanErr` dikirim dengan error deskriptif saat aggregate transition `all-ready → any-down` terjadi.

**Implementasi:**
```go
setQueueReady := func(name string, ready bool) {
    c.queueStatus[name].Store(ready)

    allReady := true
    for _, n := range c.readyOrder {
        if !c.queueStatus[n].Load() {
            allReady = false
            break
        }
    }

    prev := aggregateState.Swap(allReady)
    if prev == allReady {
        return // no transition
    }
    if allReady {
        select { case c.chanOk <- struct{}{}: default: }
    } else {
        // transition from all-ready → any-down
        err := fmt.Errorf("consumer queue %q lost readiness", name)
        select { case c.chanErr <- err: default: }
    }
}
```

**Semantik `chanErr`:**
- Error yang dikirim berisi nama queue yang baru saja kehilangan readiness (penyebab transition)
- Buffer size 1 dengan non-blocking send — kalau caller belum sempat consume sinyal sebelumnya, signal baru di-drop (acceptable karena status real-time bisa di-query via `IsQueueReady`)
- `chanErr` tetap dipakai juga untuk Start() error (dari `eg.Wait()`) — caller tidak perlu bedakan sumbernya, keduanya berarti "ada masalah"

**Interaksi dengan Q1 (jawaban berpengaruh):**
- Kalau Q1 = continuous: setiap transition `any-down → all-up` fire `chanOk`, setiap `all-up → any-down` fire `chanErr`
- Kalau Q1 = 1-shot: `chanOk` fire sekali di awal saja, tapi `chanErr` tetap fire di setiap transition disconnect

### ✅ Q3. Ekspos status map — RESOLVED

**Keputusan:** tetap pakai `Status() (chan, chan)` **dan** tambah method `IsQueueReady(name string) bool` untuk query single queue.

**Interface update:**
```go
type Consumer interface {
    ConsumerBuilder
    Start(ctx context.Context) error
    Status() (ok chan struct{}, err chan error)
    IsQueueReady(name string) bool   // ◄── NEW
}
```

**Implementasi (amqp/cosumer.go):**
```go
func (c *csmr) IsQueueReady(name string) bool {
    flag, ok := c.queueStatus[name]
    if !ok { return false }
    return flag.Load()
}
```

Lock-free O(1) lookup karena:
- `c.queueStatus` map structure **immutable** setelah `Start()` populate
- Value adalah `*atomic.Bool` — `Load()` aman dari goroutine manapun

**Use-case yang di-enable:**
- HTTP liveness probe per queue (`/health/queue/:name`)
- Degraded-mode gating di business logic (critical queue down → fail fast)
- Per-queue Prometheus gauge untuk dashboard
- Startup sequence gating (Worker B wait queue X ready)

Method `QueueStatus() map[string]bool` (snapshot seluruh queue) bisa ditambahkan nanti kalau ada kebutuhan loop semua queue sekaligus — untuk sekarang single-query sudah cukup.

### ✅ Q4. `sync.Map` vs `map[string]*atomic.Bool` — RESOLVED

**Keputusan:** pakai `map[string]*atomic.Bool` yang di-populate sekali di awal `Start()` dari `c.stack`.

Alasan:
- Set key **fixed** di startup (queue names dari `c.stack`) — tidak ada add/remove setelah init
- Map structure tidak perlu locking karena tidak pernah mutasi struktural
- Setiap slot punya `*atomic.Bool` independen — lock-free flip
- Snapshot konsisten via iterasi `readyOrder` slice
- `sync.Map` optimized untuk "disjoint keys / add-once" pattern — bukan pattern kita

---

## Files That Will Change

- `higo/messagebroker/consumer/amqp/cosumer.go` — main refactor:
  - Tambah field `queueStatus map[string]*atomic.Bool` + `readyOrder []string` di struct `csmr`
  - Populate `queueStatus` di awal `Start()` dari `c.stack`
  - Pindahkan `chanOk` send dari goroutine-A ke dalam `startConsuming()` via `setQueueReady`
  - Tambah `defer setQueueReady(name, false)` di `startConsuming()` untuk handle disconnect
  - Implement method `IsQueueReady(name string) bool`
- `higo/messagebroker/consumer/consumer.go` — tambah method `IsQueueReady(name string) bool` di interface `Consumer`
- `higo/server/consumer/server.go` — tidak berubah (tetap select di `ok` dan `errChan`)

---

## Dependencies / Prior Context

Refactor ini lanjutan dari:
1. Sebelumnya: fix `fx.Shutdowner` di HTTP/gRPC/consumer servers supaya graceful shutdown saat salah satu crash
2. `running atomic.Bool` flag di HTTP/gRPC `OnStop` untuk skip `Shutdown`/`GracefulStop` kalau server gagal start
3. `higo/server/consumer/server.go` — fx.Invoke pattern yang blocking di `select` pada `c.Status()` sampai ready

---

## Full Picture — `higo/messagebroker/consumer/` Package

### Layout Direktori

```
higo/messagebroker/consumer/
├── consumer.go          ← public interfaces & types (TopologyConsumer, ConsumeHandler, etc)
├── topology.go          ← AMQP topology structs + WithAmqpTopology() builder
└── amqp/                ← AMQP-specific implementation
    ├── cosumer.go       ← csmr struct (main impl, Start/Consume/Status)
    ├── context.go       ← contextAmqp (per-message CtxConsumer impl, middleware chain)
    ├── cosumer_test.go
    └── context_test.go
```

### Type Hierarchy

```
┌─────────────────────────────────────────────────────────────────────┐
│  PUBLIC INTERFACES (consumer.go)                                    │
│                                                                     │
│  Consumer ──────────────┐                                           │
│    ├── ConsumerBuilder (embedded)                                   │
│    │     ├── Consume(queue, topology, ...handlers) ConsumerBuilder  │
│    │     ├── SimpleConsume(queue, ...handlers) ConsumerBuilder      │
│    │     └── Use(...handlers) ConsumerBuilder    ← global middleware│
│    ├── Start(ctx) error                                             │
│    └── Status() (ok chan struct{}, err chan error)                  │
│                                                                     │
│  ConsumerTopology = func() TopologyConsumer       ← builder func    │
│  ConsumeHandler   = func(CtxConsumer) error       ← handler/mw sig  │
│                                                                     │
│  CtxConsumer (per-message context, passed to handlers)              │
│    ├── Route() / UserContext() / SetUserContext(ctx)                │
│    ├── Body() / Header() / UpdateBody() / UpdateHeader()            │
│    └── Next() error                               ← invoke next mw  │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│  TOPOLOGY (topology.go)                                             │
│                                                                     │
│  TopologyConsumer                                                   │
│    └── Amqp AmqpTopologyConsumer                                    │
│          ├── AutoAck / Exclusive / NoLocal / NoWait / Durable ...   │
│          ├── PrefetchCount int64                                    │
│          ├── Args amqp091.Table                                     │
│          └── BindExchange *AmqpBindExchange                         │
│                ├── RoutingKey / ExchangeName / NoWait / Args        │
│                └── Exchange *AmqpTopologyConsumerExchange           │
│                      └── Kind / AutoDelete / Durable / Internal ... │
│                                                                     │
│  WithAmqpTopology(cfg) ConsumerTopology                             │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│  AMQP IMPLEMENTATION (amqp/cosumer.go)                              │
│                                                                     │
│  csmr (implements Consumer)                                         │
│    ├── man              manager.Manager[ConnectionAMQP]             │
│    ├── restartTime      time.Duration                               │
│    ├── mut              sync.Mutex        ← guards stack mutation   │
│    ├── stack            []amqpStack       ← registered queues       │
│    ├── globalMiddleware []ConsumeHandler  ← from Use()              │
│    ├── ctxPool          sync.Pool         ← contextAmqp reuse       │
│    ├── chanErr          chan error (buf:1)                          │
│    └── chanOk           chan struct{} (buf:1)                       │
│                                                                     │
│  amqpStack (per-queue registration)                                 │
│    ├── queueName        string                                      │
│    ├── consumerName     string                                      │
│    ├── topology         AmqpTopologyConsumer                        │
│    └── handlers         []ConsumeHandler   ← queue-specific mw+hdl  │
│                                                                     │
│  contextAmqp (implements CtxConsumer — per-message)                 │
│    ├── ctx / body / header / routeKey                               │
│    ├── stack *amqpStack                                             │
│    └── ciStackHandler, lenStackHandler ← middleware chain cursor    │
└─────────────────────────────────────────────────────────────────────┘
```

### Registration Flow (delivery-side)

```
adios/internals/delivery/consumer/routing.go
  NewRoutingConsumer(handler, builder) {
    builder.Use(middleware.OtelConsumerExtract())            ← global mw
    builder.Consume("queue_name",
                    WithAmqpTopology(AmqpTopologyConsumer{...}),
                    handler.HandleX)                         ← per-queue handler
  }

         ▼ calls into csmr.Consume() / .Use()

csmr.Use(handlers...)      → c.globalMiddleware = append(...)
csmr.Consume(q, topo, hs)  → c.stack = append(c.stack, amqpStack{...})
```

### Start Flow (current — buggy, see "Current Structure" section above)

```
higo/server/consumer/server.go — InvokeConsumerServer()
  └─ OnStart:
       c, err := param.Bk.Consumer(invCtx, ConsumeWithAmqp(...))
          │
          └─ broker/impl/amqp_con.go:openConnectionAmqp()
              ├─ amqp091.DialConfig()                    ← sync dial
              └─ go watchConnectionAmqp()
                    └─ man.SetCon() → readyChan<-{}      ← trigger ready
       param.Routing(c)                                  ← register queues
       go c.Start(invCtx)                                ← launch consumer
       ok, errChan := c.Status()
       select {
         case <-errChan: ...
         case <-ok:      ...                             ← unblock when ready
         case <-ctx.Done(): cancel()
       }
```

### Message Processing Flow (per message)

```
AMQP delivery arrives on `del` channel (inside startConsuming's for-select)
  │
  ▼
go c.processMessage(gctx, attrs, stack, msgDelivery)     ← one goroutine per msg
  │
  ├─ consumerCtx := c.ctxPool.Get()                      ← reuse context obj
  ├─ consumerCtx.populateContext(gctx, msgDelivery, stack)
  │    └─ sets body/header/route/handlers cursor
  ├─ consumerCtx.Next()                                  ← starts mw chain
  │    └─ handlers[0](ctx) → ctx.Next() → handlers[1](ctx) → ...
  └─ defer postProcesMessage()
       ├─ ctxPool.Put(ctx)                               ← return to pool
       ├─ if err != nil: Reject(tag, false)
       └─ else: Ack(tag, false)       (skipped if AutoAck)
```

### Middleware Chain via `Next()`

```
handlers = [globalMw1, globalMw2, queueMw1, queueHandler]
                                             ▲
                                     prepended during Consume()

Next() call sequence:
  ctx.Next() ─► handlers[0]=globalMw1(ctx)
                  │
                  ├─ do pre-work
                  ├─ ctx.Next() ─► handlers[1]=globalMw2(ctx)
                  │                  │
                  │                  ├─ do pre-work
                  │                  ├─ ctx.Next() ─► handlers[2]=queueMw1(ctx)
                  │                  │                  │
                  │                  │                  └─ ctx.Next() ─► handlers[3]=queueHandler(ctx)
                  │                  │                                     │
                  │                  │                                     └─ return nil (no more Next)
                  │                  └─ do post-work
                  └─ do post-work

Implemented in context.go:Next() via ciStackHandler cursor increment.
```

### Cross-Package Dependencies

```
higo/messagebroker/consumer/
   ↑ uses
   ├─ higo/messagebroker/manager/        (ConnectionAMQP, Manager[T])
   ├─ higo/messagebroker/manager/connections/     (channel abstraction)
   ├─ higo/utils/                        (OptionBool)
   └─ github.com/rabbitmq/amqp091-go     (underlying AMQP client)

   ↓ used by
   ├─ higo/messagebroker/broker/         (Consumer factory via Broker.Consumer())
   ├─ higo/middleware/                   (OtelConsumerExtract — ConsumeHandler impl)
   └─ higo/server/consumer/              (fx lifecycle wrapper)
```

### Key Observations Relevant to Refactor

1. **`chanOk` / `chanErr` buffer size = 1** dengan non-blocking send pattern (`select { case ... <- x: default: }`) — artinya kalau sudah full, signal berikutnya **silently dropped**. Cocok untuk 1-shot semantic, kurang cocok untuk continuous monitoring.

2. **`c.stack` di-access tanpa mutex di `Start()`** setelah registration phase selesai. Kalau mau tambah method query status, harus hati-hati jangan mutasi `c.stack`.

3. **Queue name adalah natural key** untuk per-queue status map — sudah tersimpan di `amqpStack.queueName`, tinggal dipakai.

4. **Handler chain immutable setelah Consume()** — `compiledStack` di-build sekali di awal `Start()` goroutine-A. Tidak ada concurrent mutation risk untuk handler slice setelah itu.

5. **`ctxPool` pakai `sync.Pool`** untuk reuse `contextAmqp` — jangan sampai refactor mengubah lifecycle ini (bisa leak atau reuse data antar goroutine).

6. **`buildAndConsume` dipanggil per-queue dalam goroutine terpisah** — ini lokasi natural untuk call `setQueueReady(name, true)` setelah subscribe sukses, dan `setQueueReady(name, false)` saat return.
