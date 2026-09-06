# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Сборка с информацией о версии

Пакеты `cmd/server` и `cmd/agent` содержат переменные `buildVersion`, `buildDate` и `buildCommit`, которые по умолчанию равны `"N/A"`. Их значения можно задать на этапе компиляции через флаг `-ldflags` и опцию `-X`:

```bash
go build -ldflags "-X main.buildVersion=v1.0.0 -X main.buildDate=$(date +'%Y/%m/%d %H:%M:%S') -X main.buildCommit=$(git rev-parse HEAD)" -o cmd/server/server ./cmd/server
```
```bash
go build -ldflags "-X main.buildVersion=v1.0.0 -X main.buildDate=$(date +'%Y/%m/%d %H:%M:%S') -X main.buildCommit=$(git rev-parse HEAD)" -o cmd/agent/agent ./cmd/agent
```

При запуске бинарник выведет эти значения в stdout:

```
Build version: v1.0.0
Build date: 2026/09/06 12:00:00
Build commit: a1b2c3d4e5f6...
```

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

# Результаты оптимизации инкремента 17

```
File: server
Type: alloc_space
Time: 2026-08-22 09:59:51 +03
Showing nodes accounting for -2532.57MB, 14.52% of 17437.20MB total
Dropped 91 nodes (cum <= 87.19MB)
      flat  flat%   sum%        cum   cum%
-4309.75MB 24.72% 24.72% -4336.25MB 24.87%  encoding/json.(*Decoder).refill
 2996.09MB 17.18%  7.53%  3024.59MB 17.35%  io.ReadAll
 -647.70MB  3.71% 11.25%  -647.70MB  3.71%  encoding/json.NewDecoder (inline)
  277.04MB  1.59%  9.66%  5352.25MB 30.69%  encoding/json.Unmarshal
 -244.63MB  1.40% 11.06%  -244.63MB  1.40%  reflect.growslice
 -168.55MB  0.97% 12.03%  -168.55MB  0.97%  net/http.(*Request).WithContext (inline)
  -77.02MB  0.44% 12.47%   -77.02MB  0.44%  net/textproto.readMIMEHeader
  -54.51MB  0.31% 12.78% -2235.56MB 12.82%  main.main.WithLogging.func2.1
  -48.51MB  0.28% 13.06%  -128.54MB  0.74%  net/http.readRequest
  -33.50MB  0.19% 13.25%   -33.50MB  0.19%  github.com/LifeforDream/gometrics/internal/middlewares/mwhash.newHashWriter
  -32.50MB  0.19% 13.44%   -32.50MB  0.19%  reflect.unsafe_New
  -30.50MB  0.17% 13.61%   -30.50MB  0.17%  encoding/json.(*scanner).pushParseState
  -29.51MB  0.17% 13.78%  -168.06MB  0.96%  net/http.(*conn).readRequest
     -25MB  0.14% 13.93%      -25MB  0.14%  net/http.Header.Clone (inline)
     -24MB  0.14% 14.07%      -24MB  0.14%  compress/gzip.NewWriterLevel
  -17.50MB   0.1% 14.17%   -17.50MB   0.1%  sync.(*poolChain).pushHead
  -17.50MB   0.1% 14.27%   -17.50MB   0.1%  context.withCancel (inline)
  -15.50MB 0.089% 14.35%   -15.50MB 0.089%  github.com/LifeforDream/gometrics/internal/middlewares/mwcompress.newCompressWriter
  -13.50MB 0.077% 14.43%   -35.50MB   0.2%  encoding/json.(*decodeState).object
     -13MB 0.075% 14.51%      -13MB 0.075%  net.(*conn).Read
     -13MB 0.075% 14.58%      -13MB 0.075%  net/textproto.(*Reader).ReadLine (inline)
   10.50MB  0.06% 14.52%      -22MB  0.13%  encoding/json.(*decodeState).literalStore
      -2MB 0.011% 14.53%      -26MB  0.15%  github.com/LifeforDream/gometrics/internal/compress.NewWriter
       1MB 0.0057% 14.53% -2510.58MB 14.40%  net/http.(*conn).serve
    0.50MB 0.0029% 14.52% -2016.45MB 11.56%  github.com/LifeforDream/gometrics/internal/handler.(*Handler).UpdateMetrics
         0     0% 14.52% -9722.09MB 55.75%  encoding/json.(*Decoder).Decode
         0     0% 14.52% -4402.25MB 25.25%  encoding/json.(*Decoder).readValue
         0     0% 14.52%  -315.63MB  1.81%  encoding/json.(*decodeState).array
         0     0% 14.52%   -37.50MB  0.22%  encoding/json.(*decodeState).scanWhile
         0     0% 14.52%  -317.63MB  1.82%  encoding/json.(*decodeState).unmarshal
         0     0% 14.52%  -315.63MB  1.81%  encoding/json.(*decodeState).value
         0     0% 14.52%       73MB  0.42%  encoding/json.checkValid
         0     0% 14.52%   -32.50MB  0.19%  encoding/json.indirect
         0     0% 14.52%   -30.50MB  0.17%  encoding/json.stateBeginValue
         0     0% 14.52%      -35MB   0.2%  encoding/json.stateBeginValueOrEmpty
         0     0% 14.52%      -25MB  0.14%  github.com/LifeforDream/gometrics/internal/middlewares/logs.(*loggingResponseWriter).WriteHeader
         0     0% 14.52%      -25MB  0.14%  github.com/LifeforDream/gometrics/internal/middlewares/mwcompress.(*compressWriter).WriteHeader
         0     0% 14.52%      -25MB  0.14%  github.com/LifeforDream/gometrics/internal/middlewares/mwhash.(*hashWriter).WriteHeader
         0     0% 14.52% -2181.05MB 12.51%  github.com/LifeforDream/gometrics/internal/middlewares/mwip.WithClientIP.func1
         0     0% 14.52% -2338.09MB 13.41%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP
         0     0% 14.52% -2013.02MB 11.54%  github.com/go-chi/chi/v5.(*Mux).routeHTTP
         0     0% 14.52% -2054.52MB 11.78%  github.com/go-chi/chi/v5/middleware.StripSlashes.func1
         0     0% 14.52%    14.09MB 0.081%  io.Copy (inline)
         0     0% 14.52%    14.09MB 0.081%  io.copyBuffer
         0     0% 14.52%    14.09MB 0.081%  io.discard.ReadFrom
         0     0% 14.52% -2054.52MB 11.78%  main.main.Compress.func4.1
         0     0% 14.52% -2090.52MB 11.99%  main.main.WithHash.func3.1
         0     0% 14.52%      -13MB 0.075%  net/http.(*connReader).backgroundRead
         0     0% 14.52%      -25MB  0.14%  net/http.(*response).WriteHeader
         0     0% 14.52% -2235.56MB 12.82%  net/http.HandlerFunc.ServeHTTP
         0     0% 14.52% -2338.09MB 13.41%  net/http.serverHandler.ServeHTTP
         0     0% 14.52%   -77.02MB  0.44%  net/textproto.(*Reader).ReadMIMEHeader (inline)
         0     0% 14.52%   -32.50MB  0.19%  reflect.New
         0     0% 14.52%  -244.63MB  1.40%  reflect.Value.Grow
         0     0% 14.52%  -244.63MB  1.40%  reflect.Value.grow
         0     0% 14.52%      -15MB 0.086%  sync.(*Pool).Put
```