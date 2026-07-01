package profile

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// arrayFields are the only user_profile columns exposed as JSONB item lists.
// Column names are interpolated into SQL below, so this whitelist is what
// keeps that safe — never accept a field name from the request directly.
var arrayFields = map[string]bool{
	"skills":     true,
	"experience": true,
	"education":  true,
	"projects":   true,
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type Profile struct {
	ID                 int64           `json:"id"`
	FullName           string          `json:"full_name"`
	Email              string          `json:"email"`
	Phone              string          `json:"phone"`
	Location           string          `json:"location"`
	BaseResumeDocxPath string          `json:"base_resume_docx_path"`
	Skills             json.RawMessage `json:"skills"`
	Experience         json.RawMessage `json:"experience"`
	Education          json.RawMessage `json:"education"`
	Projects           json.RawMessage `json:"projects"`
}

type Basics struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Location string `json:"location"`
}

func ValidField(field string) bool {
	return arrayFields[field]
}

func (s *Store) Get(ctx context.Context) (Profile, error) {
	var p Profile
	err := s.pool.QueryRow(ctx, `
		SELECT id, full_name, email, phone, location, base_resume_docx_path, skills, experience, education, projects
		FROM user_profile WHERE id = 1
	`).Scan(&p.ID, &p.FullName, &p.Email, &p.Phone, &p.Location, &p.BaseResumeDocxPath, &p.Skills, &p.Experience, &p.Education, &p.Projects)
	return p, err
}

func (s *Store) UpdateBasics(ctx context.Context, b Basics) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE user_profile SET full_name = $1, email = $2, phone = $3, location = $4, updated_at = now()
		WHERE id = 1
	`, b.FullName, b.Email, b.Phone, b.Location)
	return err
}

func (s *Store) SetResumePath(ctx context.Context, path string) error {
	_, err := s.pool.Exec(ctx, `UPDATE user_profile SET base_resume_docx_path = $1, updated_at = now() WHERE id = 1`, path)
	return err
}

// AddItem appends item to the given array field, assigning it a fresh "id".
func (s *Store) AddItem(ctx context.Context, field string, item map[string]any) (map[string]any, error) {
	if !ValidField(field) {
		return nil, fmt.Errorf("invalid field: %s", field)
	}
	id, err := randID()
	if err != nil {
		return nil, err
	}
	item["id"] = id

	itemJSON, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		UPDATE user_profile
		SET %s = %s || $1::jsonb, updated_at = now()
		WHERE id = 1
	`, field, field)
	if _, err := s.pool.Exec(ctx, query, fmt.Sprintf("[%s]", itemJSON)); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Store) UpdateItem(ctx context.Context, field, id string, item map[string]any) error {
	if !ValidField(field) {
		return fmt.Errorf("invalid field: %s", field)
	}
	item["id"] = id
	itemJSON, err := json.Marshal(item)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		UPDATE user_profile
		SET %s = (
			SELECT jsonb_agg(CASE WHEN elem->>'id' = $1 THEN $2::jsonb ELSE elem END)
			FROM jsonb_array_elements(%s) elem
		), updated_at = now()
		WHERE id = 1
	`, field, field)
	_, err = s.pool.Exec(ctx, query, id, string(itemJSON))
	return err
}

func (s *Store) DeleteItem(ctx context.Context, field, id string) error {
	if !ValidField(field) {
		return fmt.Errorf("invalid field: %s", field)
	}
	query := fmt.Sprintf(`
		UPDATE user_profile
		SET %s = COALESCE((
			SELECT jsonb_agg(elem)
			FROM jsonb_array_elements(%s) elem
			WHERE elem->>'id' IS DISTINCT FROM $1
		), '[]'::jsonb), updated_at = now()
		WHERE id = 1
	`, field, field)
	_, err := s.pool.Exec(ctx, query, id)
	return err
}

// NewID generates the same kind of id used for items added via AddItem, so
// callers seeding array data up front (e.g. cmd/seed) can make every item
// addressable for update/delete from the start.
func NewID() (string, error) {
	return randID()
}

func randID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
