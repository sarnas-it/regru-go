# regru-go

Go-клиент [REG.API 2](https://www.reg.ru/reseller/api2doc) — API регистратора Рег.ру.
Только стандартная библиотека, без зависимостей.

Это неофициальная библиотека, к Рег.ру она отношения не имеет.

## Что умеет

Версия 0.1 — только чтение:

| Метод клиента | Метод REG.API | Что делает |
|---|---|---|
| `Nop` | `nop` | проверка логина, пароля и IP |
| `Balance` | `user/get_balance` | баланс счёта |
| `ListServices`, `ListDomains` | `service/get_list` | услуги и домены аккаунта |
| `ServiceInfo` | `service/get_info` | состояние и сроки по именам доменов |
| `DomainPrices` | `domain/get_prices` | цены регистрации и продления по зонам |
| `LotList`, `EachLot` | `shop/get_lot_list` | лоты Магазина доменов (в том числе только свои) |
| `LotInfo` | `shop/get_info` | подробности лота |
| `Categories` | `shop/get_category_list` | категории лотов |
| `SuggestedTags` | `shop/get_suggested_tags` | популярные теги |

Методы, которые меняют данные или тратят деньги (регистрация, продление, удаление лота), не реализованы.

## Установка

```
go get github.com/sarnas-it/regru-go
```

## Пример

```go
c := regru.New(login, password, regru.WithLang("ru"))

domains, err := c.ListDomains(ctx)

// все свои лоты в магазине
err = c.EachLot(ctx, regru.LotListParams{MyLots: true, PerPage: 100}, func(l regru.Lot) error {
	fmt.Println(l.Domain, l.StartPrice, l.PriceType)
	return nil
})
```

Полный пример: [`examples/mylots`](examples/mylots).

## Перед использованием

- **Список IP.** Доступ к API работает только с адресов, добавленных в «Настройки API» в личном кабинете. С другого адреса вернётся `regru.ErrAccessDeniedFromIP`.
- **Лимит.** 1200 запросов в час на аккаунт и на IP. Клиент его не считает: при превышении вернётся `ErrIPRateLimit` или `ErrAccountRateLimit`. Рег.ру советует хранить данные о доменах у себя и ходить в API только за изменениями.
- **Тестовый режим.** Логин и пароль `test`/`test` проверяют параметры, но реальных данных не возвращают.
- **Клиентский сертификат.** Если в настройках API загружен SSL-сертификат, передайте его через `regru.WithClientCertificate`. Аутентификация по подписи (`sig`) пока не поддерживается.

## Ошибки

Ошибка API приходит как `*regru.APIError` с полем `Code`. Сравнивать удобно через `errors.Is`:

```go
if errors.Is(err, regru.ErrAccessDeniedFromIP) { ... }
```

В `ServiceInfo` ошибка по отдельному домену лежит в `Service.Err`, а общий вызов при этом считается успешным.

Числа REG.API присылает то строкой, то числом, поэтому цены и счётчики имеют тип `regru.Num` (строка с методами `Float64()` и `Int()`).

## Лицензия

MIT
