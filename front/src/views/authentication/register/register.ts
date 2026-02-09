import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterLink } from '@angular/router';
import { ReactiveFormsModule, FormBuilder, Validators, AbstractControl, ValidationErrors } from '@angular/forms';
import { AuthService } from '../../../core/api/auth.service';

@Component({
  selector: 'app-register',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, RouterLink],
  templateUrl: './register.html',
  styleUrl: './register.css',
})
export class Register {
  private authService = inject(AuthService);
  private router = inject(Router);
  private fb = inject(FormBuilder);

  readonly registerMutation = this.authService.register(() => {
    this.router.navigate(['/']);
  });

  form = this.fb.group({
    prenom: ['', [Validators.required, Validators.maxLength(50)]],
    nom: ['', [Validators.required, Validators.maxLength(50)]],
    email: ['', [Validators.required, Validators.email]],
    password: ['', [Validators.required, Validators.minLength(6)]],
    confirmPassword: ['', [Validators.required]]
  }, { validators: this.passwordMatchValidator });

  constructor() {
  }

  passwordMatchValidator(control: AbstractControl): ValidationErrors | null {
    const password = control.get('password');
    const confirmPassword = control.get('confirmPassword');

    if (password && confirmPassword && password.value !== confirmPassword.value) {
      confirmPassword.setErrors({ passwordMismatch: true });
      return { passwordMismatch: true };
    }
    return null;
  }

  submit() {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const { prenom, nom, email, password } = this.form.value;
    this.registerMutation.execute({
      prenom: prenom!,
      nom: nom!,
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
    if (field.errors['maxlength']) return `Maximum ${field.errors['maxlength'].requiredLength} caractères`;
    if (field.errors['passwordMismatch']) return 'Les mots de passe ne correspondent pas';
    return 'Valeur invalide';
  }
}
