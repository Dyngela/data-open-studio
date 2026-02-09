import {inject, Injectable, signal} from '@angular/core';
import { Observable, tap, of } from 'rxjs';
import {BaseApiService} from '../services/base-api.service';
import {AuthResponse, LoginDto, RegisterDto, User} from './auth.type';
import {CookieService} from 'ngx-cookie-service';
import {ApiMutation, ApiResult} from '../services/base-api.type';
import {jwtDecode} from 'jwt-decode';

/**
 * Authentication service
 * Handles login, register, logout, and token management
 */
@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private api = inject(BaseApiService);
  private cookieService = inject(CookieService);
  private currentUserSignal = signal<User | null>(null);
  public currentUser = this.currentUserSignal.asReadonly();
  private authenticatedSignal = signal(false);
  public isAuthenticated = this.authenticatedSignal.asReadonly();

  /**
   * Register new user
   */
  register(onSuccess?: () => void): ApiMutation<AuthResponse, RegisterDto> {
    return this.api.post<AuthResponse, RegisterDto>(
      '/auth/register',
      (response) => {
        this.setSession(response);
        onSuccess?.();
      },
    );
  }

  /**
   * Login user
   */
  login(onSuccess?: () => void): ApiMutation<AuthResponse, LoginDto> {
    return this.api.post(
      '/auth/login',
      (response) => {
        this.setSession(response);
        onSuccess?.();
      },
      undefined,
      { showSuccessMessage: true, successMessage: "OK" }
    );
  }

  /**
   * Logout user (client-side only, no backend endpoint)
   */
  logout(): void {
    this.clearSession();
  }

  /**
   * Refresh access token using refresh token
   */
  refreshToken(): Observable<AuthResponse> {
    const refreshToken = this.getRefreshToken();

    if (!refreshToken) {
      throw new Error('No refresh token available');
    }

    return this.api.request$<AuthResponse>('POST', '/auth/refresh', { refreshToken }).pipe(
      tap(response => this.setSession(response))
    );
  }


  /**
   * Set session data
   */
  private setSession(authResponse: AuthResponse): void {
    this.cookieService.set('access_token', authResponse.token, undefined, '/');
    this.cookieService.set('refresh_token', authResponse.refreshToken, undefined, '/');
    this.currentUserSignal.set(authResponse.user);
    this.authenticatedSignal.set(true);
  }

  /**
   * Clear session data
   */
  private clearSession(): void {
    this.cookieService.delete('access_token', '/');
    this.cookieService.delete('refresh_token', '/');
    this.currentUserSignal.set(null);
    this.authenticatedSignal.set(false);
  }

  /**
   * Get access token
   */
  getAccessToken(): string | null {
    return this.cookieService.get('access_token') ?? null;
  }

  /**
   * Get refresh token
   */
  getRefreshToken(): string | null {
    return this.cookieService.get('refresh_token') ?? null ;

  }

  /**
   * Initialize auth state from storage
   */
  initializeAuth(): void {
    const token = this.getAccessToken();
    if (!token) return;

    try {
      const decoded = jwtDecode<User>(token);

      this.currentUserSignal.set({
        id: decoded.id,
        email: decoded.email,
        prenom: decoded.prenom,
        nom: decoded.nom,
        role: decoded.role,
      });
      this.authenticatedSignal.set(true);

    } catch (err) {
      console.error('Invalid token', err);
      this.logout();
    }
  }
}
