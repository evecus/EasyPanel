use serde::Serialize;
use std::process::Command;
use std::time::Duration;
use sysinfo::{CpuExt, CpuRefreshKind, DiskExt, NetworkExt, RefreshKind, System, SystemExt};

#[derive(Debug, Clone, Serialize, Default)]
pub struct SystemInfo {
    pub hostname: String,
    pub os: String,
    pub platform: String,
    pub platform_version: String,
    pub kernel_version: String,
    pub arch: String,
    pub uptime: u64,
    pub uptime_str: String,
    pub boot_time: u64,
    pub cpu_model: String,
    pub cpu_cores: u32,
    pub cpu_threads: u32,
    pub local_ipv4: String,
}

#[derive(Debug, Clone, Serialize, Default)]
pub struct CpuStats {
    pub usage_percent: f32,
    pub per_core_usage: Vec<f32>,
    pub load_avg_1: f64,
    pub load_avg_5: f64,
    pub load_avg_15: f64,
    pub frequency_mhz: u64,
}

#[derive(Debug, Clone, Serialize, Default)]
pub struct MemoryStats {
    pub total: u64,
    pub used: u64,
    pub free: u64,
    pub available: u64,
    pub used_percent: f64,
    pub swap_total: u64,
    pub swap_used: u64,
    pub swap_free: u64,
    pub swap_percent: f64,
    pub cached: u64,
    pub buffers: u64,
}

#[derive(Debug, Clone, Serialize, Default)]
pub struct DiskPartition {
    pub device: String,
    pub mountpoint: String,
    pub fstype: String,
    pub total: u64,
    pub used: u64,
    pub free: u64,
    pub used_percent: f64,
    pub read_bytes: u64,
    pub write_bytes: u64,
}

#[derive(Debug, Clone, Serialize, Default)]
pub struct DiskStats {
    pub partitions: Vec<DiskPartition>,
}

#[derive(Debug, Clone, Serialize, Default)]
pub struct NetworkInterface {
    pub name: String,
    pub bytes_sent: u64,
    pub bytes_recv: u64,
    pub packets_sent: u64,
    pub packets_recv: u64,
    pub speed_up: u64,
    pub speed_down: u64,
    pub addrs: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Default)]
pub struct NetworkStats {
    pub interfaces: Vec<NetworkInterface>,
    pub total_sent: u64,
    pub total_recv: u64,
    pub connections: usize,
}

#[derive(Debug, Clone, Serialize, Default)]
pub struct Temperature {
    pub sensor: String,
    pub temperature: f32,
}

#[derive(Debug, Clone, Serialize, Default)]
pub struct MetricsSnapshot {
    pub timestamp: i64,
    pub system: SystemInfo,
    pub cpu: CpuStats,
    pub memory: MemoryStats,
    pub disk: DiskStats,
    pub network: NetworkStats,
    pub temperatures: Vec<Temperature>,
}

fn format_uptime(secs: u64) -> String {
    let d = secs / 86400;
    let h = (secs % 86400) / 3600;
    let m = (secs % 3600) / 60;
    if d > 0 {
        format!("{}d {}h {}m", d, h, m)
    } else {
        format!("{}h {}m", h, m)
    }
}

pub fn get_system_info() -> SystemInfo {
    let mut sys =
        System::new_with_specifics(RefreshKind::new().with_cpu(CpuRefreshKind::everything()));
    std::thread::sleep(Duration::from_millis(100));
    sys.refresh_cpu();

    let uptime = sys.uptime();
    let boot_time = sys.boot_time();
    let cpus = sys.cpus();
    let cpu_model = cpus
        .first()
        .map(|c| c.brand().to_string())
        .unwrap_or_default();
    let cpu_threads = cpus.len() as u32;
    let cpu_cores = sys.physical_core_count().unwrap_or(cpu_threads as usize) as u32;

    let local_ipv4 = local_ip_simple().unwrap_or_default();

    SystemInfo {
        hostname: sys.host_name().unwrap_or_default(),
        os: sys.name().unwrap_or_default(),
        platform: sys.name().unwrap_or_default(),
        platform_version: sys.os_version().unwrap_or_default(),
        kernel_version: sys.kernel_version().unwrap_or_default(),
        arch: std::env::consts::ARCH.to_string(),
        uptime,
        uptime_str: format_uptime(uptime),
        boot_time,
        cpu_model,
        cpu_cores,
        cpu_threads,
        local_ipv4,
    }
}

fn local_ip_simple() -> Option<String> {
    use std::net::UdpSocket;
    let socket = UdpSocket::bind("0.0.0.0:0").ok()?;
    socket.connect("8.8.8.8:80").ok()?;
    Some(socket.local_addr().ok()?.ip().to_string())
}

pub fn get_cpu_stats() -> CpuStats {
    let mut sys =
        System::new_with_specifics(RefreshKind::new().with_cpu(CpuRefreshKind::everything()));
    std::thread::sleep(Duration::from_millis(300));
    sys.refresh_cpu();

    let cpus = sys.cpus();
    let per_core: Vec<f32> = cpus.iter().map(|c| c.cpu_usage()).collect();
    let usage_percent = if per_core.is_empty() {
        0.0
    } else {
        per_core.iter().sum::<f32>() / per_core.len() as f32
    };
    let frequency_mhz = cpus.first().map(|c| c.frequency()).unwrap_or(0);
    let (load1, load5, load15) = read_loadavg();

    CpuStats {
        usage_percent,
        per_core_usage: per_core,
        load_avg_1: load1,
        load_avg_5: load5,
        load_avg_15: load15,
        frequency_mhz,
    }
}

fn read_loadavg() -> (f64, f64, f64) {
    if let Ok(content) = std::fs::read_to_string("/proc/loadavg") {
        let parts: Vec<&str> = content.split_whitespace().collect();
        if parts.len() >= 3 {
            return (
                parts[0].parse().unwrap_or(0.0),
                parts[1].parse().unwrap_or(0.0),
                parts[2].parse().unwrap_or(0.0),
            );
        }
    }
    (0.0, 0.0, 0.0)
}

pub fn get_memory_stats() -> MemoryStats {
    let mut sys = System::new_all();
    sys.refresh_memory();

    let total = sys.total_memory();
    let used = sys.used_memory();
    let available = sys.available_memory();
    let free = total.saturating_sub(used);
    let used_percent = if total > 0 {
        (used as f64 / total as f64) * 100.0
    } else {
        0.0
    };
    let swap_total = sys.total_swap();
    let swap_used = sys.used_swap();
    let swap_free = swap_total.saturating_sub(swap_used);
    let swap_percent = if swap_total > 0 {
        (swap_used as f64 / swap_total as f64) * 100.0
    } else {
        0.0
    };
    let (cached, buffers) = read_meminfo_cached_buffers();

    MemoryStats {
        total,
        used,
        free,
        available,
        used_percent,
        swap_total,
        swap_used,
        swap_free,
        swap_percent,
        cached,
        buffers,
    }
}

fn read_meminfo_cached_buffers() -> (u64, u64) {
    let mut cached = 0u64;
    let mut buffers = 0u64;
    if let Ok(content) = std::fs::read_to_string("/proc/meminfo") {
        for line in content.lines() {
            if line.starts_with("Cached:") {
                if let Some(v) = line
                    .split_whitespace()
                    .nth(1)
                    .and_then(|s| s.parse::<u64>().ok())
                {
                    cached = v * 1024;
                }
            } else if line.starts_with("Buffers:") {
                if let Some(v) = line
                    .split_whitespace()
                    .nth(1)
                    .and_then(|s| s.parse::<u64>().ok())
                {
                    buffers = v * 1024;
                }
            }
        }
    }
    (cached, buffers)
}

pub fn get_disk_stats() -> DiskStats {
    let mut sys = System::new_all();
    sys.refresh_disks_list();
    sys.refresh_disks();

    // Build a map of device short-name -> (read_bytes, write_bytes) from /proc/diskstats
    let io_map = read_diskstats();

    let mut seen = std::collections::HashSet::new();
    let mut partitions = vec![];

    for disk in sys.disks() {
        let mount = disk.mount_point().to_string_lossy().to_string();
        if !mount.starts_with('/') || !seen.insert(mount.clone()) {
            continue;
        }
        let total = disk.total_space();
        if total == 0 {
            continue;
        }
        let free = disk.available_space();
        let used = total.saturating_sub(free);
        let used_percent = (used as f64 / total as f64) * 100.0;

        let device_full = disk.name().to_string_lossy().to_string();
        let dev_short = device_full
            .rsplit('/')
            .next()
            .unwrap_or(&device_full)
            .to_string();
        let (read_bytes, write_bytes) = io_map
            .get(&dev_short)
            .copied()
            .unwrap_or((0, 0));

        partitions.push(DiskPartition {
            device: device_full,
            mountpoint: mount,
            fstype: disk.file_system().iter().map(|&b| b as char).collect(),
            total,
            used,
            free,
            used_percent,
            read_bytes,
            write_bytes,
        });
    }

    DiskStats { partitions }
}

/// Read /proc/diskstats and return map of device_name -> (read_bytes, write_bytes)
fn read_diskstats() -> std::collections::HashMap<String, (u64, u64)> {
    let mut map = std::collections::HashMap::new();
    let Ok(content) = std::fs::read_to_string("/proc/diskstats") else {
        return map;
    };
    for line in content.lines() {
        let fields: Vec<&str> = line.split_whitespace().collect();
        // /proc/diskstats columns: major minor name reads_comp reads_merged sectors_read ...
        // sectors_read at index 5, sectors_written at index 9; each sector = 512 bytes
        if fields.len() < 10 {
            continue;
        }
        let name = fields[2].to_string();
        let read_bytes = fields[5].parse::<u64>().unwrap_or(0) * 512;
        let write_bytes = fields[9].parse::<u64>().unwrap_or(0) * 512;
        map.insert(name, (read_bytes, write_bytes));
    }
    map
}

pub fn get_network_stats() -> NetworkStats {
    let mut sys = System::new_all();
    sys.refresh_networks_list();
    sys.refresh_networks();

    // Build address map from /proc/net/if_inet6 and /proc/net/fib_trie is complex;
    // use a simpler approach via `ip addr` parsing or /proc/net/arp
    let addr_map = read_interface_addrs();

    let mut interfaces = vec![];
    let mut total_sent = 0u64;
    let mut total_recv = 0u64;

    for (name, data) in sys.networks() {
        if name == "lo" || name.starts_with("veth") || name.starts_with("br-") {
            continue;
        }
        let bs = data.total_transmitted();
        let br = data.total_received();
        total_sent += bs;
        total_recv += br;
        let addrs = addr_map.get(name).cloned().unwrap_or_default();
        interfaces.push(NetworkInterface {
            name: name.clone(),
            bytes_sent: bs,
            bytes_recv: br,
            packets_sent: data.total_packets_transmitted(),
            packets_recv: data.total_packets_received(),
            speed_up: data.transmitted(),
            speed_down: data.received(),
            addrs,
        });
    }

    NetworkStats {
        interfaces,
        total_sent,
        total_recv,
        connections: count_tcp_connections(),
    }
}

/// Parse interface IPv4 addresses from /proc/net/fib_trie or fall back to `ip -o addr`
fn read_interface_addrs() -> std::collections::HashMap<String, Vec<String>> {
    let mut map: std::collections::HashMap<String, Vec<String>> = std::collections::HashMap::new();
    // Use `ip -o addr show` which is widely available
    if let Ok(out) = Command::new("ip").args(["-o", "addr", "show"]).output() {
        for line in String::from_utf8_lossy(&out.stdout).lines() {
            // Format: <idx>: <iface>    inet <addr>/<prefix> ...
            let parts: Vec<&str> = line.split_whitespace().collect();
            if parts.len() < 4 {
                continue;
            }
            let iface = parts[1].trim_end_matches(':').to_string();
            if parts[2] == "inet" || parts[2] == "inet6" {
                map.entry(iface).or_default().push(parts[3].to_string());
            }
        }
    }
    map
}

fn count_tcp_connections() -> usize {
    // Count established TCP connections from /proc/net/tcp and /proc/net/tcp6
    let mut count = 0usize;
    for path in &["/proc/net/tcp", "/proc/net/tcp6"] {
        if let Ok(content) = std::fs::read_to_string(path) {
            for line in content.lines().skip(1) {
                let fields: Vec<&str> = line.split_whitespace().collect();
                // Field 3 is the state; "01" = ESTABLISHED
                if fields.get(3) == Some(&"01") {
                    count += 1;
                }
            }
        }
    }
    count
}

pub fn get_temperatures() -> Vec<Temperature> {
    let mut temps = vec![];
    for i in 0..10 {
        let Ok(data) =
            std::fs::read_to_string(format!("/sys/class/thermal/thermal_zone{}/temp", i))
        else {
            break;
        };
        let millideg: f32 = data.trim().parse().unwrap_or(0.0);
        if millideg == 0.0 {
            continue;
        }
        let sensor = std::fs::read_to_string(format!("/sys/class/thermal/thermal_zone{}/type", i))
            .unwrap_or_default()
            .trim()
            .to_string();
        let sensor = if sensor.is_empty() {
            format!("zone{}", i)
        } else {
            sensor
        };
        temps.push(Temperature {
            sensor,
            temperature: millideg / 1000.0,
        });
    }
    temps
}

pub fn collect_all() -> MetricsSnapshot {
    MetricsSnapshot {
        timestamp: chrono::Utc::now().timestamp(),
        system: get_system_info(),
        cpu: get_cpu_stats(),
        memory: get_memory_stats(),
        disk: get_disk_stats(),
        network: get_network_stats(),
        temperatures: get_temperatures(),
    }
}
