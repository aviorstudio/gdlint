package patterns

import "regexp"

var (
	FunctionPattern  = regexp.MustCompile(`^(\s*)(?:static\s+)?func\s+(\w+)\s*\(`)
	SignalPattern    = regexp.MustCompile(`^(\s*)signal\s+(\w+)`)
	ConstantPattern  = regexp.MustCompile(`^(\s*)const\s+([A-Z_][A-Z0-9_]*)\s*[=:]`)
	EnumPattern      = regexp.MustCompile(`^(\s*)enum\s+(\w+)\s*\{`)
	EnumValuePattern = regexp.MustCompile(`^\s*([A-Z_][A-Z0-9_]*)\s*(?:[,=]|$)`)
	ClassNamePattern = regexp.MustCompile(`^class_name\s+(\w+)`)
	ExtendsPattern   = regexp.MustCompile(`^extends\s+(\w+)`)
	PreloadPattern   = regexp.MustCompile(`preload\s*\(\s*["']([^"']+)["']\s*\)`)
	LoadPattern      = regexp.MustCompile(`load\s*\(\s*["']([^"']+)["']\s*\)`)

	FunctionCallPattern    = regexp.MustCompile(`\b(\w+)\s*\(`)
	SignalEmitPattern      = regexp.MustCompile(`(\w+)\.emit\s*\(`)
	SignalConnectPattern   = regexp.MustCompile(`(\w+)\.connect\s*\(`)
	ConnectCallbackPattern = regexp.MustCompile(`\.connect\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)`)
	CallableRefPattern     = regexp.MustCompile(`(?:[:,=\[]\s*)(\b_[a-z_][a-z0-9_]*\b)(?:\s*[,\]\}]|\s*$)`)
	ConstantRefPattern     = regexp.MustCompile(`\b([A-Z_][A-Z0-9_]*)\b`)
	EnumRefPattern         = regexp.MustCompile(`\b(\w+)\.([A-Z_][A-Z0-9_]*)\b`)
	TypeAnnotationPattern  = regexp.MustCompile(`:\s*(\w+)(?:\s*[,\)]|\s*=)`)

	PrintPattern     = regexp.MustCompile(`\bprint(?:_debug|_rich|_verbose)?\s*\(`)
	DebugLogPattern  = regexp.MustCompile(`\bDebugLogUtil\.log\s*\(`)
	CommentPattern   = regexp.MustCompile(`^\s*#`)
	PassPattern      = regexp.MustCompile(`^\s*pass\s*$`)
	HasMethodPattern = regexp.MustCompile(`\bhas_method\s*\(`)

	RPCPattern     = regexp.MustCompile(`^\s*@rpc`)
	ExportPattern  = regexp.MustCompile(`^\s*@export`)
	OnreadyPattern = regexp.MustCompile(`^\s*@onready`)
	ToolPattern    = regexp.MustCompile(`^\s*@tool`)

	StringLiteralPattern = regexp.MustCompile(`["']([^"']+)["']`)
	GetNodePattern       = regexp.MustCompile(`get_node\s*\(\s*["']([^"']+)["']\s*\)`)
	NodePathPattern      = regexp.MustCompile(`\$["']?([^"'\s]+)["']?`)

	IndentPattern       = regexp.MustCompile(`^(\s*)`)
	StaticFuncPattern   = regexp.MustCompile(`^\s*static\s+func\s+`)
	GetterSetterPattern = regexp.MustCompile(`^\s*(?:get|set)\s+(\w+)\s*\(`)
	VariantPattern      = regexp.MustCompile(`\bVariant\b`)

	AutoloadPattern   = regexp.MustCompile(`autoload/(\w+)`)
	ScenePathPattern  = regexp.MustCompile(`res://[^"'\s]+\.tscn`)
	ScriptPathPattern = regexp.MustCompile(`res://[^"'\s]+\.gd`)

	TechDebtPattern    = regexp.MustCompile(`#\s*\[TECH DEBT\](.*)`)
	IgnorePrintPattern = regexp.MustCompile(`#\s*gdlint-ignore-print`)
)

func IsProtectedFunction(name string) bool {
	protectedNames := []string{
		"_ready", "_init", "_enter_tree", "_exit_tree",
		"_process", "_physics_process", "_input", "_unhandled_input",
		"_draw", "_gui_input", "_notification", "_get", "_set",
		"_get_property_list", "_property_can_revert", "_property_get_revert",
		"_to_string", "_get_configuration_warnings",
	}

	for _, protected := range protectedNames {
		if name == protected {
			return true
		}
	}

	// Protect signal handlers (_on_*)
	if len(name) >= 4 && name[:4] == "_on_" {
		return true
	}

	return false
}

func IsValidIdentifier(name string) bool {
	if name == "" {
		return false
	}

	validIdentifier := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	return validIdentifier.MatchString(name)
}

func ExtractIndentation(line string) string {
	matches := IndentPattern.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func CountIndentLevel(indent string) int {
	tabs := 0
	spaces := 0

	for _, ch := range indent {
		if ch == '\t' {
			tabs++
		} else if ch == ' ' {
			spaces++
		}
	}

	return tabs + (spaces / 4)
}

func IsEmptyOrComment(line string) bool {
	trimmed := regexp.MustCompile(`^\s*`).ReplaceAllString(line, "")
	return trimmed == "" || CommentPattern.MatchString(line)
}

func RemoveComments(line string) string {
	inString := false
	escapeNext := false
	result := []rune{}

	for _, ch := range line {
		if escapeNext {
			result = append(result, ch)
			escapeNext = false
			continue
		}

		if ch == '\\' {
			result = append(result, ch)
			escapeNext = true
			continue
		}

		if ch == '"' || ch == '\'' {
			inString = !inString
			result = append(result, ch)
			continue
		}

		if ch == '#' && !inString {
			break
		}

		result = append(result, ch)
	}

	return string(result)
}
