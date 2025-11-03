# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m main template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/main .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведенная в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Профайлинг

Diff:

```
Type: alloc_space
Time: 2025-11-03 16:50:04 MSK
Showing nodes accounting for -561.96MB, 37.21% of 1510.12MB total
Dropped 22 nodes (cum <= 7.55MB)
      flat  flat%   sum%        cum   cum%
 -184.18MB 12.20% 12.20%  -184.18MB 12.20%  text/template.addValueFuncs
 -154.60MB 10.24% 22.43%  -154.60MB 10.24%  maps.Copy[go.shape.map[string]html/template.context,go.shape.map[string]html/template.context,go.shape.string,go.shape.struct { html/template.state html/template.state; html/template.delim html/template.delim; html/template.urlPart html/template.urlPart; html/template.jsCtx html/template.jsCtx; html/template.jsBraceDepth []int; html/template.attr html/template.attr; html/template.element html/template.element; html/template.n text/template/parse.Node; html/template.err *html/template.Error }] (inline)
 -144.61MB  9.58% 32.01%  -144.61MB  9.58%  text/template.addFuncs (inline)
  108.03MB  7.15% 24.86%   108.03MB  7.15%  net/http.Header.Clone (inline)
   89.03MB  5.90% 30.76%    89.03MB  5.90%  net/textproto.MIMEHeader.Set (inline)
  -79.02MB  5.23% 35.99%   -79.02MB  5.23%  html/template.(*escaper).editActionNode
   76.52MB  5.07% 31.07%    76.52MB  5.07%  bytes.growSlice
  -71.50MB  4.73% 35.80%   -71.50MB  4.73%  html/template.makeEscaper (inline)
 -696.48MB 46.12% 81.92%  -696.48MB 46.12%  go-metrics-and-alerts/internal/handler.(*Handler).ListMetrics
```

