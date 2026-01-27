export interface User {
  id: number;
  email: string;
  prenom: string;
  nom: string;
  role: UserRole;
}

export enum UserRole {
  ADMIN = 'admin',
  USER = 'user'
}

export interface RegisterDto {
  email: string;
  prenom: string;
  nom: string;
  password: string;
}

export interface LoginDto {
  email: string;
  password: string;
}

export interface AuthResponse {
  user: User;
  token: string;
  refreshToken: string;
}

