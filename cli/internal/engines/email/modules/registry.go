// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

import (
	"github.com/kyle/basalt/internal/engines/email"
)

// All returns all available email check modules.
func All() []email.Module {
	return []email.Module{
		// Tier 1: Simple API checks.
		&Twitter{},
		&Spotify{},
		&Gravatar{},
		&Docker{},
		&Duolingo{},
		&Pinterest{},
		&Imgur{},
		// Tier 2: CSRF-protected forms.
		&Instagram{},
		&GitHub{},
		&Discord{},
		&Snapchat{},
		&Yahoo{},
		// Tier 3: Password recovery with metadata extraction.
		&Adobe{},
		&Office365{},
		&Samsung{},
		&Amazon{},
	}
}
