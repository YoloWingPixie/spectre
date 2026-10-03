// Package auditdocs embeds the audit skill, guides, and schemas.
package auditdocs

import "embed"

// Files contains the instructions and schemas shipped with the executable.
//
//go:embed SKILL.md checklist.md resuming-audits.md writing-guide.md schema templates
var Files embed.FS
