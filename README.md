# Практична робота № 4 — Device Registry API

REST API реєстру пристроїв на Go та Chi. Реалізовано три рівні: CRUD, потокобезпечне сховище, логування, фільтрацію, JSON-помилки, Panic Recovery і CORS.

## Запуск

Потрібен Go 1.24 або новіший (вимога встановленої версії Chi).

```bash
go run ./src
```

API: `http://localhost:8080/api/v1/devices`. Дані зберігаються у пам’яті та зникають після перезапуску.

## Структура

- `src/main.go` — запуск сервера на порту 8080.
- `src/internal/domain` — модель Device.
- `src/internal/repository` — інтерфейс та in-memory сховище з `sync.RWMutex`.
- `src/internal/handler` — маршрути Chi, HTTP-обробники, middleware та JSON-помилки.

## Маршрути

| Метод | Шлях | Результат |
| --- | --- | --- |
| GET | `/api/v1/devices` | Список, 200; порожній список — `[]` |
| POST | `/api/v1/devices` | Створення, 201 та заголовок Location |
| GET | `/api/v1/devices/{id}` | Пристрій, 200 або 404 |
| PUT | `/api/v1/devices/{id}` | Повна заміна даних, 200 або 404 |
| DELETE | `/api/v1/devices/{id}` | Видалення, 204 без тіла або 404 |
| OPTIONS | `/api/v1/devices` | CORS preflight, 204 |

Фільтри `status` і `type` працюють окремо або разом (логічне І). Тип — довільний непорожній рядок, наприклад `sensor`, `gateway`, `controller`. Невідоме значення фільтра дає порожній список.

POST і PUT вимагають непорожні `name`, `ip_address`, `type`. IP має бути коректною IPv4/IPv6-адресою. `status` — `online` або `offline`, за відсутності встановлюється `offline`. Сервер генерує випадковий ID під час створення; під час PUT використовує ID з URL. `updated_at` встановлюється сервером у UTC. Некоректний JSON, декілька JSON-значень або невалідні поля дають 400. Ліміт тіла запиту — 1 MiB.

## Приклади перевірки

Створити пристрій:

```bash
curl -i -X POST http://localhost:8080/api/v1/devices \
  -H 'Content-Type: application/json' \
  -d '{"name":"Main Thermal Sensor","ip_address":"192.168.1.50","type":"sensor","status":"online"}'
```

Скопіюйте отриманий `id` у змінну:

```bash
DEVICE_ID='dev-ідентифікатор-з-відповіді'
curl -i "http://localhost:8080/api/v1/devices/$DEVICE_ID"
curl -i 'http://localhost:8080/api/v1/devices?status=online&type=sensor'
curl -i -X PUT "http://localhost:8080/api/v1/devices/$DEVICE_ID" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Updated Sensor","ip_address":"192.168.1.51","type":"sensor","status":"offline"}'
curl -i -X DELETE "http://localhost:8080/api/v1/devices/$DEVICE_ID"
curl -i "http://localhost:8080/api/v1/devices/$DEVICE_ID"
```

Очікувані статуси: 200 → 200 → 200 → 204 → 404.

Перевірити валідацію (400) та CORS (204):

```bash
curl -i -X POST http://localhost:8080/api/v1/devices \
  -H 'Content-Type: application/json' -d '{}'
curl -i -X OPTIONS http://localhost:8080/api/v1/devices \
  -H 'Origin: https://example.com' \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: Content-Type, Authorization'
```

## Middleware та помилки

Chi Logger записує метод, адресу клієнта, шлях, статус і тривалість запиту. RequestID додає ідентифікатор запиту. Власний Recoverer перехоплює паніку, записує стек у журнал та повертає JSON з кодом 500. CORS дозволяє всі джерела, методи `GET, POST, PUT, DELETE, OPTIONS` і заголовки `Content-Type, Authorization`.

Помилки валідації, відсутні ресурси/маршрути, недозволені методи та паніки мають єдиний формат з `Content-Type: application/json`:

```json
{
  "error": "Not Found",
  "message": "Device with ID \"unknown-device\" does not exist in registry",
  "status": 404,
  "timestamp": "2026-10-29T12:00:00Z"
}
```

## Автоматична перевірка

```bash
go test -race ./...
go vet ./...
```

Тести перевіряють повний CRUD-цикл, генерацію ID, фільтри, валідацію POST/PUT, JSON-помилки, CORS, відновлення після паніки та конкурентний доступ до репозиторію.

