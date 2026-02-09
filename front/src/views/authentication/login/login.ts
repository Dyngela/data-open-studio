import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterLink, ActivatedRoute } from '@angular/router';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { AuthService } from '../../../core/api/auth.service';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, RouterLink],
  templateUrl: './login.html',
  styleUrl: './login.css',
})
export class Login {
  private authService = inject(AuthService);
  private router = inject(Router);
  private route = inject(ActivatedRoute);
  private fb = inject(FormBuilder);

  private returnUrl: string = '/';

  readonly loginMutation = this.authService.login(() => {
    this.router.navigateByUrl(decodeURIComponent(this.returnUrl));
  });

  form = this.fb.group({
    email: ['', [Validators.required, Validators.email]],
    password: ['', [Validators.required, Validators.minLength(6)]]
  });

  constructor() {
    this.route.queryParams.subscribe(params => {
      this.returnUrl = params['returnUrl'] || '/';
    });
  }

  submit() {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const { email, password } = this.form.value;
    this.loginMutation.execute({
      email: email!,
      password: password!
    });
  }

  isFieldInvalid(fieldName: string): boolean {
    const field = this.form.get(fieldName);
    return !!(field && field.invalid && field.touched);
  }

  getFieldError(fieldName: string): string {
    const field = this.form.get(fieldName);
    if (!field?.errors) return '';
    if (field.errors['required']) return 'Ce champ est requis';
    if (field.errors['email']) return 'Email invalide';
    if (field.errors['minlength']) return `Minimum ${field.errors['minlength'].requiredLength} caractères`;
    return 'Valeur invalide';
  }
}
