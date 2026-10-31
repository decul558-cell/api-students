package model

import "time"

type Student struct {
	ID        int       `json:"id"`
	NIM       string    `json:"nim"`
	Name      string    `json:"name"`
	Grade     float64   `json:"grade"`
	IsActive  bool      `json:"is_active"`
	OwnerID   int       `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

// POST — semua field wajib. Validasi sekarang deklaratif lewat tag
// `validate`, dipanggil helper.ValidateStruct(req) di service.
// Tidak ada field OwnerID di sini (mencegah mass assignment).
type CreateStudentRequest struct {
	NIM   string  `json:"nim" validate:"required,min=3,max=20,alphanum"`
	Name  string  `json:"name" validate:"required,min=2,max=100"`
	Grade float64 `json:"grade" validate:"gte=0,lte=100"`
}

// PUT — ganti seluruh isi, jadi field bertipe biasa dan semuanya wajib
type ReplaceStudentRequest struct {
	NIM      string  `json:"nim" validate:"required,min=3,max=20,alphanum"`
	Name     string  `json:"name" validate:"required,min=2,max=100"`
	Grade    float64 `json:"grade" validate:"gte=0,lte=100"`
	IsActive bool    `json:"is_active"`
}

// PATCH — SEMUA field bertipe pointer, supaya validator bisa membedakan
// "tidak dikirim" (nil, dilewati lewat omitnil) dari "dikirim tapi salah"
// (bukan nil, tetap divalidasi). Ini pelajaran dari Bug #3: field yang
// tidak konsisten tipenya dengan cara ia dipakai (dereference *req.X,
// dibandingkan dengan nil) akan ditolak compiler.
type PatchStudentRequest struct {
	NIM      *string  `json:"nim,omitempty" validate:"omitnil,min=3,max=20,alphanum"`
	Name     *string  `json:"name,omitempty" validate:"omitnil,min=2,max=100"`
	Grade    *float64 `json:"grade,omitempty" validate:"omitnil,gte=0,lte=100"`
	IsActive *bool    `json:"is_active,omitempty"`
}

type WebResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ListQuery struct {
	Page     int
	Limit    int
	Search   string
	Sort     string
	Order    string
	IsActive *bool
	MinGrade *float64
	MaxGrade *float64

	// Untuk cursor-based pagination (endpoint terpisah dari List biasa).
	Cursor string
}

func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.Limit
}

// CursorMeta menggantikan pagination berbasis nomor halaman untuk
// endpoint yang mendukung cursor-based pagination (lihat D.3).
type CursorMeta struct {
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}
