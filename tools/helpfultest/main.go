// helpfultest membuktikan bahwa tombol "membantu" atomik.
// Uji 1: 30 mahasiswa berbeda menilai satu ulasan serentak.
// Uji 2: 1 mahasiswa menilai 20 kali serentak.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"api-students/app/model"
	"api-students/config"
	"api-students/database"
	"api-students/helper"
)

func fire(n int, job func(i int) string) map[string]int {
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int, out chan<- string) {
			defer wg.Done()
			<-start
			out <- job(i)
		}(i, results)
	}
	close(start)
	wg.Wait()
	close(results)

	tally := map[string]int{}
	for r := range results {
		tally[r]++
	}
	return tally
}

var client = &http.Client{Timeout: 10 * time.Second}

func call(method, url, token string) (string, []byte) {
	req, err := http.NewRequest(method, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return "ERROR request", nil
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "ERROR koneksi", nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return fmt.Sprintf("%d OK", resp.StatusCode), raw
	}
	var e struct {
		Code string `json:"code"`
	}
	if json.Unmarshal(raw, &e) == nil && e.Code != "" {
		return fmt.Sprintf("%d %s", resp.StatusCode, e.Code), raw
	}
	return fmt.Sprintf("%d", resp.StatusCode), raw
}

func helpfulCount(base, id, token string) int {
	status, raw := call("GET", base+"/api/v1/reviews/"+id, token)
	if status != "200 OK" {
		log.Fatalf("gagal membaca ulasan: %s", status)
	}
	var out struct {
		Data struct {
			HelpfulCount int `json:"helpful_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		log.Fatalf("membaca response: %v", err)
	}
	return out.Data.HelpfulCount
}

func printTally(t map[string]int) {
	keys := make([]string, 0, len(t))
	for k := range t {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %-28s %d\n", k, t[k])
	}
}

func main() {
	reviewID := flag.String("review", "", "id ulasan yang dinilai")
	base := flag.String("base", "http://localhost:3000", "alamat server")
	flag.Parse()
	if *reviewID == "" {
		log.Fatal("isi -review dengan id ulasan")
	}

	config.LoadEnv()
	ctx := context.Background()

	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	secret := config.GetEnv("JWT_SECRET", "")
	if secret == "" {
		log.Fatal("JWT_SECRET kosong: jalankan dari folder proyek yang berisi .env")
	}
	jwtManager := helper.NewJWTManager(
		secret,
		config.GetEnv("JWT_ISSUER", "api-students"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15))*time.Minute,
	)

	rows, err := pool.Query(ctx,
		`SELECT id, username, email, role, is_active
		 FROM users WHERE username ~ '^mhs[0-9]{3}$' ORDER BY username`)
	if err != nil {
		log.Fatalf("membaca akun uji: %v", err)
	}
	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.IsActive); err != nil {
			log.Fatalf("membaca akun: %v", err)
		}
		users = append(users, u)
	}
	rows.Close()
	if len(users) < 31 {
		log.Fatalf("butuh minimal 31 akun mhs, ditemukan %d", len(users))
	}

	tokens := make([]string, len(users))
	for i, u := range users {
		t, err := jwtManager.GenerateAccess(u)
		if err != nil {
			log.Fatalf("membuat token %s: %v", u.Username, err)
		}
		tokens[i] = t
	}

	url := *base + "/api/v1/reviews/" + *reviewID + "/helpful"
	before := helpfulCount(*base, *reviewID, tokens[0])
	fmt.Printf("helpful_count sebelum uji: %d (harus 0; reset dulu bila bukan)\n\n", before)

	fmt.Println("Uji 1: 30 mahasiswa berbeda menilai serentak (mhs001..mhs030)")
	t1 := fire(30, func(i int) string {
		s, _ := call("POST", url, tokens[i])
		return s
	})
	printTally(t1)
	c1 := helpfulCount(*base, *reviewID, tokens[0])
	fmt.Printf("  helpful_count sesudah: %d (harus %d)\n\n", c1, before+30)

	fmt.Println("Uji 2: 1 mahasiswa (mhs031) menilai 20 kali serentak")
	t2 := fire(20, func(int) string {
		s, _ := call("POST", url, tokens[30])
		return s
	})
	printTally(t2)
	c2 := helpfulCount(*base, *reviewID, tokens[0])
	fmt.Printf("  helpful_count sesudah: %d (harus %d)\n", c2, c1+1)
}
