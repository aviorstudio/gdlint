use crate::model::*;
use std::{collections::BTreeMap, fs, path::Path};

#[derive(Clone)]
struct Task {
    start: usize,
    end: usize,
    kind: usize,
}
fn enum_end(lines: &[String], start: usize) -> Result<usize> {
    let mut balance = 0_i64;
    let mut found = false;
    for (index, line) in lines.iter().enumerate().skip(start) {
        let clean = remove_comments(line);
        for ch in clean.chars() {
            if ch == '{' {
                balance += 1;
                found = true;
            }
            if ch == '}' {
                balance -= 1;
            }
        }
        if found && balance == 0 {
            return Ok(index + 1);
        }
    }
    Err("cannot safely remove enum: unbalanced braces".into())
}
fn needs_pass(lines: &[String], start: usize, end: usize) -> bool {
    let Some(previous) = lines[..start].iter().rev().find(|s| !empty(s)) else {
        return false;
    };
    if !previous.trim().ends_with(':') {
        return false;
    }
    let base = level(previous);
    for next in lines[end..].iter().filter(|s| !empty(s)) {
        let indent = level(next);
        if indent <= base {
            break;
        }
        if indent == base + 1 {
            return false;
        }
    }
    true
}
fn transform(lines: &mut Vec<String>, task: &Task) {
    let start = task.start;
    let end = task.end;
    let mut replacement = Vec::new();
    match task.kind {
        0 if needs_pass(lines, start, end) => {
            replacement.push(format!("{}pass", indent(&lines[start])))
        }
        7 => {
            let prefix = indent(&lines[start]);
            let previous_block = start > 0 && lines[start - 1].trim().ends_with(':');
            let last_statement = lines
                .get(start + 1)
                .is_none_or(|next| indent(next).len() <= prefix.len() || next.trim().is_empty());
            if previous_block && last_statement {
                replacement.push(format!("{prefix}pass"));
            }
        }
        8 if !pattern("CommentPattern").is_match(lines[start].trim()) => {
            let clean = remove_comments(&lines[start]);
            if !clean.trim().is_empty() {
                replacement.push(clean);
            }
        }
        _ => {}
    }
    lines.splice(start..end, replacement);
}
fn safe_path(root: &Path, name: &str) -> Result<std::path::PathBuf> {
    let path = root.join(name);
    if path
        .strip_prefix(root)?
        .components()
        .any(|c| !matches!(c, std::path::Component::Normal(_)))
    {
        return Err("refusing a fix outside the project".into());
    }
    let mut current = root.to_path_buf();
    for component in path.strip_prefix(root)?.components() {
        current.push(component);
        if fs::symlink_metadata(&current)?.file_type().is_symlink() {
            return Err(format!("refusing to modify symlink: {}", current.display()).into());
        }
    }
    Ok(path)
}
pub fn remove(root: &Path, results: &Findings) -> Result<()> {
    let mut files: BTreeMap<String, Vec<Task>> = BTreeMap::new();
    for category in [0, 1, 2, 3, 5, 7, 8, 9] {
        for finding in &results.items[category] {
            let start = finding.line - 1;
            let end = if category == 0 {
                finding.entity.as_ref().expect("function finding").end
            } else {
                start + 1
            };
            files.entry(finding.file.clone()).or_default().push(Task {
                start,
                end,
                kind: category,
            });
        }
    }
    // Preflight every path and edit before any write. Coordinates always refer
    // to the original file; contained findings must not shift outer ranges.
    let mut planned = Vec::new();
    for (name, mut tasks) in files {
        let path = safe_path(root, &name)?;
        let source = fs::read_to_string(&path)?;
        let mut lines: Vec<_> = source.split('\n').map(str::to_owned).collect();
        for task in &mut tasks {
            if task.start >= lines.len() || task.end > lines.len() || task.end <= task.start {
                return Err(format!("invalid removal range in {name}").into());
            }
            if task.kind == 3 {
                task.end = enum_end(&lines, task.start)?;
            }
            if task.kind == 0 {
                for index in (0..task.start).rev() {
                    let line = &lines[index];
                    if pattern("RPCPattern").is_match(line)
                        || pattern("ExportPattern").is_match(line)
                    {
                        task.start = index;
                    } else if !empty(line) {
                        break;
                    }
                }
            }
        }
        tasks.sort_by_key(|task| (task.start, std::cmp::Reverse(task.end), task.kind));
        let mut selected: Vec<Task> = Vec::new();
        for task in tasks {
            if let Some(previous) = selected.last()
                && task.start < previous.end
            {
                if task.end <= previous.end {
                    continue;
                }
                return Err(format!("overlapping removal ranges in {name}").into());
            }
            selected.push(task);
        }
        for task in selected.iter().rev() {
            transform(&mut lines, task);
        }
        let mut empty_count = 0;
        lines.retain(|line| {
            if line.trim().is_empty() {
                empty_count += 1;
                empty_count <= 2
            } else {
                empty_count = 0;
                true
            }
        });
        while lines.last().is_some_and(|line| line.trim().is_empty()) {
            lines.pop();
        }
        planned.push((path, lines.join("\n")));
    }
    let mut deletions = Vec::new();
    for category in [6, 10] {
        for finding in &results.items[category] {
            deletions.push((safe_path(root, &finding.file)?, category, &finding.file));
        }
    }
    for (path, content) in planned {
        fs::write(path, content)?;
    }
    for (path, category, name) in deletions {
        println!(
            "Removing {}: {name}",
            if category == 6 {
                "unused file"
            } else {
                "orphaned UID"
            }
        );
        if fs::symlink_metadata(&path)?.is_dir() {
            fs::remove_dir(path)?;
        } else {
            fs::remove_file(path)?;
        }
    }
    Ok(())
}
