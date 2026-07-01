package resume

import "strconv"

// Placeholder keys the base .docx template must contain (single curly
// braces, e.g. {SKILLS_BACKEND} — see .claude/skills/docx-templating).
// Keep this list in sync with the actual template: a mismatch here
// silently produces resumes with literal "{PLACEHOLDER}" text left in them.
//
// v1 scope only covers Skills + the most recent role's bullets — the base
// template has no Summary/Projects sections yet.
const (
	SkillsBackend     = "SKILLS_BACKEND"
	SkillsFrontend    = "SKILLS_FRONTEND"
	SkillsDatabase    = "SKILLS_DATABASE"
	SkillsIntegration = "SKILLS_INTEGRATION"
	SkillsDevops      = "SKILLS_DEVOPS"
	SkillsCloud       = "SKILLS_CLOUD"
	SkillsTools       = "SKILLS_TOOLS"
)

// Exp1BulletCount is how many bullet placeholders the template defines for
// the most recent role (EXP1_BULLET1 .. EXP1_BULLETN).
const Exp1BulletCount = 8

func Exp1Bullet(n int) string {
	return "EXP1_BULLET" + strconv.Itoa(n)
}

// SkillCategoryKeys maps user_profile skill categories to the template
// placeholder that should receive that category's skill list.
var SkillCategoryKeys = map[string]string{
	"backend":     SkillsBackend,
	"frontend":    SkillsFrontend,
	"database":    SkillsDatabase,
	"integration": SkillsIntegration,
	"devops":      SkillsDevops,
	"cloud":       SkillsCloud,
	"tools":       SkillsTools,
}
