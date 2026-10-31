use crate::model::{Findings, TITLES};
const RESET: &str = "\x1b[0m";
const RED: &str = "\x1b[31m";
const GREEN: &str = "\x1b[32m";
const YELLOW: &str = "\x1b[33m";
const CYAN: &str = "\x1b[36m";
const GRAY: &str = "\x1b[90m";
const BOLD: &str = "\x1b[1m";

pub fn show(results: &mut Findings, warnings: bool) -> bool {
    let errors = results.errors();
    let warning_count = results.warnings();
    if errors == 0 && warning_count == 0 {
        println!("{GREEN}✅ No issues found - codebase is clean!{RESET}");
        return false;
    }
    for (category, findings) in results.items.iter_mut().enumerate() {
        if category == 0 && errors > 0 {
            println!("{BOLD}{RED}=== ERRORS (will be removed with -fix) ==={RESET}\n");
        }
        if category == 15 && warnings && warning_count > 0 {
            println!("{BOLD}{YELLOW}=== WARNINGS ==={RESET}\n");
        }
        if findings.is_empty() || (category >= 15 && !warnings) {
            continue;
        }
        let color = if category >= 15 { YELLOW } else { RED };
        let title = TITLES[category];
        if category == 12 {
            println!(
                "{BOLD}{RED}❌ has_method() calls (ALWAYS ERRORS):{RESET} {}",
                findings.len()
            );
        } else {
            println!("{color}{title}:{RESET} {}", findings.len());
        }
        findings.sort_by(|a, b| (&a.file, a.line).cmp(&(&b.file, b.line)));
        for finding in findings {
            let color = if category == 12 { RED } else { GRAY };
            if finding.line == 0 {
                println!("  {color}• {}{RESET}", finding.file);
            } else {
                let text = if finding.entity.is_none() && category != 12 && finding.text.len() > 60
                {
                    let end = finding
                        .text
                        .char_indices()
                        .map(|(i, _)| i)
                        .take_while(|i| *i <= 60)
                        .last()
                        .unwrap_or(0);
                    format!("{}...", &finding.text[..end])
                } else {
                    finding.text.clone()
                };
                println!(
                    "  {color}• {}:{}: {text}{RESET}",
                    finding.file, finding.line
                );
            }
        }
        println!();
    }
    println!("{BOLD}{CYAN}=== SUMMARY ==={RESET}");
    if errors > 0 {
        println!("{RED}❌ {errors} errors found{RESET}");
        println!("   Run with {BOLD}-fix{RESET} to remove all errors");
    }
    if warning_count > 0 {
        let hint = if warnings {
            String::new()
        } else {
            format!(" (run with {BOLD}-warn{RESET} to see them)")
        };
        println!("{YELLOW}⚠️  {warning_count} warnings found{RESET}{hint}");
    }
    errors > 0
}
