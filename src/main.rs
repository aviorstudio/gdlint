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
    warn: bool,
    debt: bool,
    benchmark: bool,
    command: String,
}
fn usage() {
    let binary = std::env::args().next().unwrap_or_else(|| "gdlint".into());
    eprintln!(
        "Usage: {binary} [options]\n       {binary} init\n       {binary} version\n\nGDScript Linter - Analyzes and cleans GDScript code\n\nOptions:\n  -benchmark\n    Show performance timing\n  -debt\n    Show [TECH DEBT] comments\n  -fix\n    Remove all unused code and clean files\n  -warn\n    Show warnings in addition to errors\n\nCommands:\n  init           Create default gdlint.json in the current Godot project\n  version        Print version information"
    );
}
fn options() -> Result<Option<Options>> {
    let mut options = Options::default();
    let mut commands = Vec::new();
    let mut flags = true;
    for argument in std::env::args().skip(1) {
        if flags && argument == "--" {
            flags = false;
            continue;
        }
        if flags && argument.starts_with('-') && argument != "-" {
            let argument = argument.trim_start_matches('-');
            let (key, value) = argument.split_once('=').unwrap_or((argument, "true"));
            if key == "h" || key == "help" {
                usage();
                return Ok(None);
            }
            let target = match key {
                "fix" => &mut options.fix,
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
        println!(
            "gdlint {} ({}, {})",
            env!("GDLINT_VERSION"),
            env!("GDLINT_COMMIT"),
            env!("GDLINT_DATE")
        );
        return Ok(false);
    }
    let root = project().map_err(|e| format!("Error validating project root: {e}"))?;
    if options.command == "init" {
        let path = root.join("gdlint.json");
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
        println!("Created default configuration file: gdlint.json");
        return Ok(false);
    }
    let started = Instant::now();
    let config = Config::load(&root).map_err(|e| format!("Error loading config: {e}"))?;
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
            fix::remove(&root, &results).map_err(|e| format!("Error during removal: {e}"))?;
        }
        if !clean {
            eprintln!("Warning: Reached maximum iterations (10) - there may still be issues");
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
fn main() {
    let options = match options() {
        Ok(Some(options)) => options,
        Ok(None) => return,
        Err(error) => {
            eprintln!("Error: {error}");
            usage();
            std::process::exit(1);
        }
    };
    match execute(options) {
        Ok(false) => {}
        Ok(true) => std::process::exit(1),
        Err(error) => {
            eprintln!("{error}");
            std::process::exit(1);
        }
    }
}
