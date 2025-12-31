import {Component, input, output, signal} from '@angular/core';
import {FormControl, ReactiveFormsModule} from "@angular/forms";
import {AutoComplete} from "primeng/autocomplete";
import {NgClass} from "@angular/common";
import {FloatLabel} from "primeng/floatlabel";

@Component({
  selector: 'kui-autocomplete',
  imports: [
    AutoComplete,
    ReactiveFormsModule,
    NgClass,
    FloatLabel,
  ],
  templateUrl: './kui-autocomplete.html',
  styleUrl: './kui-autocomplete.css'
})
export class KuiAutocomplete {
  // Inputs
  label = input<string>('Rechercher');
  control = input.required<FormControl>();
  suggestions = input.required<any[]>();
  disabled = input<boolean>(false);
  hint = input<string>('');

  isFocused = signal<boolean>(false);

  focus = output<void>();
  blur = output<void>();
  completeMethod = output<any>();

  onFocus(event: Event): void {
    this.isFocused.set(true);
    this.focus.emit();
  }

  onBlur(event: Event): void {
    this.isFocused.set(false);
    this.blur.emit();
  }

  onComplete(event: any): void {
    this.completeMethod.emit(event);
  }
}
