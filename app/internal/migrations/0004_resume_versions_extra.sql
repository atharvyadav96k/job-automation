-- match_score is the LLM's overall fit estimate, distinct from ats_score
-- (keyword-match estimate). tailored_content holds the structured tailoring
-- output (skills/bullets/projects) so the review UI can diff versions
-- without parsing the generated .docx binary.
ALTER TABLE resume_versions ADD COLUMN match_score INT;
ALTER TABLE resume_versions ADD COLUMN tailored_content JSONB;
