package models

type LintConfig struct {
	Errors   ErrorSection    `json:"errors"`
	Warnings WarningSection  `json:"warnings"`
	Settings SettingsSection `json:"settings"`
}

type ErrorSection struct {
	UnusedFunctions    bool `json:"unused_functions"`
	UnusedSignals      bool `json:"unused_signals"`
	UnusedConstants    bool `json:"unused_constants"`
	UnusedEnums        bool `json:"unused_enums"`
	UnusedClassNames   bool `json:"unused_class_names"`
	UnusedFiles        bool `json:"unused_files"`
	PrintStatements    bool `json:"print_statements"`
	Comments           bool `json:"comments"`
	PassStatements     bool `json:"pass_statements"`
	OrphanedUIDs       bool `json:"orphaned_uids"`
	Indentation        bool `json:"indentation"`
	HasMethod          bool `json:"has_method"`
	VariantUsage       bool `json:"variant_usage"`
	MissingReturnTypes bool `json:"missing_return_types"`
}

type WarningSection struct {
	SingleUseFunctions bool `json:"single_use_functions"`
	EmptyFunctions     bool `json:"empty_functions"`
}

type SettingsSection struct {
	AllowedCommentPrefixes []string `json:"allowed_comment_prefixes"`
	IgnorePatterns         []string `json:"ignore_patterns"`
	UnusedIgnorePatterns   []string `json:"unused_ignore_patterns"`
	VariantIgnorePatterns  []string `json:"variant_ignore_patterns"`
}

func NewDefaultConfig() *LintConfig {
	return &LintConfig{
		Errors: ErrorSection{
			UnusedFunctions:    false,
			UnusedSignals:      false,
			UnusedConstants:    false,
			UnusedEnums:        false,
			UnusedClassNames:   false,
			UnusedFiles:        false,
			PrintStatements:    true,
			Comments:           false,
			PassStatements:     true,
			OrphanedUIDs:       true,
			Indentation:        true,
			HasMethod:          false,
			VariantUsage:       false,
			MissingReturnTypes: false,
		},
		Warnings: WarningSection{
			SingleUseFunctions: false,
			EmptyFunctions:     false,
		},
		Settings: SettingsSection{
			AllowedCommentPrefixes: []string{"[TECH DEBT]"},
			IgnorePatterns: []string{
				"*.tmp", "*.backup", ".import/",
			},
			UnusedIgnorePatterns:  []string{},
			VariantIgnorePatterns: []string{},
		},
	}
}
