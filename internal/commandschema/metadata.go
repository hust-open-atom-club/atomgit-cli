// Package commandschema describes registered Cobra commands without executing
// their validators, completion functions, or business logic.
package commandschema

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

const annotationKey = "atomgit.command-description"

// Metadata supplements facts Cobra cannot expose. Keep it next to the command
// registration, not in a second command catalogue. Unspecified facts remain
// undescribed; these annotations do not change command execution.
type Metadata struct {
	Positionals   *Positionals `json:"positionals,omitempty"`
	RequiredFlags []string     `json:"requiredFlags,omitempty"`
	Output        string       `json:"output,omitempty"`
	Effects       string       `json:"effects,omitempty"`
	Notes         []string     `json:"notes,omitempty"`
}

// Positionals describes argument counts and semantics explicitly. MaxCount -1
// means unbounded. Additional business validation may still apply.
type Positionals struct {
	MinCount    int    `json:"minCount"`
	MaxCount    int    `json:"maxCount"`
	Description string `json:"description"`
}

// Annotate attaches static metadata only. Metadata contains no runtime values.
func Annotate(cmd *cobra.Command, metadata Metadata) {
	// The fixed, JSON-compatible Metadata type cannot fail to marshal.
	data, _ := json.Marshal(metadata)
	if cmd.Annotations == nil {
		cmd.Annotations = make(map[string]string)
	}
	cmd.Annotations[annotationKey] = string(data)
}
