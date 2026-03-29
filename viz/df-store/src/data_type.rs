use crate::store::Column;
use crate::time::{TimeUnit, TimeZone};

#[derive(Debug, Clone)]
pub enum DataValue {
    Boolean(bool),

    // --- Entiers non-signés ---
    UInt8(u8),
    UInt16(u16),
    UInt32(u32), // Pour les index de lignes ou IDs de taille moyenne
    UInt64(u64), // Pour les compteurs massifs ou les pointeurs mémoire

    // --- Entiers signés ---
    Int8(i8),
    Int16(i16),
    Int32(i32),
    Int64(i64), // Le standard pour les calculs de mesures

    // --- Nombres à virgule flottante ---
    Float32(f32),
    Float64(f64),

    String(String),
    Binary(Vec<u8>),

    /// Nombre de jours depuis l'époque Unix (01-01-1970)
    Date(i32),

    /// Temps écoulé depuis minuit (souvent en nanosecondes)
    Time(i64),

    /// Date et heure complète (Timestamp)
    /// Le i64 stocke la valeur, TimeUnit définit l'échelle (ms, ns, etc.)
    Datetime(i64, TimeUnit, Option<TimeZone>),

    /// Une durée relative (ex: 2h 30min) stockée en u64 pour pouvoir être négatif
    Duration(u64, TimeUnit),

    /// Collection récursive : contient un vecteur de données du même type
    List(Box<Vec<DataValue>>),

    /// Structure complexe : contient plusieurs colonnes avec leurs propres noms/types
    Struct(Vec<Column>),
    Null,
}

#[derive(Debug, PartialEq, Clone)]
pub enum DataType {
    Boolean,
    UInt8,
    UInt16,
    UInt32,
    UInt64,
    Int8,
    Int16,
    Int32,
    Int64,
    Float32,
    Float64,
    String,
    Binary,
    Date,
    Time,
    Datetime,
    Duration,
    List,
    Struct,
    Null,
}

