package regru

import (
	"context"
	"errors"
)

// ErrNotRegruNS возвращается для домена, у которого не NS-серверы Рег.ру:
// его DNS-зоной через API управлять нельзя.
var ErrNotRegruNS = &APIError{Code: "DOMAIN_IS_NOT_USE_REGRU_NSS"}

// ZoneRecord — ресурсная запись DNS-зоны.
type ZoneRecord struct {
	Sub      string `json:"subname"` // "@" — сам домен, "*" — все остальные поддомены
	Type     string `json:"rectype"` // A, AAAA, CNAME, MX, ...
	Content  string `json:"content"`
	Priority Num    `json:"prio"`
}

// Zone — записи зоны одного домена.
type Zone struct {
	Domain    string       `json:"dname"`
	ServiceID Num          `json:"service_id"`
	Records   []ZoneRecord `json:"rrs"`

	// Err не nil, если зоной этого домена нельзя управлять
	// (например, ErrNotRegruNS или ErrDomainNotFound).
	Err error `json:"-"`
}

// ZoneRecords возвращает записи DNS-зон доменов. Запросы сверх лимита API
// режутся на пачки по 1000. Порядок результата повторяет порядок domains;
// ошибка по отдельному домену лежит в Zone.Err.
func (c *Client) ZoneRecords(ctx context.Context, domains ...string) ([]Zone, error) {
	out := make([]Zone, 0, len(domains))
	for start := 0; start < len(domains); start += maxObjects {
		end := min(start+maxObjects, len(domains))
		items := make([]map[string]string, 0, end-start)
		for _, d := range domains[start:end] {
			items = append(items, map[string]string{"dname": d})
		}
		var ans struct {
			Domains []struct {
				Zone
				itemResult
			} `json:"domains"`
		}
		if err := c.call(ctx, "zone/get_resource_records", nil, map[string]any{"domains": items}, &ans); err != nil {
			return nil, err
		}
		for _, d := range ans.Domains {
			d.Zone.Err = d.itemResult.err()
			out = append(out, d.Zone)
		}
	}
	return out, nil
}

// AddARecord добавляет A-запись: subdomain "@" — для самого домена, "www" — для
// поддомена. Метод не проверяет существующие записи: сначала посмотрите их через
// ZoneRecords, иначе можно получить дубль.
func (c *Client) AddARecord(ctx context.Context, domain, subdomain, ip string) error {
	in := map[string]any{
		"domains":   []map[string]string{{"dname": domain}},
		"subdomain": subdomain,
		"ipaddr":    ip,
	}
	var ans struct {
		Domains []itemResult `json:"domains"`
	}
	if err := c.call(ctx, "zone/add_alias", nil, in, &ans); err != nil {
		return err
	}
	if len(ans.Domains) == 0 {
		return errors.New("regru: zone/add_alias: empty answer")
	}
	return ans.Domains[0].err()
}
