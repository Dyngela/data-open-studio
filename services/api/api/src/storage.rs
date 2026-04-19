/// Persistent frame storage using the cedrus (.cedr) format from df-store.
use std::path::PathBuf;
use uuid::Uuid;

use df_store::cedrus::Cedrus;
use df_store::frame::Frame;

/// Returns the directory where frames for a workspace are persisted.
pub fn frame_store_dir(workspace_id: Uuid) -> PathBuf {
    let base = std::env::var("FRAME_STORE_DIR").unwrap_or_else(|_| "/tmp/frame-store".into());
    PathBuf::from(base).join(workspace_id.to_string())
}

/// Returns the .cedr file path for a given workspace + frame name.
pub fn frame_file_path(workspace_id: Uuid, frame_name: &str) -> PathBuf {
    frame_store_dir(workspace_id).join(format!("{}.cedr", frame_name))
}

/// Spawn a background task to persist a frame to disk.
pub fn spawn_persist(workspace_id: Uuid, frame: Frame) {
    tokio::spawn(async move {
        let dir = frame_store_dir(workspace_id);
        if let Err(e) = std::fs::create_dir_all(&dir) {
            tracing::warn!("storage: could not create dir {:?}: {}", dir, e);
            return;
        }
        let mut c = Cedrus::new();
        match c.write_to(&frame, &dir) {
            Ok(()) => tracing::debug!("storage: persisted frame '{}' -> {:?}", frame.name, dir),
            Err(e) => tracing::warn!("storage: failed to persist frame '{}': {}", frame.name, e),
        }
    });
}

/// Restore all persisted frames for a workspace from disk.
pub fn restore_all(workspace_id: Uuid) -> Vec<Frame> {
    let dir = frame_store_dir(workspace_id);
    if !dir.exists() {
        return Vec::new();
    }

    let entries = match std::fs::read_dir(&dir) {
        Ok(e)  => e,
        Err(e) => {
            tracing::warn!("storage: could not read dir {:?}: {}", dir, e);
            return Vec::new();
        }
    };

    let mut frames = Vec::new();
    for entry in entries.flatten() {
        let path = entry.path();
        if path.extension().and_then(|e| e.to_str()) == Some("cedr") {
            if let Some(stem) = path.file_stem().and_then(|s| s.to_str()) {
                match Cedrus::read_from(stem, &dir) {
                    Ok(frame) => {
                        tracing::debug!("storage: restored frame '{}' from {:?}", frame.name, path);
                        frames.push(frame);
                    }
                    Err(e) => {
                        tracing::warn!("storage: failed to restore {:?}: {}", path, e);
                    }
                }
            }
        }
    }

    frames
}
