package regru

import (
	"context"
	"net/url"
)

// ZonePrice — цены одной доменной зоны (за год; для части зон минимальный срок больше).
type ZonePrice struct {
	RegPrice         Num `json:"reg_price"`
	RenewPrice       Num `json:"renew_price"`
	RetailRegPrice   Num `json:"retail_reg_price"`
	RetailRenewPrice Num `json:"retail_renew_price"`
	RegMinPeriod     Num `json:"reg_min_period"`
	RegMaxPeriod     Num `json:"reg_max_period"`
}

// Prices — прайс-лист регистратора.
type Prices struct {
	Currency       string               `json:"currency"`
	PriceGroup     string               `json:"price_group"`
	LastUpdateTime string               `json:"last_update_time"`
	Zones          map[string]ZonePrice `json:"prices"`
}

// IDNPrefix — префикс ключа в Prices.Zones для кириллических имён, если их
// цена отличается: "__idn.com".
const IDNPrefix = "__idn."

// PricesOptions — необязательные параметры DomainPrices.
type PricesOptions struct {
	Currency  string // RUR (по умолчанию), UAH, USD, EUR
	WithRenew bool   // добавить цены продления
}

// DomainPrices возвращает цены регистрации и продления по всем зонам. Метод
// общедоступный; с учётными данными клиента цены считаются по вашему тарифу.
func (c *Client) DomainPrices(ctx context.Context, o PricesOptions) (*Prices, error) {
	p := url.Values{}
	if o.Currency != "" {
		p.Set("currency", o.Currency)
	}
	if o.WithRenew {
		p.Set("show_renew_data", "1")
	}
	var pr Prices
	if err := c.call(ctx, "domain/get_prices", p, nil, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}
