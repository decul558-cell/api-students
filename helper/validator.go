package helper

import (
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()

	// strongpassword adalah validasi custom: password dianggap VALID
	// (fungsi mengembalikan true) hanya bila passwordStrength melaporkan
	// TIDAK ada masalah (string kosong).
	v.RegisterValidation("strongpassword", func(fl validator.FieldLevel) bool {
		return passwordStrength(fl.Field().String()) == ""
	})

	return v
}

// passwordStrength mengembalikan string kosong bila password valid,
// atau pesan alasan penolakan bila tidak.
func passwordStrength(password string) string {
	if len(password) < 8 {
		return "minimal 8 karakter"
	}
	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return "harus memuat huruf dan angka"
	}
	weak := map[string]bool{
		"password1": true, "12345678": true, "qwerty123": true,
		"admin123": true, "password123": true,
	}
	if weak[strings.ToLower(password)] {
		return "password terlalu umum"
	}
	return ""
}

// ValidateStruct menjalankan validasi deklaratif (tag `validate:"..."`)
// pada sebuah struct, lalu mengubah hasilnya menjadi *AppError 422 dengan
// rincian per-field bila ada yang gagal. Mengembalikan nil bila lolos.
func ValidateStruct(s any) *AppError {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	fields := map[string]string{}
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range validationErrs {
			fields[toSnakeCase(fe.Field())] = describeTag(fe)
		}
	} else {
		// Error selain ValidationErrors (mis. tipe struct salah) adalah
		// kesalahan programmer, bukan kesalahan input pemakai — fail
		// closed sebagai kegagalan validasi generik daripada panic.
		fields["_"] = "gagal memvalidasi input"
	}

	return Validation(fields)
}

// describeTag menerjemahkan nama tag validator menjadi pesan berbahasa
// Indonesia yang bisa dibaca pemakai API, tanpa membocorkan istilah
// internal library validator.
func describeTag(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "min":
		return "minimal " + fe.Param() + " karakter/nilai"
	case "max":
		return "maksimal " + fe.Param() + " karakter/nilai"
	case "alphanum":
		return "hanya boleh huruf dan angka"
	case "gte":
		return "harus lebih besar atau sama dengan " + fe.Param()
	case "lte":
		return "harus lebih kecil atau sama dengan " + fe.Param()
	case "email":
		return "format email tidak valid"
	case "strongpassword":
		return "password terlalu lemah (minimal 8 karakter, campuran huruf dan angka)"
	default:
		return "tidak memenuhi aturan " + fe.Tag()
	}
}

func toSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
