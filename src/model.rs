use regex::Regex;
use serde::Deserialize;
use serde_json::Value;
use std::{collections::BTreeMap, path::Path, sync::LazyLock};

pub type Result<T> = std::result::Result<T, Box<dyn std::error::Error>>;
pub const DEFAULTS: &str = include_str!("defaults.json");

#[derive(Deserialize)]
pub struct Config {
    pub errors: BTreeMap<String, bool>,
    pub warnings: BTreeMap<String, bool>,
    pub settings: Settings,
}
#[derive(Deserialize)]
pub struct Settings {
    pub allowed_comment_prefixes: Vec<String>,
    pub ignore_patterns: Vec<String>,
    pub unused_ignore_patterns: Vec<String>,
    pub variant_ignore_patterns: Vec<String>,
}
impl Config {
    pub fn load(root: &Path) -> Result<Self> {
        let mut value: Value = serde_json::from_str(DEFAULTS)?;
        match std::fs::read(root.join("gdlint.json")) {
            Ok(bytes) => {
                let supplied: Value = serde_json::from_slice(&bytes)
                    .map_err(|e| format!("invalid config file: {e}"))?;
                merge(&mut value, supplied).map_err(|e| format!("invalid config file: {e}"))?;
            }
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {}
            Err(e) => return Err(e.into()),
        }
        Ok(serde_json::from_value(value).map_err(|e| format!("invalid config file: {e}"))?)
    }
    pub fn enabled(&self, key: &str) -> bool {
        self.errors.get(key).copied().unwrap_or(false)
    }
    pub fn warn(&self, key: &str) -> bool {
        self.warnings.get(key).copied().unwrap_or(false)
    }
}
fn merge(base: &mut Value, supplied: Value) -> Result<()> {
    if supplied.is_null() {
        if base.is_array() {
            *base = Value::Array(Vec::new());
        }
        return Ok(());
    }
    match (base, supplied) {
        (Value::Object(base), Value::Object(supplied)) => {
            for (key, value) in supplied {
                if let Some(field) = base.get_mut(&key.to_ascii_lowercase()) {
                    merge(field, value)?;
                }
            }
        }
        (base @ Value::Bool(_), value @ Value::Bool(_)) => *base = value,
        (base @ Value::Array(_), Value::Array(values)) => {
            // encoding/json leaves a null string element as the zero string.
            let values: Vec<_> = values
                .into_iter()
                .map(|v| {
                    if v.is_null() {
                        Value::String(String::new())
                    } else {
                        v
                    }
                })
                .collect();
            if values.iter().any(|v| !v.is_string()) {
                return Err("settings must contain string arrays".into());
            }
            *base = Value::Array(values);
        }
        _ => return Err("expected a configuration object, boolean option or string array".into()),
    }
    Ok(())
}

static PATTERNS: LazyLock<BTreeMap<String, Regex>> = LazyLock::new(|| {
    let values: BTreeMap<String, String> =
        serde_json::from_str(include_str!("patterns.json")).expect("embedded patterns");
    values
        .into_iter()
        .map(|(key, value)| {
            let pattern =
                Regex::new(&value).unwrap_or_else(|e| panic!("invalid embedded {key}: {e}"));
            (key, pattern)
        })
        .collect()
});
pub fn pattern(name: &str) -> &'static Regex {
    &PATTERNS[name]
}
pub fn indent(line: &str) -> &str {
    pattern("IndentPattern")
        .captures(line)
        .and_then(|c| c.get(1))
        .map_or("", |m| m.as_str())
}
pub fn level(line: &str) -> usize {
    let text = indent(line);
    text.bytes().filter(|b| *b == b'\t').count() + text.bytes().filter(|b| *b == b' ').count() / 4
}
pub fn empty(line: &str) -> bool {
    line[indent(line).len()..].is_empty() || pattern("CommentPattern").is_match(line)
}
pub fn remove_comments(line: &str) -> String {
    let (mut quoted, mut escaped) = (false, false);
    let mut result = String::new();
    for ch in line.chars() {
        if escaped {
            escaped = false;
        } else if ch == '\\' {
            escaped = true;
        } else if ch == '\'' || ch == '"' {
            quoted = !quoted;
        } else if ch == '#' && !quoted {
            break;
        }
        result.push(ch);
    }
    result
}
pub fn callback(name: &str) -> bool {
    name.starts_with("_on_")
        || matches!(
            name,
            "_ready"
                | "_init"
                | "_enter_tree"
                | "_exit_tree"
                | "_process"
                | "_physics_process"
                | "_input"
                | "_unhandled_input"
                | "_draw"
                | "_gui_input"
                | "_notification"
                | "_get"
                | "_set"
                | "_get_property_list"
                | "_property_can_revert"
                | "_property_get_revert"
                | "_to_string"
                | "_get_configuration_warnings"
        )
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Kind {
    Function,
    Signal,
    Constant,
    Enum,
    EnumValue,
    ClassName,
}
#[derive(Clone, Debug)]
pub struct Entity {
    pub kind: Kind,
    pub name: String,
    pub parent: String,
    pub file: String,
    pub line: usize,
    pub end: usize,
    pub rpc: bool,
}
impl Entity {
    pub fn full_name(&self) -> String {
        if self.parent.is_empty() {
            self.name.clone()
        } else {
            format!("{}.{}", self.parent, self.name)
        }
    }
    pub fn protected(&self) -> bool {
        self.kind == Kind::ClassName
            || (self.kind == Kind::Function
                && (callback(&self.name)
                    || self.name.starts_with("get_")
                    || self.name.starts_with("set_")
                    || self.rpc))
    }
}
pub struct File {
    pub path: std::path::PathBuf,
    pub name: String,
    pub content: String,
    pub lines: Vec<String>,
    pub entities: Vec<Entity>,
}
#[derive(Clone)]
pub struct Finding {
    pub file: String,
    pub line: usize,
    pub text: String,
    pub entity: Option<Entity>,
}
impl Finding {
    pub fn location(file: &File, line: usize, text: impl Into<String>) -> Self {
        Self {
            file: file.name.clone(),
            line,
            text: text.into(),
            entity: None,
        }
    }
    pub fn entity(entity: &Entity) -> Self {
        Self {
            file: entity.file.clone(),
            line: entity.line,
            text: entity.name.clone(),
            entity: Some(entity.clone()),
        }
    }
    pub fn file(file: String) -> Self {
        Self {
            file,
            line: 0,
            text: String::new(),
            entity: None,
        }
    }
}
pub const TITLES: [&str; 17] = [
    "Unused functions",
    "Unused signals",
    "Unused constants",
    "Unused enums",
    "Unused enum values",
    "Unused class names",
    "Unused files",
    "Print statements",
    "Comments to remove",
    "Unnecessary pass statements",
    "Orphaned UID files",
    "Indentation issues",
    "has_method() calls",
    "Explicit Variant usages (disallowed once enabled)",
    "Missing return type annotations",
    "Single-use functions",
    "Empty functions",
];
pub struct Findings {
    pub items: [Vec<Finding>; 17],
}
impl Findings {
    pub fn new() -> Self {
        Self {
            items: std::array::from_fn(|_| Vec::new()),
        }
    }
    pub fn errors(&self) -> usize {
        self.items[..15].iter().map(Vec::len).sum()
    }
    pub fn warnings(&self) -> usize {
        self.items[15..].iter().map(Vec::len).sum()
    }
}
