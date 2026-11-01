// Команда txtcheck — живой smoke-тест AddTXTRecord: добавляет TXT-запись в зону
// домена и перечитывает зону, проверяя, что запись появилась с тем же текстом.
//
//	REGRU_USER=login REGRU_PASSWORD=secret go run ./examples/txtcheck example.ru [subdomain] [text]
//
// По умолчанию subdomain "_regru-go-test", текст "regru-go smoke <время>".
// Запись НЕ удаляется (в клиенте пока нет RemoveTXTRecord) — снимите её
// в личном кабинете Рег.ру после прогона.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"code.sarnas.ru/regru-go"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: txtcheck <domain> [subdomain] [text]")
	}
	domain := os.Args[1]
	sub := "_regru-go-test"
	if len(os.Args) > 2 {
		sub = os.Args[2]
	}
	text := "regru-go smoke " + time.Now().UTC().Format(time.RFC3339)
	if len(os.Args) > 3 {
		text = os.Args[3]
	}

	c := regru.New(os.Getenv("REGRU_USER"), os.Getenv("REGRU_PASSWORD"), regru.WithLang("ru"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if _, err := c.Nop(ctx); err != nil {
		log.Fatalf("nop: %v (проверьте логин, пароль и список IP в настройках API)", err)
	}

	zone, err := zoneOf(ctx, c, domain)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Записей в зоне %s до добавления: %d\n", domain, len(zone.Records))

	if err := c.AddTXTRecord(ctx, domain, sub, text); err != nil {
		log.Fatalf("AddTXTRecord: %v", err)
	}
	fmt.Printf("Добавлена TXT %s.%s = %q\n", sub, domain, text)

	zone, err = zoneOf(ctx, c, domain)
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range zone.Records {
		if r.Type == "TXT" && r.Sub == sub {
			fmt.Printf("В зоне: TXT %s = %q\n", r.Sub, r.Content)
			if r.Content != text {
				log.Fatalf("FAIL: текст не совпал: хотели %q", text)
			}
			fmt.Println("OK: запись появилась с тем же текстом")
			fmt.Println("Не забудьте удалить запись в личном кабинете Рег.ру.")
			return
		}
	}
	log.Fatalf("FAIL: запись TXT %s не найдена в зоне после добавления", sub)
}

func zoneOf(ctx context.Context, c *regru.Client, domain string) (regru.Zone, error) {
	zs, err := c.ZoneRecords(ctx, domain)
	if err != nil {
		return regru.Zone{}, err
	}
	if len(zs) == 0 {
		return regru.Zone{}, fmt.Errorf("zone/get_resource_records: пустой ответ по %s", domain)
	}
	if zs[0].Err != nil {
		return regru.Zone{}, fmt.Errorf("зона %s: %w", domain, zs[0].Err)
	}
	return zs[0], nil
}
