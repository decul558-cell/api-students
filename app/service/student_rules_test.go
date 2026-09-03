package service

import (
	"testing"

	"api-students/app/model"
)

// Perhatikan: pengujian ini tidak menyalakan server, tidak menyentuh
// database, dan tidak membuat fiber.Ctx.

func TestCountTotalPages(t *testing.T) {
	cases := []struct{ total, limit, want int }{
		{0, 10, 0},
		{1, 10, 1},
		{10, 10, 1},
		{11, 10, 2},
		{137, 20, 7},
	}
	for _, tc := range cases {
		if got := CountTotalPages(tc.total, tc.limit); got != tc.want {
			t.Errorf("total=%d limit=%d: harap %d, dapat %d",
				tc.total, tc.limit, tc.want, got)
		}
	}
}

func TestValidateCreate(t *testing.T) {
	// Kasus valid: tidak boleh ada error sama sekali.
	errs := ValidateCreate(model.CreateStudentRequest{
		NIM: "12345", Name: "Budi Santoso", Grade: 85,
	})
	if len(errs) != 0 {
		t.Errorf("data valid seharusnya tidak menghasilkan error, dapat: %v", errs)
	}

	// Kasus tidak valid: NIM kosong dan grade di luar rentang.
	errs = ValidateCreate(model.CreateStudentRequest{
		NIM: "", Name: "Budi", Grade: 150,
	})
	if _, ok := errs["nim"]; !ok {
		t.Error("nim kosong seharusnya menghasilkan error")
	}
	if _, ok := errs["grade"]; !ok {
		t.Error("grade di atas 100 seharusnya menghasilkan error")
	}
}

func TestApplyPatch(t *testing.T) {
	initial := model.Student{ID: 1, NIM: "12345", Name: "Sari", Grade: 80, IsActive: true}
	inactive := false

	result, errs := ApplyPatch(initial, model.PatchStudentRequest{IsActive: &inactive})
	if len(errs) != 0 {
		t.Fatalf("tidak seharusnya ada error: %v", errs)
	}
	if result.IsActive {
		t.Error("is_active seharusnya berubah menjadi false")
	}
	if result.Name != "Sari" {
		t.Error("field yang tidak dikirim seharusnya tidak berubah")
	}
}
