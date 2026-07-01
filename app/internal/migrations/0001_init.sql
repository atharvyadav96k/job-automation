CREATE TABLE user_profile (
    id BIGSERIAL PRIMARY KEY,
    full_name TEXT NOT NULL,
    email TEXT NOT NULL,
    phone TEXT,
    location TEXT,
    base_resume_docx_path TEXT,
    skills JSONB NOT NULL DEFAULT '[]',
    experience JSONB NOT NULL DEFAULT '[]',
    education JSONB NOT NULL DEFAULT '[]',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE companies (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    careers_url TEXT,
    ats_platform TEXT, -- 'greenhouse', 'lever', 'workday', 'direct'
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE jobs (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT NOT NULL REFERENCES companies(id),
    title TEXT NOT NULL,
    url TEXT NOT NULL,
    description_raw TEXT,
    description_clean TEXT,
    source TEXT NOT NULL, -- 'scraper', 'api', 'manual'
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    status TEXT NOT NULL DEFAULT 'new' -- 'new', 'scored', 'skipped', 'queued', 'applied'
);
CREATE UNIQUE INDEX jobs_url_idx ON jobs(url);

CREATE TABLE job_context (
    id BIGSERIAL PRIMARY KEY,
    job_id BIGINT NOT NULL REFERENCES jobs(id),
    company_summary TEXT,
    key_requirements JSONB NOT NULL DEFAULT '[]',
    inferred_tone TEXT,
    research_sources JSONB NOT NULL DEFAULT '[]',
    model_used TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE resume_versions (
    id BIGSERIAL PRIMARY KEY,
    job_id BIGINT NOT NULL REFERENCES jobs(id),
    job_context_id BIGINT REFERENCES job_context(id),
    version_number INT NOT NULL,
    generated_resume_docx_path TEXT,
    generated_cover_letter_text TEXT,
    ats_score INT,
    ats_score_breakdown JSONB,
    changes_summary TEXT,
    reasoning TEXT,
    model_used TEXT,
    prompt_tokens INT,
    completion_tokens INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved BOOLEAN NOT NULL DEFAULT false,
    approved_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true
);
CREATE UNIQUE INDEX resume_versions_job_version_idx ON resume_versions(job_id, version_number);

CREATE TABLE applications (
    id BIGSERIAL PRIMARY KEY,
    job_id BIGINT NOT NULL REFERENCES jobs(id),
    resume_version_id BIGINT NOT NULL REFERENCES resume_versions(id),
    method TEXT NOT NULL, -- 'browser_auto', 'email', 'manual'
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'submitted', 'failed', 'needs_review'
    screenshot_path TEXT,
    submitted_at TIMESTAMPTZ,
    error_message TEXT
);
