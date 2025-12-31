import {Component, input, output, signal} from '@angular/core';
import {FormControl, ReactiveFormsModule} from "@angular/forms";
import {Password} from "primeng/password";
import {NgClass} from "@angular/common";
import {FloatLabel} from "primeng/floatlabel";

@Component({
  selector: 'kui-input-password',
  imports: [
    Password,
    ReactiveFormsModule,
    NgClass,
    FloatLabel,
  ],
  templateUrl: './kui-input-password.html',
  styleUrl: './kui-input-password.css'
})
export class KuiInputPassword {
  // Inputs
  label = input<string>('Mot de passe');
  control = input.required<FormControl>();
  disabled = input<boolean>(false);
  feedback = input<boolean>(true);
  toggleMask = input<boolean>(true);
  hint = input<string>('');

  isFocused = signal<boolean>(false);

  focus = output<void>();
  blur = output<void>();

  onFocus(event: Event): void {
    this.isFocused.set(true);
    this.focus.emit();
  }

  onBlur(event: Event): void {
    this.isFocused.set(false);
    this.blur.emit();
  }
}
