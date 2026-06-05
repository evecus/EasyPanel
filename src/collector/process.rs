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

pub fn get_processes(sort_by: &str, sort_dir: &str, limit: usize) -> Result<Vec<ProcessInfo>> {
    let mut sys = System::new_with_specifics(
        RefreshKind::new().with_processes(ProcessRefreshKind::everything()),
    );
    sys.refresh_processes();

    let total_mem = sys.total_memory() as f32;

    let mut infos: Vec<ProcessInfo> = sys.processes().values().map(|p| {
        let mem_rss = p.memory();
        let mem_percent = if total_mem > 0.0 { (mem_rss as f32 / total_mem) * 100.0 } else { 0.0 };
        let cmdline = p.cmd().join(" ");
        ProcessInfo {
            pid: p.pid().as_u32(),
            name: p.name().to_string(),
            username: p.user_id().map(|u| u.to_string()).unwrap_or_default(),
            cpu_percent: p.cpu_usage(),
            mem_percent,
            mem_rss,
            status: format!("{:?}", p.status()),
            cmdline,
        }
    }).collect();

    let asc = sort_dir == "asc";
    match sort_by {
        "mem" => infos.sort_by(|a, b| {
            if asc { a.mem_percent.partial_cmp(&b.mem_percent).unwrap() }
            else { b.mem_percent.partial_cmp(&a.mem_percent).unwrap() }
        }),
        _ => infos.sort_by(|a, b| {
            if asc { a.cpu_percent.partial_cmp(&b.cpu_percent).unwrap() }
            else { b.cpu_percent.partial_cmp(&a.cpu_percent).unwrap() }
        }),
    }

    if limit > 0 && infos.len() > limit { infos.truncate(limit); }
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
