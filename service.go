package regru

import (
	"context"
	"net/url"
)

// Типы услуг для ListServices (часть списка из документации).
const (
	ServiceDomain  = "domain"
	ServiceWebFwd  = "srv_webfwd"
	ServiceParking = "srv_parking"
	ServiceDNS     = "srv_dns_both"
	ServiceHosting = "srv_hosting_ispmgr"
	ServiceCert    = "srv_certificate"
)

// Состояния услуги (Service.State).
const (
	StateInactive    = "N" // не активна: домен не зарегистрирован или не перенесён
	StateActive      = "A"
	StateSuspended   = "S"
	StateDeleted     = "D"
	StateTransferred = "O" // домен перенесён к другому регистратору
)

// maxObjects — предел REG.API на число услуг в одном запросе.
const maxObjects = 1000

// Service — услуга или домен аккаунта.
type Service struct {
	ID             Num    `json:"service_id"`
	Domain         string `json:"dname"`
	Type           string `json:"servtype"`
	Subtype        string `json:"subtype"`
	State          string `json:"state"`
	CreationDate   string `json:"creation_date"`
	ExpirationDate string `json:"expiration_date"`
	UplinkID       Num    `json:"uplink_service_id"`

	// Err не nil, если по этой услуге в ServiceInfo пришла ошибка
	// (например, DOMAIN_NOT_FOUND). Остальные поля тогда пусты.
	Err error `json:"-"`
}

// ListServices возвращает активные услуги аккаунта. Пустой servtype — все виды.
func (c *Client) ListServices(ctx context.Context, servtype string) ([]Service, error) {
	p := url.Values{}
	if servtype != "" {
		p.Set("servtype", servtype)
	}
	var ans struct {
		Services []Service `json:"services"`
	}
	if err := c.call(ctx, "service/get_list", p, nil, &ans); err != nil {
		return nil, err
	}
	return ans.Services, nil
}

// ListDomains возвращает домены аккаунта.
func (c *Client) ListDomains(ctx context.Context) ([]Service, error) {
	return c.ListServices(ctx, ServiceDomain)
}

// ServiceInfo возвращает состояние и сроки услуг по именам доменов. Запросы
// сверх лимита API режутся на пачки по 1000. Порядок результата повторяет
// порядок domains; ошибка по отдельному домену лежит в Service.Err.
func (c *Client) ServiceInfo(ctx context.Context, domains ...string) ([]Service, error) {
	out := make([]Service, 0, len(domains))
	for start := 0; start < len(domains); start += maxObjects {
		end := min(start+maxObjects, len(domains))
		items := make([]map[string]string, 0, end-start)
		for _, d := range domains[start:end] {
			items = append(items, map[string]string{"domain_name": d})
		}
		var ans struct {
			Services []struct {
				Service
				itemResult
			} `json:"services"`
		}
		if err := c.call(ctx, "service/get_info", nil, map[string]any{"services": items}, &ans); err != nil {
			return nil, err
		}
		for _, s := range ans.Services {
			s.Service.Err = s.itemResult.err()
			out = append(out, s.Service)
		}
	}
	return out, nil
}
