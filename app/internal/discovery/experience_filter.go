package discovery

import "strings"

// seniorLevelTitleMarkers are title keywords that signal a job wants
// meaningfully more experience than a ~1 year candidate has. Simple
// substring match on the title, matching this codebase's "start with
// keyword matching" approach elsewhere (see skill gap matching).
var seniorLevelTitleMarkers = []string{
	"senior", "sr.", "sr ", "staff", "principal", "lead ", "architect",
	"director", "head of", "vp ", "vice president", "manager",
}

// IsSeniorLevelTitle reports whether a job title reads as wanting more
// experience than a ~1 year candidate has — used to filter discovery
// results down to roles actually worth applying to.
func IsSeniorLevelTitle(title string) bool {
	lower := strings.ToLower(title)
	for _, marker := range seniorLevelTitleMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
