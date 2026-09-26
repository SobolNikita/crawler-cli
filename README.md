# crawler-cli

CLI на Go для асинхронного обхода сайтов. Строит дерево страниц и сохраняет его в JSON.

## Требования

- Go 1.22+

## Сборка

```bash
go build -o crawler-cli ./cmd/crawler-cli
```

На Windows:

```bash
go build -o crawler-cli.exe ./cmd/crawler-cli
```

## Запуск

```bash
./crawler-cli \
  --urls https://example.com,https://golang.org \
  --depth 2 \
  --timeout 2m \
  --request-timeout 10s \
  --output result.json \
  --log crawler.log
```

### Параметры

| Флаг | Описание | По умолчанию |
|------|----------|--------------|
| `--urls` | Стартовые URL через запятую | — (обязательный) |
| `--depth` | Максимальная глубина обхода | `0` |
| `--timeout` | Общий таймаут выполнения | `2m` |
| `--request-timeout` | Таймаут одного HTTP-запроса | `10s` |
| `--output` | Файл с результатом (JSON) | `output.json` |
| `--log` | Файл логов (статусы и ошибки) | `log.txt` |

Остановка по `Ctrl+C` — корректное завершение через context.

## Тесты

```bash
go test ./...
```

## Структура

```
cmd/crawler-cli/   — точка входа
internal/config/   — флаги CLI
internal/crawler/  — обход (workers, channels)
internal/fetcher/  — HTTP-запросы
internal/parser/   — title и ссылки из HTML
internal/logger/   — запись в лог-файл
internal/model/    — модель Page для JSON
```
