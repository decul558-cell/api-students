package service

import (
	"api-students/app/model"
	"api-students/helper"
)

// CanAccessStudent memutuskan apakah seseorang boleh menyentuh data
// mahasiswa milik orang lain.
//
// Dua jalur yang diizinkan:
//  1. Kepemilikan (ownership) — dialah yang mendaftarkan data ini (owner_id
//     sama dengan user id-nya).
//  2. Permission — role-nya memang berhak atas data siapa pun (mis. student:read:any).
//
// Urutannya disengaja: pemeriksaan kepemilikan didahulukan karena paling
// murah (tidak perlu lihat PermissionSet) dan paling sering benar (mayoritas
// akses adalah orang mengurus datanya sendiri). Bila keduanya gagal, false.
func CanAccessStudent(
	current model.AuthUser,
	ownerID int,
	perms *helper.PermissionSet,
	anyPermission string,
) bool {
	if current.UserID == ownerID {
		return true
	}
	return perms.Can(current.Role, anyPermission)
}
