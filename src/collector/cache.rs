use once_cell::sync::Lazy;
use std::sync::{Arc, RwLock};
use std::time::{Duration, Instant};
use tokio::time;
use tracing::error;

use super::docker::{get_containers, Container};
use super::systemd::{get_services, sort_services, SystemdService};
use crate::config;

// ── Systemd cache ─────────────────────────────────────────────────

struct Cache<T: Clone> {
    data: Option<Vec<T>>,
    updated_at: Option<Instant>,
    ready: bool,
}

impl<T: Clone> Cache<T> {
    fn new() -> Self {
        Self { data: None, updated_at: None, ready: false }
    }
}

static SYSTEMD_CACHE: Lazy<Arc<RwLock<Cache<SystemdService>>>> =
    Lazy::new(|| Arc::new(RwLock::new(Cache::new())));

static DOCKER_CACHE: Lazy<Arc<RwLock<Cache<Container>>>> =
    Lazy::new(|| Arc::new(RwLock::new(Cache::new())));

pub fn get_services_from_cache(sort_by: &str, sort_dir: &str) -> Result<Vec<SystemdService>, String> {
    let ready = {
        let guard = SYSTEMD_CACHE.read().unwrap();
        guard.ready
    };

    let mut snapshot = if ready {
        let guard = SYSTEMD_CACHE.read().unwrap();
        guard.data.clone().unwrap_or_default()
    } else {
        // fetch synchronously on first miss
        match get_services() {
            Ok(data) => {
                let mut guard = SYSTEMD_CACHE.write().unwrap();
                guard.data = Some(data.clone());
                guard.updated_at = Some(Instant::now());
                guard.ready = true;
                data
            }
            Err(e) => return Err(e.to_string()),
        }
    };

    sort_services(&mut snapshot, sort_by, sort_dir);
    Ok(snapshot)
}

pub fn invalidate_systemd_cache() {
    let mut guard = SYSTEMD_CACHE.write().unwrap();
    guard.ready = false;
}

pub fn get_containers_from_cache() -> Result<Vec<Container>, String> {
    let ready = {
        let guard = DOCKER_CACHE.read().unwrap();
        guard.ready
    };

    if ready {
        let guard = DOCKER_CACHE.read().unwrap();
        Ok(guard.data.clone().unwrap_or_default())
    } else {
        match get_containers() {
            Ok(data) => {
                let mut guard = DOCKER_CACHE.write().unwrap();
                guard.data = Some(data.clone());
                guard.updated_at = Some(Instant::now());
                guard.ready = true;
                Ok(data)
            }
            Err(e) => Err(e.to_string()),
        }
    }
}

pub fn invalidate_docker_cache() {
    let mut guard = DOCKER_CACHE.write().unwrap();
    guard.ready = false;
}

// ── Background refresh ─────────────────────────────────────────────

pub fn start_background_cache(
    feature_systemd: impl Fn() -> bool + Send + 'static,
    feature_docker: impl Fn() -> bool + Send + 'static,
) {
    let systemd_cache = SYSTEMD_CACHE.clone();
    let docker_cache = DOCKER_CACHE.clone();

    tokio::spawn(async move {
        // Initial load
        if feature_systemd() {
            refresh_systemd_inner(&systemd_cache);
        }

        let interval_secs = config::cache_interval();
        let mut ticker = time::interval(Duration::from_secs(interval_secs));
        ticker.tick().await; // skip immediate first tick

        loop {
            ticker.tick().await;
            if !feature_systemd() {
                let mut guard = systemd_cache.write().unwrap();
                guard.data = None;
                guard.ready = false;
                continue;
            }
            refresh_systemd_inner(&systemd_cache);
        }
    });

    tokio::spawn(async move {
        if feature_docker() {
            refresh_docker_inner(&docker_cache);
        }

        let interval_secs = config::cache_interval();
        let mut ticker = time::interval(Duration::from_secs(interval_secs));
        ticker.tick().await;

        loop {
            ticker.tick().await;
            if !feature_docker() {
                let mut guard = docker_cache.write().unwrap();
                guard.data = None;
                guard.ready = false;
                continue;
            }
            refresh_docker_inner(&docker_cache);
        }
    });
}

fn refresh_systemd_inner(cache: &Arc<RwLock<Cache<SystemdService>>>) {
    match get_services() {
        Ok(data) => {
            let mut guard = cache.write().unwrap();
            guard.data = Some(data);
            guard.updated_at = Some(Instant::now());
            guard.ready = true;
        }
        Err(e) => error!("[cache] systemd refresh error: {}", e),
    }
}

fn refresh_docker_inner(cache: &Arc<RwLock<Cache<Container>>>) {
    match get_containers() {
        Ok(data) => {
            let mut guard = cache.write().unwrap();
            guard.data = Some(data);
            guard.updated_at = Some(Instant::now());
            guard.ready = true;
        }
        Err(e) => error!("[cache] docker refresh error: {}", e),
    }
}
