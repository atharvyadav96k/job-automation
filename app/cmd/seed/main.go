// Loads (or updates) the single user_profile row from a JSON file matching
// data/profile.example.json. Run once to seed Phase 0 data, and again
// whenever the profile changes by hand before the Phase 0.5 UI exists.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"job-automation/app/internal/config"
	"job-automation/app/internal/db"
	"job-automation/app/internal/profile"
)

type profileFile struct {
	FullName           string           `json:"full_name"`
	Email              string           `json:"email"`
	Phone              string           `json:"phone"`
	Location           string           `json:"location"`
	BaseResumeDocxPath string           `json:"base_resume_docx_path"`
	Skills             []map[string]any `json:"skills"`
	Experience         []map[string]any `json:"experience"`
	Education          []map[string]any `json:"education"`
	Projects           []map[string]any `json:"projects"`
}

// backfillIDs assigns an id to any item missing one, so everything seeded
// up front is addressable via the profile API's update/delete endpoints.
func backfillIDs(items []map[string]any) ([]map[string]any, error) {
	for _, item := range items {
		if _, ok := item["id"]; ok {
			continue
		}
		id, err := profile.NewID()
		if err != nil {
			return nil, err
		}
		item["id"] = id
	}
	return items, nil
}

func main() {
	path := flag.String("file", "data/profile.json", "path to profile JSON file")
	flag.Parse()

	raw, err := os.ReadFile(*path)
	if err != nil {
		log.Fatalf("read profile file %s: %v", *path, err)
	}
	var p profileFile
	if err := json.Unmarshal(raw, &p); err != nil {
		log.Fatalf("parse profile file: %v", err)
	}

	for _, items := range [][]map[string]any{p.Skills, p.Experience, p.Education, p.Projects} {
		if _, err := backfillIDs(items); err != nil {
			log.Fatalf("generate id: %v", err)
		}
	}
	// jsonb columns are NOT NULL; an absent JSON key unmarshals to a nil
	// slice, which would otherwise be sent as SQL NULL.
	for _, field := range []*[]map[string]any{&p.Skills, &p.Experience, &p.Education, &p.Projects} {
		if *field == nil {
			*field = []map[string]any{}
		}
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	var id int64
	err = pool.QueryRow(ctx, `
		INSERT INTO user_profile (id, full_name, email, phone, location, base_resume_docx_path, skills, experience, education, projects, updated_at)
		VALUES (1, $1, $2, $3, $4, $5, $6, $7, $8, $9, now())
		ON CONFLICT (id) DO UPDATE SET
			full_name = EXCLUDED.full_name,
			email = EXCLUDED.email,
			phone = EXCLUDED.phone,
			location = EXCLUDED.location,
			base_resume_docx_path = EXCLUDED.base_resume_docx_path,
			skills = EXCLUDED.skills,
			experience = EXCLUDED.experience,
			education = EXCLUDED.education,
			projects = EXCLUDED.projects,
			updated_at = now()
		RETURNING id
	`, p.FullName, p.Email, p.Phone, p.Location, p.BaseResumeDocxPath, p.Skills, p.Experience, p.Education, p.Projects).Scan(&id)
	if err != nil {
		log.Fatalf("upsert user_profile: %v", err)
	}

	log.Printf("seeded user_profile id=%d", id)
}
