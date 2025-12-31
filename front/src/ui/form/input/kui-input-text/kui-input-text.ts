import {Component, forwardRef, Input, input, output, signal} from '@angular/core';
import {FormControl, NG_VALUE_ACCESSOR, ReactiveFormsModule} from "@angular/forms";
import {InputText} from "primeng/inputtext";
import {NgClass} from "@angular/common";
import {FloatLabel} from "primeng/floatlabel";

@Component({
  selector: 'kui-input-text',
  imports: [
    InputText,
    ReactiveFormsModule,
    NgClass,
    FloatLabel,
  ],

  templateUrl: './kui-input-text.html',
  styleUrl: './kui-input-text.scss'
})
export class KuiInputText {
  // Inputs
  label = input<string>('Texte');
  control = input.required<FormControl>();
  type = input<string>('text');
  disabled = input<boolean>(false);
  readonly = input<boolean>(false);
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
