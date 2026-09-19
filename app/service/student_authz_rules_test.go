package service

import (
	"testing"

	"api-students/app/model"
	"api-students/helper"
)

// testPermissions membangun PermissionSet kecil untuk keperluan test,
// meniru isi tabel role_permissions tanpa perlu database sungguhan.
func testPermissions() *helper.PermissionSet {
	return helper.NewPermissionSet(map[string][]string{
		"admin": {"student:read:any", "student:update:any"},
		"staff": {"student:read:any"},
		"user":  {},
	})
}

func TestCanAccessStudent_Owner(t *testing.T) {
	perms := testPermissions()

	// Pemilik data selalu boleh, apa pun role-nya dan meski tidak
	// punya permission apa pun.
	current := model.AuthUser{UserID: 1, Role: "user"}
	if !CanAccessStudent(current, 1, perms, "student:read:any") {
		t.Error("pemilik data seharusnya selalu boleh mengakses datanya sendiri")
	}
}

func TestCanAccessStudent_NonOwnerWithPermission(t *testing.T) {
	perms := testPermissions()

	// Bukan pemilik, tetapi role-nya punya permission yang sesuai.
	current := model.AuthUser{UserID: 2, Role: "staff"}
	if !CanAccessStudent(current, 99, perms, "student:read:any") {
		t.Error("role dengan permission yang sesuai seharusnya boleh mengakses data siapa pun")
	}
}

func TestCanAccessStudent_NonOwnerWithoutPermission(t *testing.T) {
	perms := testPermissions()

	// Bukan pemilik, dan role-nya tidak punya permission apa pun.
	// Ini kasus paling penting: user biasa mencoba akses data orang lain.
	current := model.AuthUser{UserID: 2, Role: "user"}
	if CanAccessStudent(current, 99, perms, "student:read:any") {
		t.Error("user tanpa permission seharusnya TIDAK boleh mengakses data orang lain")
	}
}

func TestCanAccessStudent_WrongPermissionType(t *testing.T) {
	perms := testPermissions()

	// staff punya student:read:any, TAPI TIDAK punya student:update:any.
	// Ini membuktikan pemisahan permission per-aksi benar-benar diperiksa,
	// bukan cuma "role ini boleh apa saja".
	current := model.AuthUser{UserID: 2, Role: "staff"}
	if CanAccessStudent(current, 99, perms, "student:update:any") {
		t.Error("staff tidak punya student:update:any, seharusnya ditolak untuk aksi update")
	}
}

func TestCanAccessStudent_UnknownRoleFailsClosed(t *testing.T) {
	perms := testPermissions()

	// Role yang tidak dikenal PermissionSet sama sekali (misalnya salah
	// ketik atau data korup) harus ditolak, bukan diloloskan.
	current := model.AuthUser{UserID: 2, Role: "peran-tidak-dikenal"}
	if CanAccessStudent(current, 99, perms, "student:read:any") {
		t.Error("role tidak dikenal seharusnya fail closed (ditolak), bukan diloloskan")
	}
}
