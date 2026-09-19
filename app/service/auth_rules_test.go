package service

import (
	"testing"

	"api-students/app/model"
)

func TestCheckPasswordStrength(t *testing.T) {
	cases := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"terlalu pendek", "abc123", true},
		{"tidak ada angka", "hanyahurufsaja", true},
		{"tidak ada huruf", "12345678", true},
		{"password umum", "password123", true},
		{"valid", "Rahasia123", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := checkPasswordStrength(tc.password)
			gotErr := msg != ""
			if gotErr != tc.wantErr {
				t.Errorf("password=%q: harap error=%v, dapat pesan=%q",
					tc.password, tc.wantErr, msg)
			}
		})
	}
}

func TestValidateRegister_Valid(t *testing.T) {
	errs := ValidateRegister(model.RegisterRequest{
		Username: "budi_santoso",
		Email:    "budi@example.com",
		Password: "Rahasia123",
	})
	if len(errs) != 0 {
		t.Errorf("data valid seharusnya tidak menghasilkan error, dapat: %v", errs)
	}
}

func TestValidateRegister_InvalidUsername(t *testing.T) {
	// Username mengandung karakter yang tidak diizinkan (spasi dan tanda seru).
	errs := ValidateRegister(model.RegisterRequest{
		Username: "budi santoso!",
		Email:    "budi@example.com",
		Password: "Rahasia123",
	})
	if _, ok := errs["username"]; !ok {
		t.Error("username dengan karakter tidak valid seharusnya menghasilkan error")
	}
}

func TestValidateRegister_InvalidEmail(t *testing.T) {
	errs := ValidateRegister(model.RegisterRequest{
		Username: "budisantoso",
		Email:    "bukan-email-valid",
		Password: "Rahasia123",
	})
	if _, ok := errs["email"]; !ok {
		t.Error("email dengan format tidak valid seharusnya menghasilkan error")
	}
}

func TestValidateLogin_TidakMengecekKekuatanPassword(t *testing.T) {
	// ValidateLogin sengaja TIDAK menolak password lemah/pendek,
	// karena user lama mungkin punya password yang dibuat sebelum
	// aturan kekuatan ini ada. Aturan kekuatan cuma berlaku saat
	// MEMBUAT password baru, bukan saat memverifikasi yang sudah ada.
	errs := ValidateLogin(model.LoginRequest{
		Username: "budisantoso",
		Password: "123", // pendek dan lemah, tapi harus tetap lolos di sini
	})
	if len(errs) != 0 {
		t.Errorf("ValidateLogin seharusnya tidak menolak password lemah, dapat: %v", errs)
	}
}
