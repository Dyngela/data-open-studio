import {VehicleStatus} from "./vehicle.response.model";

/**
 * DTO for creating a vehicle manually
 */
export interface CreateVehicleDto {
  vin?: string;
  make: string;
  model: string;
  year: number;
  trim?: string;
  color?: string;
  mileage?: number;
  fuelType?: string;
  transmission?: string;
  bodyType?: string;
  doors?: number;
  seats?: number;
  engine?: string;
  power?: number;
  purchasePrice?: number;
  salePrice?: number;
  location?: string;
  description?: string;
}

/**
 * DTO for updating a vehicle
 */
export interface UpdateVehicleDto {
  vin?: string;
  make?: string;
  model?: string;
  year?: number;
  trim?: string;
  color?: string;
  mileage?: number;
  fuelType?: string;
  transmission?: string;
  bodyType?: string;
  doors?: number;
  seats?: number;
  engine?: string;
  power?: number;
  purchasePrice?: number;
  salePrice?: number;
  status?: VehicleStatus;
  isAvailable?: boolean;
  location?: string;
  description?: string;
}

/**
 * DTO for creating a vehicle from DAT
 */
export interface CreateVehicleFromDATDto {
  vin: string;
}
