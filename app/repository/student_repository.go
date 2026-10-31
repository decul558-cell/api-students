package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/model"
	"api-students/helper"
)

// Sentinel error: error milik lapisan repository, bukan error milik pgx.
var (
	ErrNotFound  = errors.New("data tidak ditemukan")
	ErrDuplicate = errors.New("data sudah ada")
)

// StudentRepository adalah KONTRAK penyimpanan data mahasiswa.
type StudentRepository interface {
	FindAll(ctx context.Context, q model.ListQuery) ([]model.Student, int, error)
	FindByID(ctx context.Context, id int) (model.Student, error)
	Create(ctx context.Context, s model.Student) (model.Student, error)
	Update(ctx context.Context, s model.Student) (model.Student, error)
	Delete(ctx context.Context, id int) error
	FindWithCursor(ctx context.Context, cursor string, limit int) ([]model.Student, string, bool, error)
}

var kolomUrut = map[string]string{
	"id":         "id",
	"nim":        "nim",
	"name":       "name",
	"grade":      "grade",
	"created_at": "created_at",
}

type studentPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewStudentRepository(pool *pgxpool.Pool) StudentRepository {
	return &studentPostgresRepository{pool: pool}
}

func buildFilter(q model.ListQuery) (string, []any) {
	where := " WHERE 1 = 1"
	args := []any{}

	if q.Search != "" {
		where += fmt.Sprintf(" AND name ILIKE $%d", len(args)+1)
		args = append(args, "%"+q.Search+"%")
	}
	if q.IsActive != nil {
		where += fmt.Sprintf(" AND is_active = $%d", len(args)+1)
		args = append(args, *q.IsActive)
	}
	if q.MinGrade != nil {
		where += fmt.Sprintf(" AND grade >= $%d", len(args)+1)
		args = append(args, *q.MinGrade)
	}
	if q.MaxGrade != nil {
		where += fmt.Sprintf(" AND grade <= $%d", len(args)+1)
		args = append(args, *q.MaxGrade)
	}

	return where, args
}

func (r *studentPostgresRepository) FindAll(
	ctx context.Context, q model.ListQuery,
) ([]model.Student, int, error) {
	where, args := buildFilter(q)

	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM students"+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("menghitung mahasiswa: %w", err)
	}

	arah := "ASC"
	if q.Order == "desc" {
		arah = "DESC"
	}

	sqlText := fmt.Sprintf(
		`SELECT id, nim, name, grade, is_active, owner_id, created_at
		 FROM students%s
		 ORDER BY %s %s
		 LIMIT $%d OFFSET $%d`,
		where, kolomUrut[q.Sort], arah, len(args)+1, len(args)+2,
	)
	args = append(args, q.Limit, q.Offset())

	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil daftar mahasiswa: %w", err)
	}
	defer rows.Close()

	hasil := []model.Student{}
	for rows.Next() {
		var s model.Student
		if err := rows.Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive,
			&s.OwnerID, &s.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("membaca baris mahasiswa: %w", err)
		}
		hasil = append(hasil, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("membaca hasil query: %w", err)
	}

	return hasil, total, nil
}

func (r *studentPostgresRepository) FindByID(
	ctx context.Context, id int,
) (model.Student, error) {
	var s model.Student
	err := r.pool.QueryRow(ctx,
		`SELECT id, nim, name, grade, is_active, owner_id, created_at
		 FROM students WHERE id = $1`, id,
	).Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.OwnerID, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, fmt.Errorf("mengambil mahasiswa: %w", err)
	}
	return s, nil
}

// Create menyimpan owner_id yang sudah ditentukan SERVICE (dari identitas
// pemanggil), bukan dari client. Repository di sini cuma menyimpan apa
// yang dikirim; keputusan "owner_id-nya siapa" sudah selesai di layer atas.
func (r *studentPostgresRepository) Create(
	ctx context.Context, s model.Student,
) (model.Student, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO students (nim, name, grade, is_active, owner_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		s.NIM, s.Name, s.Grade, s.IsActive, s.OwnerID,
	).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("menyimpan mahasiswa: %w", err)
	}
	return s, nil
}

// Update sengaja TIDAK menyertakan owner_id pada SET. Kepemilikan data
// tidak boleh berpindah lewat endpoint update biasa.
func (r *studentPostgresRepository) Update(
	ctx context.Context, s model.Student,
) (model.Student, error) {
	err := r.pool.QueryRow(ctx,
		`UPDATE students SET nim = $1, name = $2, grade = $3, is_active = $4
		 WHERE id = $5
		 RETURNING id, nim, name, grade, is_active, owner_id, created_at`,
		s.NIM, s.Name, s.Grade, s.IsActive, s.ID,
	).Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.OwnerID, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("memperbarui mahasiswa: %w", err)
	}
	return s, nil
}

func (r *studentPostgresRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM students WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("menghapus mahasiswa: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// FindWithCursor mengambil satu halaman data terurut dari yang TERBARU,
// memakai keyset pagination (bukan OFFSET). Mengembalikan data, cursor
// untuk halaman berikutnya, dan apakah masih ada halaman setelahnya.
//
// PERHATIKAN: ORDER BY di sini (created_at DESC, id DESC) HARUS SAMA
// PERSIS dengan urutan kolom pada index students_created_at_id_desc_idx.
// Bila arahnya tidak cocok (Bug #8), PostgreSQL tidak bisa memakai index
// secara optimal, dan urutan "dari yang terbaru" bisa jadi salah.
func (r *studentPostgresRepository) FindWithCursor(
	ctx context.Context, cursor string, limit int,
) ([]model.Student, string, bool, error) {
	cursorCreatedAt, cursorID, err := helper.DecodeCursor(cursor)
	if err != nil {
		return nil, "", false, err
	}

	var rows pgx.Rows
	if cursor == "" {
		// Halaman pertama: tidak ada batas bawah.
		rows, err = r.pool.Query(ctx,
			`SELECT id, nim, name, grade, is_active, owner_id, created_at
			 FROM students
			 ORDER BY created_at DESC, id DESC
			 LIMIT $1`,
			limit+1, // ambil satu ekstra untuk tahu apakah masih ada halaman berikutnya
		)
	} else {
		// Keyset: ambil baris yang "lebih lama" dari posisi cursor,
		// dibandingkan sebagai pasangan (created_at, id) sekaligus --
		// bukan dua kondisi AND terpisah, supaya baris dengan created_at
		// sama tapi id lebih kecil tetap tertangkap dengan benar.
		rows, err = r.pool.Query(ctx,
			`SELECT id, nim, name, grade, is_active, owner_id, created_at
			 FROM students
			 WHERE (created_at, id) < ($1, $2)
			 ORDER BY created_at DESC, id DESC
			 LIMIT $3`,
			cursorCreatedAt, cursorID, limit+1,
		)
	}
	if err != nil {
		return nil, "", false, fmt.Errorf("mengambil halaman mahasiswa: %w", err)
	}
	defer rows.Close()

	hasil := []model.Student{}
	for rows.Next() {
		var s model.Student
		if err := rows.Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive,
			&s.OwnerID, &s.CreatedAt); err != nil {
			return nil, "", false, fmt.Errorf("membaca baris mahasiswa: %w", err)
		}
		hasil = append(hasil, s)
	}
	if err := rows.Err(); err != nil {
		return nil, "", false, fmt.Errorf("membaca hasil query: %w", err)
	}

	hasMore := len(hasil) > limit
	if hasMore {
		hasil = hasil[:limit] // buang baris ekstra yang cuma dipakai untuk deteksi hasMore
	}

	nextCursor := ""
	if hasMore && len(hasil) > 0 {
		last := hasil[len(hasil)-1]
		nextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}

	return hasil, nextCursor, hasMore, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
