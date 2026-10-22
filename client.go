// Пакет regru — клиент REG.API 2 (https://www.reg.ru/reseller/api2doc).
//
// Покрывает методы чтения: баланс, список услуг и доменов, информация о них,
// цены доменных зон и лоты Магазина доменов. Зависимостей, кроме стандартной
// библиотеки, нет.
//
// Доступ к API ограничен по IP: адрес сервера нужно добавить в «Настройки API»
// в личном кабинете, иначе все вызовы вернут ErrAccessDeniedFromIP.
package regru

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultBaseURL — боевой адрес REG.API 2.
	DefaultBaseURL = "https://api.reg.ru/api/regru2"

	maxResponseBytes = 32 << 20
)

// Client — клиент REG.API. Безопасен для использования из нескольких горутин.
//
// У API есть лимит 1200 запросов в час на аккаунт и на IP. Клиент его не
// отслеживает: при превышении вернётся ErrIPRateLimit или ErrAccountRateLimit.
type Client struct {
	username string
	password string
	baseURL  string
	lang     string
	http     *http.Client
}

// Option настраивает Client.
type Option func(*Client)

// WithHTTPClient подменяет http.Client (прокси, таймауты, трассировка).
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

// WithBaseURL меняет адрес API (для тестов).
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") } }

// WithLang задаёт язык текстов ошибок: "ru", "en" или "th". Коды ошибок не меняются.
func WithLang(lang string) Option { return func(c *Client) { c.lang = lang } }

// WithClientCertificate добавляет клиентский TLS-сертификат. Он нужен, если в
// настройках API загружен хотя бы один SSL-сертификат.
func WithClientCertificate(cert tls.Certificate) Option {
	return func(c *Client) {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.TLSClientConfig = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
		c.http = &http.Client{Transport: t, Timeout: c.http.Timeout}
	}
}

// New создаёт клиента с аутентификацией по логину и паролю. Логин и пароль
// "test"/"test" включают тестовый режим: данные не возвращаются, деньги не
// списываются.
func New(username, password string, opts ...Option) *Client {
	c := &Client{
		username: username,
		password: password,
		baseURL:  DefaultBaseURL,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// envelope — общая обёртка ответа.
type envelope struct {
	Result    string          `json:"result"`
	Answer    json.RawMessage `json:"answer"`
	ErrorCode string          `json:"error_code"`
	ErrorText string          `json:"error_text"`
	ErrorPrms json.RawMessage `json:"error_params"`
}

// call выполняет POST на <baseURL>/<method>. Простые параметры идут как form,
// сложные (input) — одним JSON-полем input_data, без учётных данных внутри.
func (c *Client) call(ctx context.Context, method string, params url.Values, input any, out any) error {
	form := url.Values{}
	for k, vs := range params {
		form[k] = vs
	}
	form.Set("username", c.username)
	form.Set("password", c.password)
	form.Set("output_format", "json")
	if c.lang != "" {
		form.Set("lang", c.lang)
	}
	if input != nil {
		b, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("regru: encode input: %w", err)
		}
		form.Set("input_format", "json")
		form.Set("input_data", string(b))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/"+method, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("regru: %s: %w", method, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("regru: %s: read body: %w", method, err)
	}

	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("regru: %s: HTTP %d, unexpected response: %w", method, resp.StatusCode, err)
	}
	if env.Result != "success" {
		code := env.ErrorCode
		if code == "" {
			code = fmt.Sprintf("HTTP_%d", resp.StatusCode)
		}
		return &APIError{Code: code, Text: env.ErrorText, Params: env.ErrorPrms}
	}
	if out == nil || len(env.Answer) == 0 {
		return nil
	}
	if err := json.Unmarshal(env.Answer, out); err != nil {
		return fmt.Errorf("regru: %s: decode answer: %w", method, err)
	}
	return nil
}
