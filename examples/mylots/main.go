// Команда mylots печатает домены аккаунта и ваши лоты в Магазине доменов.
//
//	REGRU_USER=login REGRU_PASSWORD=secret go run ./examples/mylots
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sarnas-it/regru-go"
)

func main() {
	c := regru.New(os.Getenv("REGRU_USER"), os.Getenv("REGRU_PASSWORD"), regru.WithLang("ru"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	domains, err := c.ListDomains(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Домены:")
	for _, d := range domains {
		fmt.Printf("  %-30s %s  до %s\n", d.Domain, d.State, d.ExpirationDate)
	}

	fmt.Println("Лоты в магазине:")
	err = c.EachLot(ctx, regru.LotListParams{MyLots: true, PerPage: 100}, func(l regru.Lot) error {
		fmt.Printf("  %-30s %s ₽ (%s)\n", l.Domain, l.StartPrice, l.PriceType)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
