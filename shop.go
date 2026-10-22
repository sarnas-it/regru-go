package regru

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Типы цены лота (Lot.PriceType).
const (
	PriceFixed = "fixed" // фиксированная цена
	PriceOffer = "offer" // договорная
)

// Lot — лот Магазина доменов. Часть полей приходит только в списке, часть —
// только из LotInfo.
type Lot struct {
	Domain            string `json:"dname"`
	DomainPuny        string `json:"dname_puny"`
	PriceType         string `json:"price_type"`
	StartPrice        Num    `json:"start_price"`
	WithSite          Flag   `json:"site"`
	WithTrademark     Flag   `json:"tm"`
	Online            Flag   `json:"is_online"`
	Description       string `json:"description"`
	Keywords          List   `json:"keywords"`
	CategoryIDs       List   `json:"category_ids"`
	Hits              Num    `json:"lot_hits"`
	BidsCount         Num    `json:"bids_cnt"`
	FirstCreateDomain string `json:"first_create_domain"`
	CreationDate      string `json:"creation_date"`
	LotDate           string `json:"lot_date"`
	LotModified       string `json:"lot_mtime"`
	YandexTIC         Num    `json:"yandex_tic"`
	GooglePR          Num    `json:"google_pr"`
}

// Допустимые размеры страницы для LotList.
var lotPageSizes = map[int]bool{25: true, 50: true, 100: true, 200: true, 500: true}

// LotListParams — фильтры и пагинация LotList. Нулевое значение поля означает
// «не задано».
type LotListParams struct {
	Page      int    // страница с нуля
	PerPage   int    // 25 (по умолчанию), 50, 100, 200 или 500
	SortCol   string // dname, dname_length, start_price, lot_date, lot_hits, bids_cnt и др.
	SortDesc  bool
	Keywords  string
	MyLots    bool // только лоты вашего аккаунта
	PriceFrom int  // в рублях
	PriceTo   int
	LenFrom   int // длина имени домена
	LenTo     int
}

func (p LotListParams) values() (url.Values, error) {
	if p.PerPage != 0 && !lotPageSizes[p.PerPage] {
		return nil, fmt.Errorf("regru: PerPage must be one of 25, 50, 100, 200, 500, got %d", p.PerPage)
	}
	v := url.Values{}
	setInt := func(k string, n int) {
		if n > 0 {
			v.Set(k, strconv.Itoa(n))
		}
	}
	v.Set("pg", strconv.Itoa(p.Page))
	setInt("itemsonpage", p.PerPage)
	if p.SortCol != "" {
		v.Set("sortcol", p.SortCol)
		if p.SortDesc {
			v.Set("sortorder", "DESC")
		} else {
			v.Set("sortorder", "ASC")
		}
	}
	if p.Keywords != "" {
		v.Set("keywords", p.Keywords)
	}
	if p.MyLots {
		v.Set("show_my_lots", "1")
	}
	setInt("price_from", p.PriceFrom)
	setInt("price_to", p.PriceTo)
	setInt("dname_length_from", p.LenFrom)
	setInt("dname_length_to", p.LenTo)
	return v, nil
}

// LotList возвращает одну страницу лотов Магазина доменов.
func (c *Client) LotList(ctx context.Context, p LotListParams) ([]Lot, error) {
	v, err := p.values()
	if err != nil {
		return nil, err
	}
	var ans struct {
		Lots []Lot `json:"lots"`
	}
	if err := c.call(ctx, "shop/get_lot_list", v, nil, &ans); err != nil {
		return nil, err
	}
	return ans.Lots, nil
}

// EachLot обходит все страницы, начиная с p.Page, и вызывает fn для каждого
// лота. Обход заканчивается на неполной странице или при ошибке fn.
func (c *Client) EachLot(ctx context.Context, p LotListParams, fn func(Lot) error) error {
	size := p.PerPage
	if size == 0 {
		size = 25
	}
	for {
		lots, err := c.LotList(ctx, p)
		if err != nil {
			return err
		}
		for _, l := range lots {
			if err := fn(l); err != nil {
				return err
			}
		}
		if len(lots) < size {
			return nil
		}
		p.Page++
	}
}

// LotInfo возвращает подробности лота по имени домена.
func (c *Client) LotInfo(ctx context.Context, domain string) (*Lot, error) {
	var l Lot
	if err := c.call(ctx, "shop/get_info", url.Values{"dname": {domain}}, nil, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// Category — категория лотов; у верхнего уровня есть подкатегории.
type Category struct {
	ID            Num        `json:"id"`
	Name          string     `json:"category_name"`
	NameEN        string     `json:"category_name_en"`
	Subcategories []Category `json:"subcategories"`
}

// Categories возвращает дерево категорий лотов.
func (c *Client) Categories(ctx context.Context) ([]Category, error) {
	var ans struct {
		List []Category `json:"category_list"`
	}
	if err := c.call(ctx, "shop/get_category_list", nil, nil, &ans); err != nil {
		return nil, err
	}
	return ans.List, nil
}

// SuggestedTags возвращает популярные теги лотов (limit: по умолчанию 10, максимум 50).
func (c *Client) SuggestedTags(ctx context.Context, limit int) ([]string, error) {
	v := url.Values{}
	if limit > 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	var ans struct {
		Tags []string `json:"tags"`
	}
	if err := c.call(ctx, "shop/get_suggested_tags", v, nil, &ans); err != nil {
		return nil, err
	}
	return ans.Tags, nil
}
