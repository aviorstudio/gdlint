use crate::model::*;
use regex::Regex;
use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    path::{Component, Path, PathBuf},
};

pub fn walk(root: &Path) -> Result<Vec<PathBuf>> {
    fn visit(path: &Path, result: &mut Vec<PathBuf>) -> Result<()> {
        let metadata = fs::symlink_metadata(path)?;
        result.push(path.to_owned());
        if metadata.is_dir() {
            let mut children: Vec<_> = fs::read_dir(path)?
                .map(|e| e.map(|e| e.path()))
                .collect::<std::io::Result<_>>()?;
            children.sort();
            for child in children {
                visit(&child, result)?;
            }
        }
        Ok(())
    }
    let mut result = Vec::new();
    visit(root, &mut result)?;
    Ok(result)
}
pub fn ignored(path: &Path, patterns: &[String]) -> bool {
    let name = path.file_name().unwrap_or_default().to_string_lossy();
    let path = path.to_string_lossy();
    patterns
        .iter()
        .any(|pattern| path.contains(pattern) || glob(pattern, &name))
}
fn glob(pattern: &str, name: &str) -> bool {
    // filepath.Match semantics: match the basename, with ^ for class negation.
    let mut chars = pattern.chars().peekable();
    let mut expression = String::from("\\A");
    while let Some(ch) = chars.next() {
        match ch {
            '*' => expression.push_str("[^/]*"),
            '?' => expression.push_str("[^/]"),
            '\\' => match chars.next() {
                Some(ch) => expression.push_str(&regex::escape(&ch.to_string())),
                None => return false,
            },
            '[' => {
                expression.push('[');
                if chars.peek() == Some(&'^') {
                    expression.push('^');
                    chars.next();
                }
                let mut count = 0;
                loop {
                    match chars.next() {
                        None => return false,
                        Some(']') => {
                            if count == 0 {
                                return false;
                            }
                            expression.push(']');
                            break;
                        }
                        Some('\\') => match chars.next() {
                            Some(ch) => {
                                expression.push_str(&regex::escape(&ch.to_string()));
                                count += 1;
                            }
                            None => return false,
                        },
                        Some(ch) => {
                            if ch != '-' {
                                expression.push_str(&regex::escape(&ch.to_string()));
                            } else {
                                expression.push('-');
                            }
                            count += 1;
                        }
                    }
                }
            }
            ch => expression.push_str(&regex::escape(&ch.to_string())),
        }
    }
    expression.push_str("\\z");
    Regex::new(&expression).is_ok_and(|pattern| pattern.is_match(name))
}
fn build_path(path: &Path) -> bool {
    let mut src = false;
    for part in path.components() {
        if part.as_os_str() == "src" {
            src = true;
        }
        if part.as_os_str() == "build" && !src {
            return true;
        }
    }
    false
}
fn clean(path: PathBuf) -> PathBuf {
    let mut result = PathBuf::new();
    for part in path.components() {
        match part {
            Component::ParentDir => {
                result.pop();
            }
            Component::CurDir => {}
            _ => result.push(part),
        }
    }
    result
}
fn root_join(root: &Path, name: &str) -> PathBuf {
    clean(root.join(name.trim_start_matches('/')))
}
fn resolve(root: &Path, source: &Path, name: &str) -> Option<PathBuf> {
    if name.starts_with("./") || name.starts_with("../") {
        let path = clean(source.parent()?.join(name));
        if path.exists() {
            return Some(path);
        }
    }
    let path = root_join(root, name.strip_prefix("res://").unwrap_or(name));
    path.exists().then_some(path)
}
pub fn detect(file: &mut File) {
    let (mut class, mut current_enum) = (String::new(), String::new());
    let (mut enum_indent, mut rpc_line) = (0, 0);
    for (index, line) in file.lines.iter().enumerate() {
        let number = index + 1;
        if let Some(m) = pattern("ClassNamePattern").captures(line) {
            class = m[1].into();
            file.entities.push(Entity {
                kind: Kind::ClassName,
                name: class.clone(),
                parent: String::new(),
                file: file.name.clone(),
                line: number,
                end: number,
                rpc: false,
            });
        }
        if pattern("RPCPattern").is_match(line) {
            rpc_line = number;
        }
        for (pattern_name, kind) in [
            ("FunctionPattern", Kind::Function),
            ("SignalPattern", Kind::Signal),
            ("ConstantPattern", Kind::Constant),
            ("EnumPattern", Kind::Enum),
        ] {
            if let Some(m) = pattern(pattern_name).captures(line) {
                let end = if kind == Kind::Function {
                    file.lines
                        .iter()
                        .enumerate()
                        .skip(index + 1)
                        .find(|(_, next)| !empty(next) && level(next) <= level(&m[1]))
                        .map_or(file.lines.len(), |(i, _)| i)
                } else {
                    number
                };
                file.entities.push(Entity {
                    kind,
                    name: m[2].into(),
                    parent: class.clone(),
                    file: file.name.clone(),
                    line: number,
                    end,
                    rpc: kind == Kind::Function && rpc_line == number - 1,
                });
                if kind == Kind::Enum {
                    current_enum = m[2].into();
                    enum_indent = level(&m[1]);
                }
            }
        }
        if !current_enum.is_empty() && line.contains('}') && level(line) <= enum_indent {
            current_enum.clear();
        }
        if !current_enum.is_empty()
            && let Some(m) = pattern("EnumValuePattern").captures(line.trim())
        {
            file.entities.push(Entity {
                kind: Kind::EnumValue,
                name: m[1].into(),
                parent: current_enum.clone(),
                file: file.name.clone(),
                line: number,
                end: number,
                rpc: false,
            });
        }
    }
}
type Usages = BTreeMap<String, Vec<(String, usize)>>;
fn usages(files: &[File]) -> Usages {
    let mut entities: BTreeMap<String, Vec<&Entity>> = BTreeMap::new();
    for entity in files.iter().flat_map(|f| &f.entities) {
        entities.entry(entity.full_name()).or_default().push(entity);
    }
    let mut usages: Usages = BTreeMap::new();
    for file in files {
        for (index, line) in file.lines.iter().enumerate() {
            let line = remove_comments(line);
            let mut add = |name: &str| {
                usages
                    .entry(name.into())
                    .or_default()
                    .push((file.name.clone(), index + 1))
            };
            for key in [
                "FunctionCallPattern",
                "SignalEmitPattern",
                "SignalConnectPattern",
                "ConnectCallbackPattern",
                "CallableRefPattern",
                "ConstantRefPattern",
            ] {
                for m in pattern(key).captures_iter(&line) {
                    let name = &m[1];
                    if matches!(
                        key,
                        "FunctionCallPattern" | "ConnectCallbackPattern" | "CallableRefPattern"
                    ) && callback(name)
                    {
                        continue;
                    }
                    if key == "CallableRefPattern"
                        && !entities
                            .keys()
                            .any(|key| key == name || key.ends_with(&format!(".{name}")))
                    {
                        continue;
                    }
                    add(name);
                }
            }
            for m in pattern("EnumRefPattern").captures_iter(&line) {
                add(&m[1]);
                add(&format!("{}.{}", &m[1], &m[2]));
            }
            for m in pattern("TypeAnnotationPattern").captures_iter(&line) {
                if entities.contains_key(&m[1]) {
                    add(&m[1]);
                }
            }
            for m in pattern("StringLiteralPattern").captures_iter(&line) {
                for (name, entities) in &entities {
                    if m[1].contains(name) || entities.iter().any(|e| m[1].contains(&e.name)) {
                        add(name);
                    }
                }
            }
        }
    }
    usages
}
fn used(entity: &Entity, usages: &Usages) -> bool {
    if entity.protected()
        || usages
            .get(&entity.full_name())
            .is_some_and(|u| !u.is_empty())
    {
        return true;
    }
    let Some(uses) = usages.get(&entity.name).filter(|u| !u.is_empty()) else {
        return false;
    };
    if entity.kind == Kind::Function && uses.len() == 1 {
        let (file, line) = &uses[0];
        if file == &entity.file && *line >= entity.line && *line <= entity.end {
            return false;
        }
    }
    true
}
fn unnecessary_pass(lines: &[String], index: usize) -> bool {
    let current = level(&lines[index]);
    for previous in lines[..index].iter().rev().filter(|s| !empty(s)) {
        let indent = level(previous);
        if indent < current {
            break;
        }
        if indent == current {
            let trimmed = previous.trim();
            let block = trimmed.ends_with(':')
                || [
                    "func ", "if ", "elif ", "else:", "for ", "while ", "match ", "class ", "enum ",
                ]
                .iter()
                .any(|p| trimmed.starts_with(p));
            if !block {
                return true;
            }
        }
    }
    for next in lines[index + 1..].iter().filter(|s| !empty(s)) {
        let indent = level(next);
        if indent < current {
            break;
        }
        if indent == current {
            return true;
        }
    }
    false
}
fn annotated(file: &File, entity: &Entity) -> bool {
    let mut parts = Vec::new();
    let mut balance = 0_i64;
    for line in file.lines.iter().skip(entity.line - 1) {
        parts.push(line.trim());
        balance += line.matches('(').count() as i64 - line.matches(')').count() as i64;
        if balance <= 0 && line.contains(':') {
            break;
        }
    }
    pattern("returnAnnotationPattern").is_match(&parts.join(" "))
}

pub fn analyze(root: &Path, config: &Config) -> Result<Findings> {
    let paths = walk(root).map_err(|e| format!("failed to collect files: {e}"))?;
    let mut files = Vec::new();
    for path in &paths {
        let relative = path.strip_prefix(root)?;
        if !path.to_string_lossy().ends_with(".gd")
            || fs::symlink_metadata(path)?.is_dir()
            || build_path(relative)
            || ignored(path, &config.settings.ignore_patterns)
        {
            continue;
        }
        let content = fs::read_to_string(path).map_err(|e| {
            format!(
                "failed to parse files: failed to read {}: {e}",
                path.display()
            )
        })?;
        let lines = content.split('\n').map(str::to_owned).collect();
        let mut file = File {
            name: relative.to_string_lossy().into_owned(),
            path: path.clone(),
            content,
            lines,
            entities: Vec::new(),
        };
        detect(&mut file);
        files.push(file);
    }
    let mut referenced = BTreeSet::new();
    let mut autoloads = BTreeSet::new();
    if let Ok(project) = fs::read_to_string(root.join("project.godot")) {
        for m in pattern("autoloadPattern").captures_iter(&project) {
            autoloads.insert(Path::new(&m[1]).file_name().unwrap_or_default().to_owned());
            referenced.insert(root_join(root, &m[1]));
        }
    }
    for file in &files {
        for key in ["preloadPattern", "loadPattern", "extendsFilePattern"] {
            for m in pattern(key).captures_iter(&file.content) {
                if let Some(path) = resolve(root, &file.path, &m[1]) {
                    referenced.insert(path);
                }
            }
        }
    }
    for path in &paths {
        if path.to_string_lossy().ends_with(".tscn")
            && !build_path(path)
            && let Ok(content) = fs::read_to_string(path)
        {
            for key in ["extResourcePattern", "scriptPathPattern"] {
                for m in pattern(key).captures_iter(&content) {
                    if let Some(resolved) = resolve(root, path, &m[1]) {
                        referenced.insert(resolved);
                    }
                }
            }
        }
    }
    let usages = usages(&files);
    let mut results = Findings::new();
    for file in &files {
        for entity in &file.entities {
            let key = match entity.kind {
                Kind::Function => "unused_functions",
                Kind::Signal => "unused_signals",
                Kind::Constant => "unused_constants",
                Kind::Enum | Kind::EnumValue => "unused_enums",
                Kind::ClassName => "unused_class_names",
            };
            if config.enabled(key)
                && !ignored(&file.path, &config.settings.unused_ignore_patterns)
                && !used(entity, &usages)
            {
                let category = match entity.kind {
                    Kind::Function => 0,
                    Kind::Signal => 1,
                    Kind::Constant => 2,
                    Kind::Enum => 3,
                    Kind::EnumValue => 4,
                    Kind::ClassName => 5,
                };
                results.items[category].push(Finding::entity(entity));
            }
            if entity.kind != Kind::Function || entity.protected() {
                continue;
            }
            if config.warn("single_use_functions")
                && usages.get(&entity.name).is_some_and(|u| u.len() == 1)
            {
                results.items[15].push(Finding::entity(entity));
            }
            if config.warn("empty_functions")
                && file
                    .lines
                    .iter()
                    .take(entity.end)
                    .skip(entity.line)
                    .all(|line| matches!(remove_comments(line).trim(), "" | "pass"))
            {
                results.items[16].push(Finding::entity(entity));
            }
            if config.enabled("missing_return_types")
                && !entity.name.starts_with('_')
                && !annotated(file, entity)
            {
                results.items[14].push(Finding::location(
                    file,
                    entity.line,
                    format!(
                        "public function '{}' missing return type annotation",
                        entity.name
                    ),
                ));
            }
        }
        if config.enabled("unused_files")
            && !ignored(&file.path, &config.settings.unused_ignore_patterns)
            && !autoloads.contains(file.path.file_name().unwrap_or_default())
            && !referenced.contains(&file.path)
        {
            let class_used = file
                .entities
                .iter()
                .filter(|e| e.kind == Kind::ClassName)
                .any(|e| {
                    files
                        .iter()
                        .any(|other| other.path != file.path && other.content.contains(&e.name))
                });
            if !class_used {
                results.items[6].push(Finding::file(file.name.clone()));
            }
        }
        for (index, line) in file.lines.iter().enumerate() {
            let number = index + 1;
            let clean = remove_comments(line);
            let add = |results: &mut Findings, category: usize, text: String| {
                results.items[category].push(Finding::location(file, number, text))
            };
            if config.enabled("print_statements")
                && (pattern("PrintPattern").is_match(line)
                    || pattern("DebugLogPattern").is_match(line))
                && !pattern("IgnorePrintPattern").is_match(line)
                && (index == 0 || !pattern("IgnorePrintPattern").is_match(&file.lines[index - 1]))
            {
                add(&mut results, 7, line.trim().into());
            }
            if config.enabled("comments")
                && pattern("CommentPattern").is_match(line)
                && !pattern("IgnorePrintPattern").is_match(line)
                && !config
                    .settings
                    .allowed_comment_prefixes
                    .iter()
                    .any(|p| line.contains(p))
            {
                add(&mut results, 8, line.trim().into());
            }
            if config.enabled("pass_statements")
                && pattern("PassPattern").is_match(line)
                && unnecessary_pass(&file.lines, index)
            {
                add(&mut results, 9, line.trim().into());
            }
            if config.enabled("has_method") && pattern("HasMethodPattern").is_match(line) {
                add(&mut results, 12, line.trim().into());
            }
            if config.enabled("variant_usage")
                && !ignored(&file.path, &config.settings.variant_ignore_patterns)
                && pattern("VariantPattern").is_match(&clean)
            {
                add(&mut results, 13, clean.trim().into());
            }
            if config.enabled("indentation") && !line.trim().is_empty() {
                let indent = &line[..line.len() - line.trim_start_matches([' ', '\t']).len()];
                if indent.contains(' ') {
                    add(
                        &mut results,
                        11,
                        if indent.contains('\t') {
                            "Mixed tabs and spaces in indentation"
                        } else {
                            "Used spaces for indentation (GDScript requires tabs)"
                        }
                        .into(),
                    );
                }
            }
        }
    }
    if config.enabled("orphaned_uids") {
        for path in &paths {
            if path.to_string_lossy().ends_with(".gd.uid")
                && !ignored(path, &config.settings.ignore_patterns)
            {
                let gd = path.with_extension("");
                if fs::metadata(gd).is_err_and(|e| e.kind() == std::io::ErrorKind::NotFound) {
                    results.items[10].push(Finding::file(
                        path.strip_prefix(root)?.to_string_lossy().into_owned(),
                    ));
                }
            }
        }
    }
    Ok(results)
}

pub fn debt(root: &Path) -> Result<()> {
    let mut comments = Vec::new();
    for path in walk(root)? {
        if path.to_string_lossy().ends_with(".gd") && !fs::symlink_metadata(&path)?.is_dir() {
            let content = fs::read_to_string(&path)?;
            for (index, line) in content.split('\n').enumerate() {
                if let Some(m) = pattern("TechDebtPattern").captures(line) {
                    comments.push((
                        path.strip_prefix(root)?.to_string_lossy().into_owned(),
                        index + 1,
                        m[1].trim().to_owned(),
                    ));
                }
            }
        }
    }
    if comments.is_empty() {
        println!("No [TECH DEBT] comments found");
    } else {
        println!("Found {} [TECH DEBT] comments:\n", comments.len());
        for (file, line, text) in comments {
            println!("{file}:{line}: {text}");
        }
    }
    Ok(())
}
