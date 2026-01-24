import {inject, Injectable, signal} from '@angular/core';
import { Observable, tap, of } from 'rxjs';
import {BaseApiService} from '../services/base-api.service';
import {AuthResponse, LoginDto, RegisterDto, User} from './auth.type';
import {CookieService} from 'ngx-cookie-service';
import {ApiMutation, ApiResult} from '../services/base-api.type';

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

  constructor() {
    this.initializeAuth()
  }

  /**
   * Register new user
   */
  register(): ApiMutation<AuthResponse> {
    return this.api.post<AuthResponse>(
      '/auth/register',
      (response) => this.setSession(response),
    );
  }

  /**
   * Login user
   */
  login(): ApiMutation<AuthResponse, LoginDto> {
    return this.api.post(
      '/auth/login',
      (response) => this.setSession(response),
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
   * Get current user profile
   */
  getCurrentUser(): ApiResult<User> {
    return this.api.get<User>('me')
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
  private async setSession(authResponse: AuthResponse): Promise<void> {
    this.cookieService.set('access_token', authResponse.token);
    this.cookieService.set('refresh_token', authResponse.refreshToken);
    this.currentUserSignal.set(authResponse.user);
  }

  /**
   * Clear session data
   */
  private clearSession(): void {
    this.cookieService.delete('access_token');
    this.cookieService.delete('refresh_token');
    this.currentUserSignal.set(null);
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
      this.getCurrentUser().refresh();
    }
  }
}
