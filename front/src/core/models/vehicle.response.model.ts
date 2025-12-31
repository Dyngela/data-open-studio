/**
 * Vehicle model matching backend API
 */
// type VehicleResponse struct {
//   ID              uint   `gorm:"primaryKey"`
//   Vin             string `gorm:"uniqueIndex;size:17;column:vin"` // Numéro d'identification du véhicule
//   Marque          string `gorm:"not null;size:100"`              // Marque (ex: Toyota, Ford)
//   Modele          string `gorm:"not null;size:100"`              // Nom du modèle
//   Annee           int    `gorm:"not null"`                       // Année de fabrication
//   Finition        string `gorm:"size:100"`                       // Niveau de finition
//   Couleur         string `gorm:"size:50"`
//   Kilometrage     int    `gorm:"default:0"`                       // Kilométrage
//   TypeCarburant   string `gorm:"size:50;column:type_carburant"`   // Essence, Diesel, Électrique, Hybride
//   Transmission    string `gorm:"size:50"`                         // Manuelle, Automatique
//   TypeCarrosserie string `gorm:"size:50;column:type_carrosserie"` // Berline, SUV, Camion, etc.
//   NombrePortes    int    `gorm:"default:4;column:nombre_portes"`
//   NombrePlaces    int    `gorm:"default:5;column:nombre_places"`
//   Moteur          string `gorm:"size:100"`  // Détails du moteur (ex: 2.0L Turbo)
//   Puissance       int    `gorm:"default:0"` // Chevaux
//
//   PrixAchat float64 `gorm:"type:decimal(10,2);default:0;column:prix_achat"`
//   PrixVente float64 `gorm:"type:decimal(10,2);default:0;column:prix_vente"`
//
//   Statut     models.StatutVehicule `gorm:"type:varchar(50);default:'en_stock'"`
//   Disponible bool                  `gorm:"default:true"`
//
//   // Localisation
//   Emplacement string `gorm:"size:200"`
//
//   // Informations supplémentaires
//   Description string `gorm:"type:text"`
//
//   // Horodatage
//   CreatedAt time.Time `gorm:"autoCreateTime;column:created_at"`
//   UpdatedAt time.Time `gorm:"autoUpdateTime;column:updated_at"`
// }

export interface Vehicule {
  id: number;
  vin: string;
  marque: string;
}

export enum VehicleStatus {
  IN_STOCK = 'en_stock',
  RESERVED = 'reserve',
  SOLD = 'vendu',
  MAINTENANCE = 'maintenance'
}


