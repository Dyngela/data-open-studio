use aes_gcm::{
    aead::{Aead, AeadCore, KeyInit, OsRng},
    Aes256Gcm, Nonce,
};
use base64::{engine::general_purpose::STANDARD as B64, Engine};

const PREFIX: &str = "enc:";

/// Encrypt a plaintext secret and return a `enc:<base64(nonce||ciphertext)>` string.
/// Empty input is returned as-is (no encryption needed for empty secrets).
pub fn encrypt_secret(key: &[u8; 32], plaintext: &str) -> Result<String, String> {
    if plaintext.is_empty() {
        return Ok(String::new());
    }
    let cipher = Aes256Gcm::new_from_slice(key).map_err(|e| e.to_string())?;
    let nonce = Aes256Gcm::generate_nonce(&mut OsRng);
    let ciphertext = cipher.encrypt(&nonce, plaintext.as_bytes())
        .map_err(|e| e.to_string())?;

    let mut combined = nonce.to_vec();
    combined.extend_from_slice(&ciphertext);
    Ok(format!("{PREFIX}{}", B64.encode(&combined)))
}

/// Decrypt an `enc:...` value produced by `encrypt_secret`.
/// Values without the `enc:` prefix are returned as-is for backwards compatibility
/// (existing plaintext credentials in the DB continue to work after deployment).
pub fn decrypt_secret(key: &[u8; 32], stored: &str) -> Result<String, String> {
    if stored.is_empty() {
        return Ok(String::new());
    }
    let encoded = match stored.strip_prefix(PREFIX) {
        Some(e) => e,
        // Legacy plaintext value — pass through until re-encrypted on next update
        None => return Ok(stored.to_string()),
    };
    let combined = B64.decode(encoded).map_err(|e| format!("base64 decode: {e}"))?;
    if combined.len() < 12 {
        return Err("encrypted value too short".to_string());
    }
    let (nonce_bytes, ciphertext) = combined.split_at(12);
    let cipher = Aes256Gcm::new_from_slice(key).map_err(|e| e.to_string())?;
    let nonce = Nonce::from_slice(nonce_bytes);
    let plaintext = cipher.decrypt(nonce, ciphertext)
        .map_err(|_| "decryption failed — wrong key or corrupted ciphertext".to_string())?;
    String::from_utf8(plaintext).map_err(|e| format!("utf-8: {e}"))
}

/// Sentinel string used in API responses in place of actual secret values.
pub const MASKED: &str = "***";
