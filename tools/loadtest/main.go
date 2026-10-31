// Alat uji beban untuk Modul 9. BUKAN bagian aplikasi: tidak ikut di-deploy.
//
//	go run ./tools/loadtest -mode crowd    // 60 mahasiswa berebut IF399 (kuota 5)
//	go run ./tools/loadtest -mode credits  // 10 mahasiswa x 12 kelas (batas 24 SKS)
//	go run ./tools/loadtest -mode swap     // 20 pasang pindah kelas berlawanan arah
//
// Server harus sedang berjalan. Alat ini membuat token sendiri karena ia
// memegang JWT_SECRET; itulah sebabnya JWT_SECRET tidak boleh bocor.
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

	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/model"
	"api-students/config"
	"api-students/database"
	"api-students/helper"
)

type api struct {
	base   string
	client *http.Client
}

func main() {
	mode := flag.String("mode", "crowd", "crowd | credits | swap")
	base := flag.String("base", "http://localhost:3000", "alamat server yang diuji")
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

	users, err := loadTestUsers(ctx, pool)
	if err != nil {
		log.Fatalf("membaca akun uji: %v", err)
	}
	if len(users) < 60 {
		log.Fatalf("butuh 60 akun mhs001..mhs060, ditemukan %d (jalankan migration 007)", len(users))
	}
	users = users[:60]

	tokens := make([]string, len(users))
	for i, u := range users {
		t, err := jwtManager.GenerateAccess(u)
		if err != nil {
			log.Fatalf("membuat token %s: %v", u.Username, err)
		}
		tokens[i] = t
	}

	a := &api{
		base: *base,
		client: &http.Client{
			Timeout:   20 * time.Second,
			Transport: &http.Transport{MaxIdleConnsPerHost: 200},
		},
	}

	switch *mode {
	case "crowd":
		runCrowd(ctx, pool, a, tokens)
	case "credits":
		runCredits(ctx, pool, a, users, tokens)
	case "swap":
		runSwap(ctx, pool, a, users, tokens)
	default:
		log.Fatalf("mode tidak dikenal: %q", *mode)
	}
}

// fire menjalankan n pekerjaan secara serentak dan menghitung hasilnya.
//
// Setiap goroutine menunggu di depan "gerbang" start yang sama. Gerbang
// baru dibuka setelah SELURUH goroutine siap, sehingga request benar-benar
// berangkat hampir bersamaan — bukan bergiliran sesuai urutan dibuatnya.
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

	close(start) // buka gerbang
	wg.Wait()    // tunggu SEMUA goroutine selesai
	close(results)

	tally := map[string]int{}
	for r := range results {
		tally[r]++
	}
	return tally
}

// call mengirim satu request dan meringkas hasilnya menjadi satu kata kunci,
// misalnya "201 OK" atau "409 COURSE_FULL".
func (a *api) call(method, path, token string, body any) string {
	payload := []byte("{}")
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return "ERROR marshal"
		}
		payload = b
	}
	req, err := http.NewRequest(method, a.base+path, bytes.NewReader(payload))
	if err != nil {
		return "ERROR request"
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "ERROR koneksi"
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return fmt.Sprintf("%d OK", resp.StatusCode)
	}
	var e struct {
		Code string `json:"code"`
	}
	if json.Unmarshal(raw, &e) == nil && e.Code != "" {
		return fmt.Sprintf("%d %s", resp.StatusCode, e.Code)
	}
	return strconv.Itoa(resp.StatusCode)
}

func enrollPath(courseID int) string {
	return "/api/v1/courses/" + strconv.Itoa(courseID) + "/enroll"
}

// ---------------------------------------------------------------- mode crowd

func runCrowd(ctx context.Context, pool *pgxpool.Pool, a *api, tokens []string) {
	ids, err := courseIDs(ctx, pool, "IF399")
	if err != nil {
		log.Fatal(err)
	}
	path := enrollPath(ids["IF399"])

	fmt.Printf("%d mahasiswa berebut kelas IF399 secara serentak...\n", len(tokens))
	begin := time.Now()
	tally := fire(len(tokens), func(i int) string {
		return a.call(http.MethodPost, path, tokens[i], nil)
	})
	elapsed := time.Since(begin)

	fmt.Println("hasil request:")
	printTally(tally)
	fmt.Printf("waktu total : %s\n", elapsed.Round(time.Millisecond))
	line, _ := courseLine(ctx, pool, "IF399")
	fmt.Println(line)
}

// -------------------------------------------------------------- mode credits

func runCredits(ctx context.Context, pool *pgxpool.Pool, a *api, users []model.User, tokens []string) {
	codes := []string{"IF101", "IF102", "IF301", "IF302", "IF303", "IF304",
		"IF305", "IF306", "IF307", "IF308", "IF309", "IF310"}
	ids, err := courseIDs(ctx, pool, codes...)
	if err != nil {
		log.Fatal(err)
	}

	// Sepuluh penyerang, bukan satu: race condition bersifat untung-untungan.
	// Cukup satu yang lolos melewati batas untuk membuktikan kodenya tidak aman.
	const attackers = 10
	fmt.Printf("%d mahasiswa, masing-masing mendaftar ke %d kelas 3 SKS secara serentak (batas 24 SKS)...\n",
		attackers, len(codes))

	begin := time.Now()
	tally := fire(attackers*len(codes), func(i int) string {
		u, c := i/len(codes), i%len(codes)
		return a.call(http.MethodPost, enrollPath(ids[codes[c]]), tokens[u], nil)
	})
	elapsed := time.Since(begin)

	fmt.Println("hasil request:")
	printTally(tally)
	fmt.Printf("waktu total : %s\n", elapsed.Round(time.Millisecond))

	ok := true
	for u := 0; u < attackers; u++ {
		if sum := creditsOf(ctx, pool, users[u].ID); sum > 24 {
			ok = false
			fmt.Printf("  %s memiliki %d SKS -> MELEBIHI BATAS\n", users[u].Username, sum)
		}
	}
	for _, code := range codes {
		if line, good := courseLine(ctx, pool, code); !good {
			ok = false
			fmt.Println(line)
		}
	}
	if ok {
		fmt.Printf("seluruh %d mahasiswa <= 24 SKS, enrolled_count sesuai baris enrollments -> KONSISTEN\n", attackers)
	} else {
		fmt.Println("-> TIDAK KONSISTEN")
	}
}

// ---------------------------------------------------------------- mode swap

func runSwap(ctx context.Context, pool *pgxpool.Pool, a *api, users []model.User, tokens []string) {
	ids, err := courseIDs(ctx, pool, "IF101", "IF102")
	if err != nil {
		log.Fatal(err)
	}
	a101, a102 := ids["IF101"], ids["IF102"]

	// Persiapan: mhs001..020 di IF101, mhs021..040 di IF102. Mode ini
	// MENGHAPUS seluruh pendaftaran akun mhs lebih dulu.
	if err := setupSwap(ctx, pool, users, a101, a102); err != nil {
		log.Fatalf("persiapan swap: %v", err)
	}

	const pairs = 20
	fmt.Printf("%d pasang mahasiswa pindah kelas berlawanan arah secara serentak...\n", pairs)

	before := deadlocks(ctx, pool)
	begin := time.Now()
	tally := fire(pairs*2, func(i int) string {
		from, to := a101, a102
		if i >= pairs {
			from, to = a102, a101
		}
		body := map[string]int{"from_course_id": from, "to_course_id": to}
		return a.call(http.MethodPost, "/api/v1/enrollments/swap", tokens[i], body)
	})
	elapsed := time.Since(begin)

	// Statistik PostgreSQL diperbarui dengan jeda singkat.
	time.Sleep(2 * time.Second)
	dead := deadlocks(ctx, pool) - before

	fmt.Println("hasil request:")
	printTally(tally)
	fmt.Printf("waktu total : %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("deadlock terdeteksi PostgreSQL selama uji: %d\n", dead)
	for _, code := range []string{"IF101", "IF102"} {
		line, _ := courseLine(ctx, pool, code)
		fmt.Println(line)
	}
}

func setupSwap(ctx context.Context, pool *pgxpool.Pool, users []model.User, c101, c102 int) error {
	if _, err := pool.Exec(ctx,
		`DELETE FROM enrollments
		 WHERE user_id IN (SELECT id FROM users WHERE username ~ '^mhs[0-9]{3}$')`); err != nil {
		return err
	}
	for i := 0; i < 40; i++ {
		course := c101
		if i >= 20 {
			course = c102
		}
		if _, err := pool.Exec(ctx,
			"INSERT INTO enrollments (course_id, user_id) VALUES ($1, $2)", course, users[i].ID); err != nil {
			return err
		}
	}
	_, err := pool.Exec(ctx,
		`UPDATE courses c SET enrolled_count =
		   (SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id)`)
	return err
}

// ---------------------------------------------------------------- bantuan DB

func loadTestUsers(ctx context.Context, pool *pgxpool.Pool) ([]model.User, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, username, email, role, is_active
		 FROM users WHERE username ~ '^mhs[0-9]{3}$' ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.IsActive); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func courseIDs(ctx context.Context, pool *pgxpool.Pool, codes ...string) (map[string]int, error) {
	rows, err := pool.Query(ctx, "SELECT code, id FROM courses WHERE code = ANY($1)", codes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := map[string]int{}
	for rows.Next() {
		var code string
		var id int
		if err := rows.Scan(&code, &id); err != nil {
			return nil, err
		}
		ids[code] = id
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, c := range codes {
		if _, ok := ids[c]; !ok {
			return nil, fmt.Errorf("kelas %s tidak ada", c)
		}
	}
	return ids, nil
}

// courseLine membandingkan enrolled_count dengan jumlah baris enrollments
// yang sebenarnya. Baris enrollments adalah sumber kebenaran.
func courseLine(ctx context.Context, pool *pgxpool.Pool, code string) (string, bool) {
	var quota, enrolled, rows int
	err := pool.QueryRow(ctx,
		`SELECT c.quota, c.enrolled_count,
		        (SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id)::int
		 FROM courses c WHERE c.code = $1`, code).Scan(&quota, &enrolled, &rows)
	if err != nil {
		log.Fatalf("membaca %s: %v", code, err)
	}
	good := enrolled == rows && enrolled <= quota
	verdict := "KONSISTEN"
	if !good {
		verdict = "TIDAK KONSISTEN"
	}
	return fmt.Sprintf("%s kuota=%d enrolled_count=%d baris enrollments=%d -> %s",
		code, quota, enrolled, rows, verdict), good
}

func creditsOf(ctx context.Context, pool *pgxpool.Pool, userID int) int {
	var n int
	err := pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(c.credits), 0)::int
		 FROM enrollments e JOIN courses c ON c.id = e.course_id
		 WHERE e.user_id = $1`, userID).Scan(&n)
	if err != nil {
		log.Fatalf("menghitung SKS: %v", err)
	}
	return n
}

func deadlocks(ctx context.Context, pool *pgxpool.Pool) int {
	var n int64
	err := pool.QueryRow(ctx,
		"SELECT deadlocks FROM pg_stat_database WHERE datname = current_database()").Scan(&n)
	if err != nil {
		log.Fatalf("membaca statistik deadlock: %v", err)
	}
	return int(n)
}

func printTally(t map[string]int) {
	keys := make([]string, 0, len(t))
	for k := range t {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %-34s %d\n", k, t[k])
	}
}
