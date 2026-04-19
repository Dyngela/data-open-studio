use crate::error::AppError;

pub fn hash(password: &str) -> Result<String, AppError> {
    bcrypt::hash(password, bcrypt::DEFAULT_COST)
        .map_err(|e| AppError::internal(format!("password hash: {e}")))
}

pub fn verify(password: &str, hash: &str) -> Result<bool, AppError> {
    bcrypt::verify(password, hash)
        .map_err(|e| AppError::internal(format!("password verify: {e}")))
}
