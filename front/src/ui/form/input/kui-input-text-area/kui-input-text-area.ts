import {Component, input, output, signal} from '@angular/core';
import {FormControl, ReactiveFormsModule} from "@angular/forms";
import {Textarea} from "primeng/textarea";
import {NgClass} from "@angular/common";
import {FloatLabel} from "primeng/floatlabel";

@Component({
  selector: 'kui-input-text-area',
  imports: [
    Textarea,
    ReactiveFormsModule,
    NgClass,
    FloatLabel,
  ],
  templateUrl: './kui-input-text-area.html',
  styleUrl: './kui-input-text-area.scss'
})
export class KuiInputTextArea {
  // Inputs
  label = input<string>('Texte');
  control = input.required<FormControl>();
  rows = input<number>(3);
  autoResize = input<boolean>(false);
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
