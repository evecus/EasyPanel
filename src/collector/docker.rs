use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::process::Command;
use std::time::Duration;

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct Container {
    pub id: String,
    pub name: String,
    pub image: String,
    pub status: String,
    pub state: String,
    pub ports: String,
    pub created: String,
    pub cpu_percent: f64,
    pub mem_percent: f64,
    pub mem_used: u64,
    pub mem_limit: u64,
}

#[derive(Debug, Clone, Serialize, Default)]
pub struct InspectResult {
    pub image: String,
    pub status: String,
    pub created: String,
    pub restart_policy: String,
    pub env: Vec<String>,
    pub mounts: Vec<String>,
    pub networks: Vec<String>,
    pub ports: String,
    pub cmd: Vec<String>,
    pub compose_file: String,
}

pub fn get_containers() -> Result<Vec<Container>> {
    let output = run_cmd_timeout(
        "docker",
        &[
            "ps",
            "-a",
            "--format",
            r#"{"id":"{{.ID}}","name":"{{.Names}}","image":"{{.Image}}","status":"{{.Status}}","state":"{{.State}}","ports":"{{.Ports}}","created":"{{.CreatedAt}}"}"#,
        ],
        10,
    )?;

    let mut containers: Vec<Container> = output
        .lines()
        .filter(|l| !l.is_empty())
        .filter_map(|l| serde_json::from_str(l).ok())
        .collect();

    // Enrich with stats
    if let Ok(stats_out) = run_cmd_timeout(
        "docker",
        &[
            "stats",
            "--no-stream",
            "--format",
            r"{{.ID}}\t{{.CPUPerc}}\t{{.MemPerc}}\t{{.MemUsage}}",
        ],
        8,
    ) {
        let mut stats_map = std::collections::HashMap::new();
        for line in stats_out.lines() {
            let parts: Vec<&str> = line.split('\t').collect();
            if parts.len() < 4 {
                continue;
            }
            let id = parts[0].to_string();
            let cpu = parts[1].trim_end_matches('%').parse::<f64>().unwrap_or(0.0);
            let mem_pct = parts[2].trim_end_matches('%').parse::<f64>().unwrap_or(0.0);
            let mem_parts: Vec<&str> = parts[3].split(" / ").collect();
            let mem_used = if mem_parts.len() == 2 {
                parse_mem_str(mem_parts[0])
            } else {
                0
            };
            let mem_limit = if mem_parts.len() == 2 {
                parse_mem_str(mem_parts[1])
            } else {
                0
            };
            stats_map.insert(id, (cpu, mem_pct, mem_used, mem_limit));
        }
        for c in containers.iter_mut() {
            if let Some(&(cpu, mem_pct, mem_used, mem_limit)) = stats_map.get(&c.id) {
                c.cpu_percent = cpu;
                c.mem_percent = mem_pct;
                c.mem_used = mem_used;
                c.mem_limit = mem_limit;
            }
        }
    }

    Ok(containers)
}

fn parse_mem_str(s: &str) -> u64 {
    let s = s.trim();
    let units: &[(&str, u64)] = &[
        ("GiB", 1024 * 1024 * 1024),
        ("MiB", 1024 * 1024),
        ("KiB", 1024),
        ("GB", 1_000_000_000),
        ("MB", 1_000_000),
        ("KB", 1_000),
        ("B", 1),
    ];
    for (suffix, mult) in units {
        if s.ends_with(suffix) {
            let num: f64 = s[..s.len() - suffix.len()].parse().unwrap_or(0.0);
            return (num * *mult as f64) as u64;
        }
    }
    s.parse().unwrap_or(0)
}

pub fn container_action(id: &str, action: &str) -> Result<()> {
    let status = Command::new("docker").args([action, id]).status()?;
    if status.success() {
        Ok(())
    } else {
        Err(anyhow::anyhow!("docker {} failed", action))
    }
}

pub fn get_container_logs(id: &str, lines: usize) -> Result<String> {
    let out = run_cmd_timeout("docker", &["logs", "--tail", &lines.to_string(), id], 10)?;
    Ok(out)
}

pub fn inspect_container(id: &str) -> Result<InspectResult> {
    let out = run_cmd_timeout("docker", &["inspect", id], 10)?;
    let raw: Vec<serde_json::Value> = serde_json::from_str(&out)?;
    let c = raw
        .into_iter()
        .next()
        .ok_or_else(|| anyhow::anyhow!("empty inspect"))?;

    let mut result = InspectResult::default();

    if let Some(cfg) = c.get("Config").and_then(|v| v.as_object()) {
        result.image = cfg
            .get("Image")
            .and_then(|v| v.as_str())
            .unwrap_or_default()
            .to_string();
        if let Some(env) = cfg.get("Env").and_then(|v| v.as_array()) {
            result.env = env
                .iter()
                .filter_map(|v| v.as_str().map(|s| s.to_string()))
                .collect();
        }
        if let Some(cmd) = cfg.get("Cmd").and_then(|v| v.as_array()) {
            result.cmd = cmd
                .iter()
                .filter_map(|v| v.as_str().map(|s| s.to_string()))
                .collect();
        }
    }

    if let Some(state) = c.get("State").and_then(|v| v.as_object()) {
        result.status = state
            .get("Status")
            .and_then(|v| v.as_str())
            .unwrap_or_default()
            .to_string();
    }

    result.created = c
        .get("Created")
        .and_then(|v| v.as_str())
        .unwrap_or_default()
        .to_string();

    if let Some(host_cfg) = c.get("HostConfig").and_then(|v| v.as_object()) {
        if let Some(rp) = host_cfg.get("RestartPolicy").and_then(|v| v.as_object()) {
            result.restart_policy = rp
                .get("Name")
                .and_then(|v| v.as_str())
                .unwrap_or_default()
                .to_string();
        }
    }

    if let Some(mounts) = c.get("Mounts").and_then(|v| v.as_array()) {
        result.mounts = mounts
            .iter()
            .filter_map(|m| {
                let src = m.get("Source").and_then(|v| v.as_str()).unwrap_or_default();
                let dst = m
                    .get("Destination")
                    .and_then(|v| v.as_str())
                    .unwrap_or_default();
                if src.is_empty() && dst.is_empty() {
                    None
                } else {
                    Some(format!("{}:{}", src, dst))
                }
            })
            .collect();
    }

    if let Some(nets) = c
        .get("NetworkSettings")
        .and_then(|v| v.get("Networks"))
        .and_then(|v| v.as_object())
    {
        result.networks = nets.keys().cloned().collect();
    }

    // Find compose file from labels
    if let Some(cfg) = c.get("Config").and_then(|v| v.as_object()) {
        if let Some(labels) = cfg.get("Labels").and_then(|v| v.as_object()) {
            if let Some(wdir) = labels
                .get("com.docker.compose.project.working_dir")
                .and_then(|v| v.as_str())
            {
                result.compose_file = format!("{}/docker-compose.yml", wdir);
            }
        }
    }

    Ok(result)
}

pub fn read_compose_file(path: &str) -> Result<String> {
    Ok(std::fs::read_to_string(path)?)
}

pub fn write_and_apply_compose(path: &str, content: &str, _container_id: &str) -> Result<String> {
    std::fs::write(path, content)?;

    let dir = std::path::Path::new(path)
        .parent()
        .ok_or_else(|| anyhow::anyhow!("invalid path"))?;

    let out = Command::new("docker")
        .args(["compose", "up", "-d", "--build"])
        .current_dir(dir)
        .output()?;

    let log = format!(
        "{}\n{}",
        String::from_utf8_lossy(&out.stdout),
        String::from_utf8_lossy(&out.stderr)
    );

    if out.status.success() {
        Ok(log)
    } else {
        Err(anyhow::anyhow!("{}", log))
    }
}

pub fn pull_and_update_container(id: &str) -> Result<String> {
    // Get image name first
    let image_out = run_cmd_timeout(
        "docker",
        &["inspect", "--format", "{{.Config.Image}}", id],
        5,
    )?;
    let image = image_out.trim();

    let pull_out = Command::new("docker").args(["pull", image]).output()?;
    let pull_log = format!(
        "{}\n{}",
        String::from_utf8_lossy(&pull_out.stdout),
        String::from_utf8_lossy(&pull_out.stderr)
    );

    // Restart container
    let _ = Command::new("docker").args(["restart", id]).output();

    Ok(pull_log)
}

fn run_cmd_timeout(cmd: &str, args: &[&str], timeout_secs: u64) -> Result<String> {
    use std::process::Stdio;
    let mut child = Command::new(cmd)
        .args(args)
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()?;

    let deadline = std::time::Instant::now() + Duration::from_secs(timeout_secs);
    loop {
        match child.try_wait()? {
            Some(status) => {
                let out = child.wait_with_output()?;
                if status.success() {
                    return Ok(String::from_utf8_lossy(&out.stdout).to_string());
                } else {
                    let stderr = String::from_utf8_lossy(&out.stderr);
                    return Err(anyhow::anyhow!("command failed: {}", stderr));
                }
            }
            None => {
                if std::time::Instant::now() > deadline {
                    let _ = child.kill();
                    return Err(anyhow::anyhow!("command timed out"));
                }
                std::thread::sleep(Duration::from_millis(100));
            }
        }
    }
}
