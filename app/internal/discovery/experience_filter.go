package discovery

import (
	"regexp"
	"strings"
)

// seniorLevelTitleMarkers are title keywords that signal a job wants
// meaningfully more experience than a ~1 year candidate has. Simple
// substring match on the title, matching this codebase's "start with
// keyword matching" approach elsewhere (see skill gap matching).
var seniorLevelTitleMarkers = []string{
	"senior", "sr.", "sr ", "staff", "principal", "lead ", "leader",
	"architect", "director", "head of", "vp ", "vice president", "manager",
}

// yearsRequirementRe catches an explicit years-of-experience figure in the
// title (e.g. "10+ years", "5+ years, Bangalore") — titles that spell this
// out tend to state a number a ~1 year candidate has no chance at,
// regardless of whether they also use a seniority word like "senior".
var yearsRequirementRe = regexp.MustCompile(`\b([3-9]|\d{2,})\+?\s*years?\b`)

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
	return yearsRequirementRe.MatchString(lower)
}
