package service

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

// PortfolioService memakai DUA database: data mahasiswa dibaca dari
// PostgreSQL (termasuk owner_id untuk aturan hak akses), portofolionya
// disimpan di MongoDB. Tidak ada FOREIGN KEY di antara keduanya, jadi
// service yang memeriksa bahwa mahasiswanya ada.
type PortfolioService struct {
	portfolios repository.PortfolioRepository
	students   repository.StudentRepository
	perms      *helper.PermissionSet
}

func NewPortfolioService(
	portfolios repository.PortfolioRepository,
	students repository.StudentRepository,
	perms *helper.PermissionSet,
) *PortfolioService {
	return &PortfolioService{portfolios: portfolios, students: students, perms: perms}
}

// authorize memastikan mahasiswanya ada di PostgreSQL, lalu memeriksa hak
// akses dengan aturan yang sama seperti endpoint students.
func (s *PortfolioService) authorize(c *fiber.Ctx, anyPermission string) (int, error) {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return 0, helper.Unauthorized("belum terautentikasi")
	}
	id, valid := helper.ParamID(c)
	if !valid {
		return 0, helper.BadRequest("id harus berupa angka positif")
	}
	student, err := s.students.FindByID(ctx, id)
	if err != nil {
		return 0, terjemahkanError(err, "gagal mengambil data mahasiswa")
	}
	if !CanAccessStudent(current, student.OwnerID, s.perms, anyPermission) {
		return 0, helper.Forbidden("tidak berhak mengakses portofolio mahasiswa lain")
	}
	return id, nil
}

// normalizeProject merapikan teknologi (huruf kecil, tanpa duplikat) supaya
// statistik teknologi bermakna.
func normalizeProject(p model.ProjectInput) model.ProjectInput {
	p.Technologies = NormalizeTags(p.Technologies)
	return p
}

// ---------- GET /students/:id/portfolio ----------
func (s *PortfolioService) Get(c *fiber.Ctx) error {
	studentID, err := s.authorize(c, "student:read:any")
	if err != nil {
		return err
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	p, err := s.portfolios.FindByStudent(ctx, studentID)
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusOK, "portofolio ditemukan", p)
}

// ---------- PUT /students/:id/portfolio ----------
func (s *PortfolioService) Put(c *fiber.Ctx) error {
	studentID, err := s.authorize(c, "student:update:any")
	if err != nil {
		return err
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.PutPortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	for i := range req.Projects {
		req.Projects[i] = normalizeProject(req.Projects[i])
	}
	if verr := helper.ValidateStruct(req); verr != nil {
		return verr
	}
	p, err := s.portfolios.Put(ctx, studentID, req)
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusOK, "portofolio berhasil disimpan", p)
}

// ---------- POST /students/:id/portfolio/projects ----------
func (s *PortfolioService) AddProject(c *fiber.Ctx) error {
	studentID, err := s.authorize(c, "student:update:any")
	if err != nil {
		return err
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.ProjectInput
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req = normalizeProject(req)
	if verr := helper.ValidateStruct(req); verr != nil {
		return verr
	}
	p, err := s.portfolios.AddProject(ctx, studentID, model.Project{
		Title: req.Title, Technologies: req.Technologies, URL: req.URL, Year: req.Year,
	})
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusCreated, "proyek berhasil ditambahkan", p)
}

// ---------- DELETE /students/:id/portfolio/projects/:projectId ----------
func (s *PortfolioService) RemoveProject(c *fiber.Ctx) error {
	studentID, err := s.authorize(c, "student:update:any")
	if err != nil {
		return err
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	p, err := s.portfolios.RemoveProject(ctx, studentID, c.Params("projectId"))
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusOK, "proyek berhasil dihapus", p)
}

// ---------- GET /portfolios/stats/technologies ----------
func (s *PortfolioService) TopTechnologies(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	stats, err := s.portfolios.TopTechnologies(ctx, 10)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "statistik teknologi berhasil diambil", stats)
}

func translatePortfolioError(err error) error {
	switch {
	case errors.Is(err, repository.ErrInvalidID):
		return helper.BadRequest("id proyek tidak valid")
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("portofolio atau proyek tidak ditemukan")
	case errors.Is(err, repository.ErrProjectLimit):
		return helper.NewError(fiber.StatusConflict, "PROJECT_LIMIT",
			"jumlah proyek sudah mencapai batas")
	default:
		return helper.Internal(err)
	}
}
