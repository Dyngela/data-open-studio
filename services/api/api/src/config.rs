use std::env;

#[derive(Clone, Debug)]
pub struct JwtConfig {
    pub secret:              String,
    pub expiry_minutes:      i64,
    pub refresh_expiry_days: i64,
}

#[derive(Clone, Debug)]
pub struct AppConfig {
    pub database_url:    String,
    pub port:            u16,
    pub jwt:             JwtConfig,
    pub nats_url:        Option<String>,
    pub ollama_url:      String,
    /// AES-256-GCM key for encrypting stored credentials.
    /// Set CREDENTIALS_KEY to 64 hex digits (32 bytes) in the environment.
    /// Defaults to all-zeros in development — set a real key for production.
    pub credentials_key: [u8; 32],
}

impl AppConfig {
    pub fn from_env() -> Self {
        let _ = dotenvy::dotenv();
        Self {
            database_url: env::var("DATABASE_URL")
                .unwrap_or_else(|_| "postgres://postgres:postgres@localhost:5433/data-open-studio".into()),
            port: env::var("API_PORT")
                .ok()
                .and_then(|s| s.trim_start_matches(':').parse().ok())
                .unwrap_or(3030),
            jwt: JwtConfig {
                secret: env::var("JWT_SECRET")
                    .expect("JWT_SECRET environment variable is required"),
                expiry_minutes: env::var("JWT_EXPIRATION_MINUTES")
                    .ok().and_then(|s| s.parse().ok()).unwrap_or(60),
                refresh_expiry_days: env::var("JWT_REFRESH_EXPIRATION_DAYS")
                    .ok().and_then(|s| s.parse().ok()).unwrap_or(30),
            },
            nats_url: env::var("NATS_URL").ok(),
            ollama_url: env::var("OLLAMA_HOST")
                .unwrap_or_else(|_| "http://localhost:11434".into()),
            credentials_key: load_credentials_key(),
        }
    }
}

fn load_credentials_key() -> [u8; 32] {
    match env::var("CREDENTIALS_KEY") {
        Ok(val) if val.len() == 64 => {
            let bytes: Result<Vec<u8>, _> = (0..32)
                .map(|i| u8::from_str_radix(&val[i * 2..i * 2 + 2], 16))
                .collect();
            bytes
                .expect("CREDENTIALS_KEY must be 64 valid hex digits")
                .try_into()
                .unwrap()
        }
        Ok(_) => panic!("CREDENTIALS_KEY must be exactly 64 hex digits (32 bytes)"),
        Err(_) => {
            tracing::warn!(
                "CREDENTIALS_KEY not set — stored credentials will not be encrypted. \
                 Set CREDENTIALS_KEY=<64 hex digits> for production."
            );
            [0u8; 32]
        }
    }
}
