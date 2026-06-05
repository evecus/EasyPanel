use crate::collector::{
    cache::{
        get_containers_from_cache, get_services_from_cache, invalidate_docker_cache,
        invalidate_systemd_cache,
    },
    docker::{
        container_action, get_container_logs, inspect_container, pull_and_update_container,
        read_compose_file, write_and_apply_compose,
    },
    process::{get_processes, kill_process},
    system::collect_all,
    systemd::{get_service_logs, service_action},
};
use axum::{extract::Path, http::StatusCode, Json};
use serde::Deserialize;
use std::collections::HashMap;

pub async fn get_metrics_all() -> Json<serde_json::Value> {
    let metrics = tokio::task::spawn_blocking(collect_all).await.unwrap();
    Json(serde_json::to_value(metrics).unwrap())
}

pub async fn get_processes_handler(
    axum::extract::Query(params): axum::extract::Query<HashMap<String, String>>,
) -> Json<serde_json::Value> {
    let sort = params
        .get("sort")
        .map(|s| s.clone())
        .unwrap_or_else(|| "cpu".into());
    let dir = params
        .get("dir")
        .map(|s| s.clone())
        .unwrap_or_else(|| "desc".into());
    let limit: usize = params
        .get("limit")
        .and_then(|v| v.parse().ok())
        .unwrap_or(100);
    match tokio::task::spawn_blocking(move || get_processes(&sort, &dir, limit))
        .await
        .unwrap()
    {
        Ok(procs) => Json(serde_json::to_value(procs).unwrap()),
        Err(e) => Json(serde_json::json!({"error": e.to_string()})),
    }
}

pub async fn kill_process_handler(
    Path(pid): Path<u32>,
) -> Result<Json<serde_json::Value>, (StatusCode, Json<serde_json::Value>)> {
    tokio::task::spawn_blocking(move || kill_process(pid))
        .await
        .unwrap()
        .map(|_| Json(serde_json::json!({"ok": true})))
        .map_err(|e| {
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({"error": e.to_string()})),
            )
        })
}

pub async fn get_containers_handler() -> Json<serde_json::Value> {
    match get_containers_from_cache() {
        Ok(c) => Json(serde_json::to_value(c).unwrap()),
        Err(_) => Json(serde_json::Value::Array(vec![])),
    }
}

pub async fn container_action_handler(
    Path((id, action)): Path<(String, String)>,
) -> Result<Json<serde_json::Value>, (StatusCode, Json<serde_json::Value>)> {
    if !matches!(action.as_str(), "start" | "stop" | "restart") {
        return Err((
            StatusCode::BAD_REQUEST,
            Json(serde_json::json!({"error":"invalid action"})),
        ));
    }
    tokio::task::spawn_blocking(move || container_action(&id, &action))
        .await
        .unwrap()
        .map_err(|e| {
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({"error": e.to_string()})),
            )
        })?;
    invalidate_docker_cache();
    Ok(Json(serde_json::json!({"ok": true})))
}

pub async fn get_container_logs_handler(
    Path(id): Path<String>,
    axum::extract::Query(params): axum::extract::Query<HashMap<String, String>>,
) -> Result<Json<serde_json::Value>, (StatusCode, Json<serde_json::Value>)> {
    let lines: usize = params
        .get("lines")
        .and_then(|v| v.parse().ok())
        .unwrap_or(200);
    tokio::task::spawn_blocking(move || get_container_logs(&id, lines))
        .await
        .unwrap()
        .map(|logs| Json(serde_json::json!({"logs": logs})))
        .map_err(|e| {
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({"error": e.to_string()})),
            )
        })
}

pub async fn inspect_container_handler(
    Path(id): Path<String>,
) -> Result<Json<serde_json::Value>, (StatusCode, Json<serde_json::Value>)> {
    tokio::task::spawn_blocking(move || inspect_container(&id))
        .await
        .unwrap()
        .map(|data| Json(serde_json::to_value(data).unwrap()))
        .map_err(|e| {
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({"error": e.to_string()})),
            )
        })
}

pub async fn get_compose_file_handler(
    axum::extract::Query(params): axum::extract::Query<HashMap<String, String>>,
) -> Result<Json<serde_json::Value>, (StatusCode, Json<serde_json::Value>)> {
    let path = params.get("path").cloned().ok_or_else(|| {
        (
            StatusCode::BAD_REQUEST,
            Json(serde_json::json!({"error":"path required"})),
        )
    })?;
    tokio::task::spawn_blocking(move || read_compose_file(&path))
        .await
        .unwrap()
        .map(|content| Json(serde_json::json!({"content": content})))
        .map_err(|e| {
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({"error": e.to_string()})),
            )
        })
}

#[derive(Deserialize)]
pub struct ApplyComposeReq {
    pub path: String,
    pub content: String,
    pub container_id: String,
}

pub async fn apply_compose_handler(
    Json(body): Json<ApplyComposeReq>,
) -> Result<Json<serde_json::Value>, (StatusCode, Json<serde_json::Value>)> {
    let (path, content, cid) = (
        body.path.clone(),
        body.content.clone(),
        body.container_id.clone(),
    );
    let log = tokio::task::spawn_blocking(move || write_and_apply_compose(&path, &content, &cid))
        .await
        .unwrap()
        .map_err(|e| {
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({"error": e.to_string()})),
            )
        })?;
    invalidate_docker_cache();
    Ok(Json(serde_json::json!({"message": "重建成功", "log": log})))
}

pub async fn pull_update_container_handler(
    Path(id): Path<String>,
) -> Result<Json<serde_json::Value>, (StatusCode, Json<serde_json::Value>)> {
    let log = tokio::task::spawn_blocking(move || pull_and_update_container(&id))
        .await
        .unwrap()
        .map_err(|e| {
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({"error": e.to_string()})),
            )
        })?;
    invalidate_docker_cache();
    Ok(Json(serde_json::json!({"log": log})))
}

pub async fn get_services_handler(
    axum::extract::Query(params): axum::extract::Query<HashMap<String, String>>,
) -> Json<serde_json::Value> {
    let sort = params.get("sort").cloned().unwrap_or_default();
    let dir = params.get("dir").cloned().unwrap_or_else(|| "desc".into());
    match get_services_from_cache(&sort, &dir) {
        Ok(svcs) => Json(serde_json::to_value(svcs).unwrap()),
        Err(_) => Json(serde_json::Value::Array(vec![])),
    }
}

pub async fn service_action_handler(
    Path((unit, action)): Path<(String, String)>,
) -> Result<Json<serde_json::Value>, (StatusCode, Json<serde_json::Value>)> {
    if !matches!(
        action.as_str(),
        "start" | "stop" | "restart" | "enable" | "disable"
    ) {
        return Err((
            StatusCode::BAD_REQUEST,
            Json(serde_json::json!({"error":"invalid action"})),
        ));
    }
    tokio::task::spawn_blocking(move || service_action(&unit, &action))
        .await
        .unwrap()
        .map_err(|e| {
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({"error": e.to_string()})),
            )
        })?;
    invalidate_systemd_cache();
    Ok(Json(serde_json::json!({"ok": true})))
}

pub async fn get_service_logs_handler(
    Path(unit): Path<String>,
    axum::extract::Query(params): axum::extract::Query<HashMap<String, String>>,
) -> Result<Json<serde_json::Value>, (StatusCode, Json<serde_json::Value>)> {
    let lines: usize = params
        .get("lines")
        .and_then(|v| v.parse().ok())
        .unwrap_or(200);
    tokio::task::spawn_blocking(move || get_service_logs(&unit, lines))
        .await
        .unwrap()
        .map(|logs| Json(serde_json::json!({"logs": logs})))
        .map_err(|e| {
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({"error": e.to_string()})),
            )
        })
}
