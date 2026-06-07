use anyhow::Result;
use serde::Serialize;
use std::process::Command;

#[derive(Debug, Clone, Serialize, Default)]
pub struct SystemdService {
    pub unit: String,
    pub load: String,
    pub active: String,
    pub sub: String,
    pub description: String,
    pub pid: String,
    pub memory_bytes: u64,
    pub memory: String,
    pub cpu_ns: u64,
    pub cpu_time: String,
    pub unit_file_state: String,
    pub main_pid: String,
    pub exec_start: String,
    pub fragment_path: String,
    pub started_at: String,
    pub tasks: String,
}

pub fn get_services() -> Result<Vec<SystemdService>> {
    let out = Command::new("systemctl")
        .args([
            "list-units",
            "--type=service",
            "--all",
            "--no-pager",
            "--plain",
            "--no-legend",
        ])
        .output()?;

    let text = String::from_utf8_lossy(&out.stdout);
    let mut services = vec![];

    for line in text.lines() {
        let fields: Vec<&str> = line.split_whitespace().collect();
        if fields.len() < 4 {
            continue;
        }
        let mut svc = SystemdService {
            unit: fields[0].to_string(),
            load: fields[1].to_string(),
            active: fields[2].to_string(),
            sub: fields[3].to_string(),
            description: if fields.len() > 4 {
                fields[4..].join(" ")
            } else {
                String::new()
            },
            ..Default::default()
        };

        if svc.active == "active" {
            enrich_service(&mut svc);
        } else {
            get_unit_file_state(&mut svc);
        }

        services.push(svc);
    }

    Ok(services)
}

fn enrich_service(svc: &mut SystemdService) {
    let props = [
        "MainPID",
        "MemoryCurrent",
        "CPUUsageNSec",
        "UnitFileState",
        "ExecStart",
        "FragmentPath",
        "TasksCurrent",
        "ActiveEnterTimestamp",
    ];
    let props_args: Vec<String> = props.iter().map(|p| format!("--property={}", p)).collect();

    let out = Command::new("systemctl")
        .arg("show")
        .arg(&svc.unit)
        .arg("--no-pager")
        .args(&props_args)
        .output();

    let Ok(out) = out else { return };
    let text = String::from_utf8_lossy(&out.stdout);

    let kv = parse_kv(&text);

    svc.main_pid = kv.get("MainPID").cloned().unwrap_or_default();
    svc.unit_file_state = kv.get("UnitFileState").cloned().unwrap_or_default();
    svc.fragment_path = kv.get("FragmentPath").cloned().unwrap_or_default();
    svc.tasks = kv.get("TasksCurrent").cloned().unwrap_or_default();

    if let Some(es) = kv.get("ExecStart") {
        if let Some(idx) = es.find("path=") {
            let rest = &es[idx + 5..];
            let end = rest
                .find(|c: char| c == ' ' || c == ';')
                .unwrap_or(rest.len());
            svc.exec_start = rest[..end].to_string();
        }
    }

    if let Some(mem_str) = kv.get("MemoryCurrent") {
        if let Ok(v) = mem_str.parse::<u64>() {
            svc.memory_bytes = v;
            svc.memory = format_bytes(v);
        }
    }

    if let Some(cpu_str) = kv.get("CPUUsageNSec") {
        if let Ok(v) = cpu_str.parse::<u64>() {
            svc.cpu_ns = v;
            svc.cpu_time = format_duration_ns(v);
        }
    }

    if let Some(ts) = kv.get("ActiveEnterTimestamp") {
        svc.started_at = ts.clone();
    }
}

fn get_unit_file_state(svc: &mut SystemdService) {
    let out = Command::new("systemctl")
        .args([
            "show",
            &svc.unit,
            "--no-pager",
            "--property=UnitFileState",
            "--property=FragmentPath",
            "--property=ExecStart",
        ])
        .output();
    if let Ok(out) = out {
        let text = String::from_utf8_lossy(&out.stdout);
        let kv = parse_kv(&text);
        svc.unit_file_state = kv.get("UnitFileState").cloned().unwrap_or_default();
        svc.fragment_path = kv.get("FragmentPath").cloned().unwrap_or_default();
        if let Some(es) = kv.get("ExecStart") {
            if let Some(idx) = es.find("path=") {
                let rest = &es[idx + 5..];
                let end = rest
                    .find(|c: char| c == ' ' || c == ';')
                    .unwrap_or(rest.len());
                svc.exec_start = rest[..end].to_string();
            }
        }
    }
}

fn parse_kv(text: &str) -> std::collections::HashMap<String, String> {
    let mut map = std::collections::HashMap::new();
    for line in text.lines() {
        if let Some(idx) = line.find('=') {
            let key = line[..idx].to_string();
            let val = line[idx + 1..].to_string();
            map.insert(key, val);
        }
    }
    map
}

fn format_bytes(bytes: u64) -> String {
    if bytes >= 1024 * 1024 * 1024 {
        format!("{:.1}G", bytes as f64 / (1024.0 * 1024.0 * 1024.0))
    } else if bytes >= 1024 * 1024 {
        format!("{:.1}M", bytes as f64 / (1024.0 * 1024.0))
    } else if bytes >= 1024 {
        format!("{:.1}K", bytes as f64 / 1024.0)
    } else {
        format!("{}B", bytes)
    }
}

fn format_duration_ns(ns: u64) -> String {
    let secs = ns / 1_000_000_000;
    let h = secs / 3600;
    let m = (secs % 3600) / 60;
    let s = secs % 60;
    if h > 0 {
        format!("{}h{}m{}s", h, m, s)
    } else if m > 0 {
        format!("{}m{}s", m, s)
    } else {
        format!("{}s", s)
    }
}

pub fn service_action(unit: &str, action: &str) -> Result<()> {
    let status = Command::new("systemctl").args([action, unit]).status()?;
    if status.success() {
        Ok(())
    } else {
        Err(anyhow::anyhow!("systemctl {} {} failed", action, unit))
    }
}

pub fn get_service_logs(unit: &str, lines: usize) -> Result<String> {
    let out = Command::new("journalctl")
        .args([
            "-u",
            unit,
            "-n",
            &lines.to_string(),
            "--no-pager",
            "--output=short-iso",
        ])
        .output()?;
    Ok(String::from_utf8_lossy(&out.stdout).to_string())
}

pub fn sort_services(services: &mut Vec<SystemdService>, sort_by: &str, sort_dir: &str) {
    let asc = sort_dir == "asc";
    match sort_by {
        "memory" => services.sort_by(|a, b| {
            if asc {
                a.memory_bytes.cmp(&b.memory_bytes)
            } else {
                b.memory_bytes.cmp(&a.memory_bytes)
            }
        }),
        "unit" => services.sort_by(|a, b| {
            if asc {
                a.unit.cmp(&b.unit)
            } else {
                b.unit.cmp(&a.unit)
            }
        }),
        _ => {}
    }
}
