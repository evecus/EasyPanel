use anyhow::Result;
use serde::Serialize;
use sysinfo::{PidExt, ProcessExt, ProcessRefreshKind, RefreshKind, System, SystemExt};

#[derive(Debug, Clone, Serialize)]
pub struct ProcessInfo {
    pub pid: u32,
    pub name: String,
    pub username: String,
    pub cpu_percent: f32,
    pub mem_percent: f32,
    pub mem_rss: u64,
    pub status: String,
    pub cmdline: String,
}

/// 读取 /etc/passwd，构建 uid_str -> username 映射
fn build_uid_map() -> std::collections::HashMap<String, String> {
    let mut map = std::collections::HashMap::new();
    if let Ok(content) = std::fs::read_to_string("/etc/passwd") {
        for line in content.lines() {
            let parts: Vec<&str> = line.split(':').collect();
            // passwd 格式: username:x:uid:gid:...
            if parts.len() >= 3 {
                map.insert(parts[2].to_string(), parts[0].to_string());
            }
        }
    }
    map
}

pub fn get_processes(sort_by: &str, sort_dir: &str, limit: usize) -> Result<Vec<ProcessInfo>> {
    // sysinfo 需要两次采样才能计算 CPU 使用率：
    // 第一次 refresh 建立基线，等 200ms 后第二次 refresh 才能得到非零 cpu_usage()。
    // Go 版的 gopsutil CPUPercent() 内部自动完成了这个时间差计算。
    let mut sys = System::new_with_specifics(
        RefreshKind::new().with_processes(ProcessRefreshKind::everything()),
    );
    sys.refresh_processes();
    std::thread::sleep(std::time::Duration::from_millis(200));
    sys.refresh_processes();
    sys.refresh_memory();

    let total_mem = sys.total_memory() as f32;

    // 预先构建 UID -> 用户名映射，避免每个进程反复读 /etc/passwd
    let uid_map = build_uid_map();

    let mut infos: Vec<ProcessInfo> = sys
        .processes()
        .values()
        .map(|p| {
            let mem_rss = p.memory();
            let mem_percent = if total_mem > 0.0 {
                (mem_rss as f32 / total_mem) * 100.0
            } else {
                0.0
            };
            let cmdline = p.cmd().join(" ");
            // 将 UID 数字解析为用户名，与 Go 版 process.Username() 行为一致
            let username = p
                .user_id()
                .map(|uid| {
                    let uid_str = uid.to_string();
                    uid_map.get(&uid_str).cloned().unwrap_or(uid_str)
                })
                .unwrap_or_default();
            ProcessInfo {
                pid: p.pid().as_u32(),
                name: p.name().to_string(),
                username,
                cpu_percent: p.cpu_usage(),
                mem_percent,
                mem_rss,
                status: format!("{:?}", p.status()),
                cmdline,
            }
        })
        .collect();

    let asc = sort_dir == "asc";
    match sort_by {
        "mem" => infos.sort_by(|a, b| {
            if asc {
                a.mem_percent.partial_cmp(&b.mem_percent).unwrap()
            } else {
                b.mem_percent.partial_cmp(&a.mem_percent).unwrap()
            }
        }),
        _ => infos.sort_by(|a, b| {
            if asc {
                a.cpu_percent.partial_cmp(&b.cpu_percent).unwrap()
            } else {
                b.cpu_percent.partial_cmp(&a.cpu_percent).unwrap()
            }
        }),
    }

    if limit > 0 && infos.len() > limit {
        infos.truncate(limit);
    }
    Ok(infos)
}

pub fn kill_process(pid: u32) -> Result<()> {
    use sysinfo::{Pid, PidExt, ProcessExt};
    let mut sys = System::new_with_specifics(
        RefreshKind::new().with_processes(ProcessRefreshKind::everything()),
    );
    sys.refresh_processes();
    let spid = Pid::from_u32(pid);
    if let Some(proc) = sys.process(spid) {
        proc.kill();
        Ok(())
    } else {
        Err(anyhow::anyhow!("process {} not found", pid))
    }
}
