package model

import "time"

// Portfolio adalah bentuk portofolio yang dikenal seluruh aplikasi. Tidak ada
// tipe driver MongoDB di sini: ID proyek disimpan sebagai string.
type Portfolio struct {
	StudentID    int           `json:"student_id"`
	Headline     string        `json:"headline"`
	Bio          string        `json:"bio"`
	Projects     []Project     `json:"projects"`
	Certificates []Certificate `json:"certificates"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// Project disimpan DI DALAM dokumen portofolio (embedding).
type Project struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Technologies []string `json:"technologies"`
	URL          string   `json:"url"`
	Year         int      `json:"year"`
}

type Certificate struct {
	Name     string `json:"name"`
	Issuer   string `json:"issuer"`
	IssuedAt string `json:"issued_at"` // YYYY-MM-DD
}

// ProjectInput dipakai saat menambah proyek dan sebagai elemen PUT.
type ProjectInput struct {
	Title        string   `json:"title" validate:"required,min=2,max=100"`
	Technologies []string `json:"technologies" validate:"max=10,dive,min=1,max=30"`
	URL          string   `json:"url" validate:"omitempty,url,max=300"`
	Year         int      `json:"year" validate:"required,min=2000,max=2100"`
}

type CertificateInput struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Issuer   string `json:"issuer" validate:"required,min=2,max=100"`
	IssuedAt string `json:"issued_at" validate:"required,datetime=2006-01-02"`
}

// PutPortfolioRequest mengganti seluruh isi portofolio.
type PutPortfolioRequest struct {
	Headline     string             `json:"headline" validate:"required,min=3,max=100"`
	Bio          string             `json:"bio" validate:"max=1000"`
	Projects     []ProjectInput     `json:"projects" validate:"max=20,dive"`
	Certificates []CertificateInput `json:"certificates" validate:"max=20,dive"`
}

type TechCount struct {
	Technology string `json:"technology"`
	Total      int    `json:"total"`
}
