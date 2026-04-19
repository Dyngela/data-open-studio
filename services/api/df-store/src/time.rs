#[derive(Debug, Clone, PartialEq)]
pub enum TimeUnit {
    Nanoseconds,
    Microseconds,
    Milliseconds,
}

/// Juste un wrapper autour d'un String pour le timezone
/// (ex: "Europe/Paris", "UTC")
pub type TimeZone = String;