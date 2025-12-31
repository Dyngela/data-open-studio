import { Injectable, signal } from '@angular/core';
import { Observable, tap, of } from 'rxjs';
import { BaseApiService } from './base-api.service';
import {User, LoginDto, AuthResponse, RegisterDto} from '../models';

/**
 * Authentication service
 * Handles login, register, logout, and token management
 */
@Injectable({
  providedIn: 'root'
})
export class AuthService extends BaseApiService {
  private currentUserSignal = signal<User | null>(null);
  public currentUser = this.currentUserSignal.asReadonly();

  /**
   * Register new user
   */
  register(userData: RegisterDto): Observable<AuthResponse> {
    return this.post<RegisterDto, AuthResponse>('auth/register', userData)
      .pipe(
        tap(response => {
          this.setSession(response);
        })
      );
  }

  /**
   * Login user
   */
  login(credentials: LoginDto): Observable<AuthResponse> {
    return this.post<LoginDto, AuthResponse>('auth/login', credentials)
      .pipe(
        tap(response => {
          this.setSession(response);
        })
      );
  }

  /**
   * Logout user (client-side only, no backend endpoint)
   */
  logout(): void {
    this.clearSession();
  }

  /**
   * Get current user profile
   */
  getCurrentUser(): Observable<User> {
    return this.get<User>('me')
      .pipe(
        tap(user => {
          this.currentUserSignal.set(user);
        })
      );
  }

  /**
   * Refresh access token using refresh token
   */
  refreshToken(): Observable<AuthResponse> {
    const refreshToken = this.getRefreshToken();
    if (!refreshToken) {
      throw new Error('No refresh token available');
    }

    return this.post<{ refreshToken: string }, AuthResponse>('auth/refresh', { refreshToken })
      .pipe(
        tap(response => {
          this.setSession(response);
        })
      );
  }

  /**
   * Set session data
   */
  private setSession(authResponse: AuthResponse): void {
    localStorage.setItem('access_token', authResponse.token);
    localStorage.setItem('refresh_token', authResponse.refreshToken);
    this.currentUserSignal.set(authResponse.user);
  }

  /**
   * Clear session data
   */
  private clearSession(): void {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    this.currentUserSignal.set(null);
  }

  /**
   * Get access token
   */
  getAccessToken(): string | null {
    return localStorage.getItem('access_token');
  }

  /**
   * Get refresh token
   */
  getRefreshToken(): string | null {
    return localStorage.getItem('refresh_token');
  }

  /**
   * Check if user is authenticated
   */
  isAuthenticated(): boolean {
    return !!this.getAccessToken();
  }

  /**
   * Initialize auth state from storage
   */
  initializeAuth(): void {
    if (this.isAuthenticated()) {
      this.getCurrentUser().subscribe();
    }
  }
}
