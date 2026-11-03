mod analysis;
mod display;
mod fix;
mod model;
use model::{Config, DEFAULTS, Result};
use std::{
    fs::{self, OpenOptions},
    io::Write,
    path::PathBuf,
    time::Instant,
};

#[derive(Default)]
struct Options {
    fix: bool,
    json: bool,
    config: Option<PathBuf>,
    warn: bool,
    debt: bool,
    benchmark: bool,
    command: String,
}
fn usage() -> String {
    let binary = std::env::args().next().unwrap_or_else(|| "gdlint".into());
    format!(
        "Usage: {binary} [options]\n       {binary} init\n       {binary} version\n\nGDScript Linter - Analyzes and cleans GDScript code\n\nOptions:\n  -benchmark\n    Show performance timing\n  -debt\n    Show [TECH DEBT] comments\n  -fix\n    Remove all unused code and clean files\n  -warn\n    Show warnings in addition to errors\n\n  --json          Print one structured result without terminal control bytes\n  --config PATH   Override project-local gdlint.json\n\nCommands:\n  init           Create default gdlint.json in the current Godot project\n  version        Print version information"
    )
}
fn options() -> Result<Option<Options>> {
    let mut options = Options::default();
    let mut commands = Vec::new();
    let mut flags = true;
    let mut arguments = std::env::args().skip(1);
    while let Some(argument) = arguments.next() {
        if flags && argument == "--" {
            flags = false;
            continue;
        }
        if flags && argument.starts_with('-') && argument != "-" {
            let argument = argument.trim_start_matches('-');
            let (key, value) = argument.split_once('=').unwrap_or((argument, "true"));
            if key == "h" || key == "help" {
                if options.json {
                    println!(
                        "{}",
                        serde_json::json!({"v":1,"ok":true,"result":{"help":usage()}})
                    );
                } else {
                    print!("{}", usage());
                }
                return Ok(None);
            }
            if key == "config" {
                let selected = argument
                    .split_once('=')
                    .map(|(_, value)| value.to_owned())
                    .or_else(|| arguments.next())
                    .filter(|v| !v.is_empty())
                    .ok_or("--config needs a path")?;
                options.config = Some(PathBuf::from(selected));
                continue;
            }
            let target = match key {
                "fix" => &mut options.fix,
                "json" => &mut options.json,
                "warn" => &mut options.warn,
                "debt" => &mut options.debt,
                "benchmark" => &mut options.benchmark,
                _ => return Err(format!("flag provided but not defined: -{key}").into()),
            };
            *target = match value {
                "1" | "t" | "T" | "TRUE" | "true" | "True" => true,
                "0" | "f" | "F" | "FALSE" | "false" | "False" => false,
                _ => return Err(format!("invalid boolean value {value} for -{key}").into()),
            };
        } else {
            flags = false;
            commands.push(argument);
        }
    }
    if commands.len() > 1 {
        return Err("expected at most one command".into());
    }
    if let Some(command) = commands.pop() {
        if command != "init" && command != "version" {
            return Err(format!("unknown command: {command}").into());
        }
        options.command = command;
    }
    Ok(Some(options))
}
fn project() -> Result<PathBuf> {
    let root = std::env::current_dir()?;
    match fs::metadata(root.join("project.godot")) {
        Ok(metadata) if metadata.is_dir() => {
            return Err("project.godot is a directory, expected a file".into());
        }
        Ok(_) => {}
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
            return Err(
                "current directory must be a Godot project root containing project.godot".into(),
            );
        }
        Err(error) => return Err(error.into()),
    }
    Ok(root)
}
fn execute(options: Options) -> Result<bool> {
    if options.command == "version" {
        if options.json {
            println!(
                "{}",
                serde_json::json!({"v":1,"ok":true,"result":{"version":env!("GDLINT_VERSION"),"commit":env!("GDLINT_COMMIT"),"date":env!("GDLINT_DATE")}})
            );
        } else {
            println!(
                "gdlint {} ({}, {})",
                env!("GDLINT_VERSION"),
                env!("GDLINT_COMMIT"),
                env!("GDLINT_DATE")
            );
        }
        return Ok(false);
    }
    let root = project().map_err(|e| format!("Error validating project root: {e}"))?;
    if options.command == "init" {
        let path = options
            .config
            .clone()
            .unwrap_or_else(|| root.join("gdlint.json"));
        let mut file = OpenOptions::new()
            .write(true)
            .create_new(true)
            .open(&path)
            .map_err(|e| {
                if e.kind() == std::io::ErrorKind::AlreadyExists {
                    format!(
                        "Error creating config: config already exists: {}",
                        path.display()
                    )
                } else {
                    format!("Error creating config: {e}")
                }
            })?;
        file.write_all(DEFAULTS.as_bytes())?;
        if options.json {
            println!(
                "{}",
                serde_json::json!({"v":1,"ok":true,"result":{"path":path}})
            );
        } else {
            println!("Created default configuration file: {}", path.display());
        }
        return Ok(false);
    }
    let started = Instant::now();
    let config = Config::load(&root, options.config.as_deref())
        .map_err(|e| format!("Error loading config: {e}"))?;
    if options.json {
        if options.debt && !options.fix {
            let comments = analysis::debt_findings(&root)?;
            println!(
                "{}",
                serde_json::json!({"v":1,"ok":true,"result":{"comments":comments}})
            );
            return Ok(false);
        }
        if options.fix {
            for _ in 0..10 {
                let results = analysis::analyze(&root, &config)?;
                if results.errors() == 0 {
                    break;
                }
                fix::remove(&root, &results, true)?;
            }
        }
        let results = analysis::analyze(&root, &config)?;
        let mut findings = Vec::new();
        for (category, rows) in results.items.iter().enumerate() {
            if category >= 15 && !options.warn {
                continue;
            }
            let mut rows: Vec<_> = rows.iter().collect();
            rows.sort_by(|a, b| (&a.file, a.line).cmp(&(&b.file, b.line)));
            findings.extend(rows.into_iter().map(|row| serde_json::json!({"category":model::TITLES[category],"severity":if category<15{"error"}else{"warning"},"file":row.file,"line":row.line,"message":row.text})));
        }
        let mut result = serde_json::json!({"v":1,"ok":results.errors()==0,"result":{"errors":results.errors(),"warnings":results.warnings(),"findings":findings,"fixed":options.fix}});
        if options.benchmark {
            result["result"]["duration_ms"] = serde_json::json!(started.elapsed().as_millis());
        }
        println!("{result}");
        return Ok(results.errors() > 0);
    }
    if options.fix {
        let mut clean = false;
        for iteration in 1..=10 {
            println!("\n=== Fix iteration {iteration} ===");
            let results = analysis::analyze(&root, &config)
                .map_err(|e| format!("Error during analysis: {e}"))?;
            if results.errors() == 0 {
                println!("✅ No issues found - codebase is clean!");
                clean = true;
                break;
            }
            fix::remove(&root, &results, false)
                .map_err(|e| format!("Error during removal: {e}"))?;
        }
        if !clean {
            eprintln!("Warning: Reached maximum iterations (10) - there may still be issues");
            return Ok(true);
        }
    } else if options.debt {
        analysis::debt(&root).map_err(|e| format!("Error scanning for tech debt: {e}"))?;
    } else {
        let mut results =
            analysis::analyze(&root, &config).map_err(|e| format!("Error during analysis: {e}"))?;
        if display::show(&mut results, options.warn) {
            return Ok(true);
        }
    }
    if options.benchmark {
        println!("\nExecution time: {:.2}s", started.elapsed().as_secs_f64());
    }
    Ok(false)
}
fn machine_error(code: &str, message: &str) {
    println!(
        "{}",
        serde_json::json!({"v":1,"ok":false,"error":{"code":code,"message":message}})
    );
}
fn main() {
    let machine = std::env::args()
        .skip(1)
        .take_while(|arg| arg != "--")
        .any(|arg| matches!(arg.as_str(), "--json" | "-json" | "--json=true"));
    let options = match options() {
        Ok(Some(options)) => options,
        Ok(None) => return,
        Err(error) => {
            eprintln!("Error: {error}");
            if machine {
                machine_error("invalid_usage", &error.to_string());
            } else {
                eprint!("{}", usage());
            }
            std::process::exit(2);
        }
    };
    match execute(options) {
        Ok(false) => {}
        Ok(true) => std::process::exit(1),
        Err(error) => {
            eprintln!("{error}");
            if machine {
                machine_error("operation_failed", &error.to_string());
            }
            std::process::exit(1);
        }
    }
}
