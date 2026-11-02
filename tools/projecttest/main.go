// projecttest membuktikan batas 20 proyek bertahan saat diserang serentak.
// Syarat awal: portofolio mahasiswa sudah berisi 18 proyek.
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
	"strconv"
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

func post(url, token string, body any) (string, []byte) {
	payload, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
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
	return strconv.Itoa(resp.StatusCode), raw
}

func projectCount(base string, studentID int, token string) int {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/students/%d/portfolio", base, studentID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("membaca portofolio: %v", err)
	}
	defer resp.Body.Close()
	var out struct {
		Data struct {
			Projects []json.RawMessage `json:"projects"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		log.Fatalf("membaca response: %v", err)
	}
	return len(out.Data.Projects)
}

func main() {
	studentID := flag.Int("student", 3, "id mahasiswa (harus sudah berisi 18 proyek)")
	ownerID := flag.Int("owner", 1, "user id pemilik mahasiswa itu (sari = 1)")
	n := flag.Int("n", 10, "jumlah permintaan serentak")
	base := flag.String("base", "http://localhost:3000", "alamat server")
	flag.Parse()

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

	var u model.User
	err = pool.QueryRow(ctx,
		`SELECT id, username, email, role, is_active FROM users WHERE id = $1`, *ownerID,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.IsActive)
	if err != nil {
		log.Fatalf("membaca pemilik: %v", err)
	}
	token, err := jwtManager.GenerateAccess(u)
	if err != nil {
		log.Fatalf("membuat token: %v", err)
	}

	before := projectCount(*base, *studentID, token)
	fmt.Printf("jumlah proyek sebelum uji: %d (harus 18)\n\n", before)

	url := fmt.Sprintf("%s/api/v1/students/%d/portfolio/projects", *base, *studentID)
	fmt.Printf("%d permintaan serentak menambah proyek...\n", *n)
	tally := fire(*n, func(i int) string {
		s, _ := post(url, token, map[string]any{
			"title":        "Proyek serentak " + strconv.Itoa(i+1),
			"technologies": []string{"go"},
			"year":         2026,
		})
		return s
	})

	keys := make([]string, 0, len(tally))
	for k := range tally {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Println("hasil request:")
	for _, k := range keys {
		fmt.Printf("  %-26s %d\n", k, tally[k])
	}

	after := projectCount(*base, *studentID, token)
	fmt.Printf("\njumlah proyek sesudah uji: %d (batas 20)\n", after)
	if after == 20 {
		fmt.Println("-> BATAS TERJAGA")
	} else {
		fmt.Println("-> BATAS TIDAK TERJAGA")
	}
}
