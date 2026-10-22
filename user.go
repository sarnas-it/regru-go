package regru

import (
	"context"
	"net/url"
)

// Account — ответ функции nop.
type Account struct {
	Login  string `json:"login"`
	UserID Num    `json:"user_id"`
}

// Nop проверяет логин, пароль и доступ с текущего IP, ничего не меняя.
func (c *Client) Nop(ctx context.Context) (*Account, error) {
	var a Account
	if err := c.call(ctx, "nop", nil, nil, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// Balance — сумма на лицевом счёте.
type Balance struct {
	Currency string `json:"currency"`
	Prepay   Num    `json:"prepay"`
	Blocked  Num    `json:"blocked"`
	Credit   Num    `json:"credit"`
}

// Balance возвращает баланс. Валюта: RUR, USD, EUR, UAH; пусто — рубли.
func (c *Client) Balance(ctx context.Context, currency string) (*Balance, error) {
	p := url.Values{}
	if currency != "" {
		p.Set("currency", currency)
	}
	var b Balance
	if err := c.call(ctx, "user/get_balance", p, nil, &b); err != nil {
		return nil, err
	}
	return &b, nil
}
