package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"api-students/app/model"
)

var ErrProjectLimit = errors.New("jumlah proyek sudah mencapai batas")

// MaxProjectsPerPortfolio membatasi pertumbuhan array projects.
const MaxProjectsPerPortfolio = 20

type PortfolioRepository interface {
	EnsureIndexes(ctx context.Context) error
	FindByStudent(ctx context.Context, studentID int) (model.Portfolio, error)
	Put(ctx context.Context, studentID int, req model.PutPortfolioRequest) (model.Portfolio, error)
	AddProject(ctx context.Context, studentID int, p model.Project) (model.Portfolio, error)
	RemoveProject(ctx context.Context, studentID int, projectID string) (model.Portfolio, error)
	TopTechnologies(ctx context.Context, limit int) ([]model.TechCount, error)
}

// portfolioDocument adalah bentuk portofolio SEBAGAIMANA TERSIMPAN di MongoDB.
type portfolioDocument struct {
	ID           bson.ObjectID         `bson:"_id,omitempty"`
	StudentID    int                   `bson:"student_id"`
	Headline     string                `bson:"headline"`
	Bio          string                `bson:"bio"`
	Projects     []projectDocument     `bson:"projects"`
	Certificates []certificateDocument `bson:"certificates"`
	CreatedAt    time.Time             `bson:"created_at"`
	UpdatedAt    time.Time             `bson:"updated_at"`
}

type projectDocument struct {
	ID           bson.ObjectID `bson:"_id"`
	Title        string        `bson:"title"`
	Technologies []string      `bson:"technologies"`
	URL          string        `bson:"url"`
	Year         int           `bson:"year"`
}

type certificateDocument struct {
	Name     string `bson:"name"`
	Issuer   string `bson:"issuer"`
	IssuedAt string `bson:"issued_at"`
}

func (d portfolioDocument) toModel() model.Portfolio {
	projects := make([]model.Project, 0, len(d.Projects))
	for _, p := range d.Projects {
		techs := p.Technologies
		if techs == nil {
			techs = []string{}
		}
		projects = append(projects, model.Project{
			ID: p.ID.Hex(), Title: p.Title, Technologies: techs, URL: p.URL, Year: p.Year,
		})
	}
	certs := make([]model.Certificate, 0, len(d.Certificates))
	for _, c := range d.Certificates {
		certs = append(certs, model.Certificate{Name: c.Name, Issuer: c.Issuer, IssuedAt: c.IssuedAt})
	}
	return model.Portfolio{
		StudentID: d.StudentID, Headline: d.Headline, Bio: d.Bio,
		Projects: projects, Certificates: certs,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

type portfolioMongoRepository struct{ coll *mongo.Collection }

func NewPortfolioRepository(db *mongo.Database) PortfolioRepository {
	return &portfolioMongoRepository{coll: db.Collection("student_portfolios")}
}

func (r *portfolioMongoRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			// Satu mahasiswa hanya boleh memiliki SATU portofolio.
			// Aturan ini dijaga database, bukan hanya kode.
			Keys:    bson.D{{Key: "student_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("student_unique"),
		},
	})
	if err != nil {
		return fmt.Errorf("membuat index student_portfolios: %w", err)
	}
	return nil
}

func (r *portfolioMongoRepository) FindByStudent(ctx context.Context, studentID int) (model.Portfolio, error) {
	var doc portfolioDocument
	if err := r.coll.FindOne(ctx, bson.M{"student_id": studentID}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Portfolio{}, ErrNotFound
		}
		return model.Portfolio{}, fmt.Errorf("mengambil portofolio: %w", err)
	}
	return doc.toModel(), nil
}

// Put membuat atau mengganti portofolio dalam satu operasi (upsert).
// created_at hanya diisi saat dokumen baru dibuat ($setOnInsert).
func (r *portfolioMongoRepository) Put(ctx context.Context, studentID int, req model.PutPortfolioRequest) (model.Portfolio, error) {
	projects := make([]projectDocument, 0, len(req.Projects))
	for _, p := range req.Projects {
		techs := p.Technologies
		if techs == nil {
			techs = []string{}
		}
		projects = append(projects, projectDocument{
			ID: bson.NewObjectID(), Title: p.Title, Technologies: techs, URL: p.URL, Year: p.Year,
		})
	}
	certs := make([]certificateDocument, 0, len(req.Certificates))
	for _, c := range req.Certificates {
		certs = append(certs, certificateDocument{Name: c.Name, Issuer: c.Issuer, IssuedAt: c.IssuedAt})
	}

	t := now()
	update := bson.M{
		"$set": bson.M{
			"headline":     req.Headline,
			"bio":          req.Bio,
			"projects":     projects,
			"certificates": certs,
			"updated_at":   t,
		},
		"$setOnInsert": bson.M{"created_at": t},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	// Dua PUT serentak untuk mahasiswa baru bisa sama-sama mencoba menyisipkan;
	// index unik menolak salah satunya. Ulangi sekali: kini dokumennya sudah ada.
	for attempt := 0; attempt < 2; attempt++ {
		var doc portfolioDocument
		err := r.coll.FindOneAndUpdate(ctx, bson.M{"student_id": studentID}, update, opts).Decode(&doc)
		if err == nil {
			return doc.toModel(), nil
		}
		if mongo.IsDuplicateKeyError(err) {
			continue
		}
		return model.Portfolio{}, fmt.Errorf("menyimpan portofolio: %w", err)
	}
	return model.Portfolio{}, fmt.Errorf("menyimpan portofolio: bentrokan berulang")
}

// AddProject menambah satu proyek. Syarat "projects.19 tidak ada" berarti
// array belum berisi 20 elemen; pemeriksaan dan $push dalam satu operasi.
func (r *portfolioMongoRepository) AddProject(ctx context.Context, studentID int, p model.Project) (model.Portfolio, error) {
	techs := p.Technologies
	if techs == nil {
		techs = []string{}
	}
	doc := projectDocument{
		ID: bson.NewObjectID(), Title: p.Title, Technologies: techs, URL: p.URL, Year: p.Year,
	}
	limitField := "projects." + strconv.Itoa(MaxProjectsPerPortfolio-1)

	var updated portfolioDocument
	err := r.coll.FindOneAndUpdate(ctx,
		bson.M{"student_id": studentID, limitField: bson.M{"$exists": false}},
		bson.M{
			"$push": bson.M{"projects": doc},
			"$set":  bson.M{"updated_at": now()},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)
	if err == nil {
		return updated.toModel(), nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return model.Portfolio{}, fmt.Errorf("menambah proyek: %w", err)
	}
	count, err := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
	if err != nil {
		return model.Portfolio{}, fmt.Errorf("memeriksa portofolio: %w", err)
	}
	if count == 0 {
		return model.Portfolio{}, ErrNotFound
	}
	return model.Portfolio{}, ErrProjectLimit
}

// RemoveProject menghapus satu proyek dengan $pull.
func (r *portfolioMongoRepository) RemoveProject(ctx context.Context, studentID int, projectID string) (model.Portfolio, error) {
	oid, err := parseID(projectID)
	if err != nil {
		return model.Portfolio{}, err
	}
	var updated portfolioDocument
	err = r.coll.FindOneAndUpdate(ctx,
		bson.M{"student_id": studentID, "projects._id": oid},
		bson.M{
			"$pull": bson.M{"projects": bson.M{"_id": oid}},
			"$set":  bson.M{"updated_at": now()},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)
	if err == nil {
		return updated.toModel(), nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return model.Portfolio{}, ErrNotFound
	}
	return model.Portfolio{}, fmt.Errorf("menghapus proyek: %w", err)
}

// TopTechnologies menghitung teknologi terpopuler di seluruh portofolio.
// Dua kali $unwind: pertama memecah array projects, kedua memecah array
// technologies di dalam tiap proyek.
func (r *portfolioMongoRepository) TopTechnologies(ctx context.Context, limit int) ([]model.TechCount, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$projects"}},
		{{Key: "$unwind", Value: "$projects.technologies"}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$projects.technologies",
			"total": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "total", Value: -1}, {Key: "_id", Value: 1}}}},
		{{Key: "$limit", Value: int64(limit)}},
	}
	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("menghitung teknologi: %w", err)
	}
	defer cursor.Close(ctx)

	var out []struct {
		Tech  string `bson:"_id"`
		Total int    `bson:"total"`
	}
	if err := cursor.All(ctx, &out); err != nil {
		return nil, fmt.Errorf("membaca statistik teknologi: %w", err)
	}
	result := make([]model.TechCount, 0, len(out))
	for _, o := range out {
		result = append(result, model.TechCount{Technology: o.Tech, Total: o.Total})
	}
	return result, nil
}
