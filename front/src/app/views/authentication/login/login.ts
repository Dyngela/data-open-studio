import {Component, effect, inject} from '@angular/core';
import {AuthService} from '../../../../core/api/auth.service';
import {Router} from '@angular/router';
import {FormControl, FormGroup, Validators} from '@angular/forms';

@Component({
  selector: 'app-login',
  imports: [],
  templateUrl: './login.html',
  styleUrl: './login.css',
})
export class Login {
  private authService = inject(AuthService);
  private router = inject(Router);

  readonly login = this.authService.login();

  form = new FormGroup({
    email: new FormControl('', [Validators.required, Validators.email]),
    password: new FormControl('', Validators.required)
  });

  constructor() {
    effect(() => {
      if (this.login.success()) {
        console.log('Login successful, navigating to dashboard');
        // this.router.navigate(['/dashboard']);
      }
    });
  }

  submit() {
    this.login.execute({email: "wstestuser@example.com", password: "password"});

    if (this.form.valid) {
      // this.login.execute(this.form.value);
    }
  }
}

