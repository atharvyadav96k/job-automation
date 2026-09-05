-- Phase 1.5: GitHub-derived skill evidence, feeding the matcher that flags
-- skill_gaps per job. Evidence must trace back to a real repo — never
-- fabricated, per plan.md's evidence-only constraint.
CREATE TABLE repos (
    id BIGSERIAL PRIMARY KEY,
    github_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    github_url TEXT NOT NULL,
    description TEXT,
    language TEXT,
    topics JSONB NOT NULL DEFAULT '[]',
    stars INT NOT NULL DEFAULT 0,
    last_commit_at TIMESTAMPTZ,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX repos_github_id_idx ON repos(github_id);

-- skill_name is derived from repo language + topics only (no dependency-file
-- parsing yet) — confidence is always 'inferred' until a manual entry path
-- is added.
CREATE TABLE skill_evidence (
    id BIGSERIAL PRIMARY KEY,
    skill_name TEXT NOT NULL,
    repo_id BIGINT NOT NULL REFERENCES repos(id),
    confidence TEXT NOT NULL DEFAULT 'inferred',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX skill_evidence_skill_repo_idx ON skill_evidence(skill_name, repo_id);

CREATE TABLE skill_gaps (
    id BIGSERIAL PRIMARY KEY,
    job_id BIGINT NOT NULL REFERENCES jobs(id),
    missing_skill TEXT NOT NULL,
    reviewed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX skill_gaps_job_skill_idx ON skill_gaps(job_id, missing_skill);
