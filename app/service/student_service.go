package service

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

type StudentService struct {
	repo  repository.StudentRepository
	perms *helper.PermissionSet
}

func NewStudentService(repo repository.StudentRepository, perms *helper.PermissionSet) *StudentService {
	return &StudentService{repo: repo, perms: perms}
}

// terjemahkanError mengembalikan *helper.AppError, BUKAN menulis response
// secara langsung. Cabang default fail closed: error yang tidak dikenal
// selalu jadi 500 INTERNAL_ERROR, tidak pernah diam-diam dianggap sukses.
func terjemahkanError(err error, pesanUmum string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("mahasiswa tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict("NIM sudah terdaftar")
	default:
		return helper.Internal(err)
	}
}

func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q := helper.ParseListQuery(c)
	students, total, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	totalPages := 0
	if q.Limit > 0 {
		totalPages = (total + q.Limit - 1) / q.Limit
	}

	return helper.SuccessList(c, "daftar mahasiswa berhasil diambil", students, &model.Meta{
		Page: q.Page, Limit: q.Limit, Total: total, TotalPages: totalPages,
	})
}

// ListCursor adalah versi cursor-based dari List. Endpoint terpisah
// (bukan menggantikan List) supaya keduanya bisa dibandingkan langsung.
func (s *StudentService) ListCursor(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	cursor := c.Query("cursor", "")
	limit := c.QueryInt("limit", 10)
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	students, nextCursor, hasMore, err := s.repo.FindWithCursor(ctx, cursor, limit)
	if err != nil {
		return helper.BadRequest("cursor tidak valid")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "daftar mahasiswa berhasil diambil",
		"data":    students,
		"meta": model.CursorMeta{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	})
}

func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(err, "gagal mengambil data mahasiswa")
	}

	if !CanAccessStudent(current, student.OwnerID, s.perms, "student:read:any") {
		return helper.Forbidden("tidak berhak mengakses data mahasiswa lain")
	}

	return helper.Success(c, fiber.StatusOK, "mahasiswa ditemukan", student)
}

func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	if appErr := helper.ValidateStruct(req); appErr != nil {
		return appErr
	}

	baru, err := s.repo.Create(ctx, model.Student{
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: true,
		OwnerID:  current.UserID, // selalu dari identitas login, bukan body
	})
	if err != nil {
		return terjemahkanError(err, "gagal menyimpan mahasiswa")
	}

	return helper.Created(c, "mahasiswa berhasil ditambahkan", baru,
		"/api/v1/students/"+strconv.Itoa(baru.ID))
}

func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(err, "gagal mengambil data mahasiswa")
	}

	if !CanAccessStudent(current, existing.OwnerID, s.perms, "student:update:any") {
		return helper.Forbidden("tidak berhak mengubah data mahasiswa lain")
	}

	var req model.ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	if appErr := helper.ValidateStruct(req); appErr != nil {
		return appErr
	}

	hasil, err := s.repo.Update(ctx, model.Student{
		ID: id, NIM: req.NIM, Name: req.Name, Grade: req.Grade, IsActive: req.IsActive,
	})
	if err != nil {
		return terjemahkanError(err, "gagal memperbarui mahasiswa")
	}
	return helper.Success(c, fiber.StatusOK, "mahasiswa berhasil diganti seluruhnya", hasil)
}

func (s *StudentService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	saatIni, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(err, "gagal mengambil data mahasiswa")
	}

	if !CanAccessStudent(current, saatIni.OwnerID, s.perms, "student:update:any") {
		return helper.Forbidden("tidak berhak mengubah data mahasiswa lain")
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if req.NIM == nil && req.Name == nil && req.Grade == nil && req.IsActive == nil {
		return helper.BadRequest("tidak ada field yang diubah")
	}

	if appErr := helper.ValidateStruct(req); appErr != nil {
		return appErr
	}

	if req.NIM != nil {
		saatIni.NIM = strings.TrimSpace(*req.NIM)
	}
	if req.Name != nil {
		saatIni.Name = strings.TrimSpace(*req.Name)
	}
	if req.Grade != nil {
		saatIni.Grade = *req.Grade
	}
	if req.IsActive != nil {
		saatIni.IsActive = *req.IsActive
	}

	hasil, err := s.repo.Update(ctx, saatIni)
	if err != nil {
		return terjemahkanError(err, "gagal memperbarui mahasiswa")
	}
	return helper.Success(c, fiber.StatusOK, "mahasiswa berhasil diperbarui sebagian", hasil)
}

func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return terjemahkanError(err, "gagal menghapus mahasiswa")
	}
	return helper.NoContent(c)
}
