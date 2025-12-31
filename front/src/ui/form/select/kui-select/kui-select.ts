import {Component, input, output, signal} from '@angular/core';
import {FormControl, ReactiveFormsModule} from "@angular/forms";
import {Select} from "primeng/select";
import {NgClass} from "@angular/common";
import {FloatLabel} from "primeng/floatlabel";

@Component({
  selector: 'kui-select',
  imports: [
    Select,
    ReactiveFormsModule,
    NgClass,
    FloatLabel,
  ],
  templateUrl: './kui-select.html',
  styleUrl: './kui-select.css'
})
export class KuiSelect {
  // Inputs
  label = input<string>('Sélectionner');
  control = input.required<FormControl>();
  options = input.required<any[]>();
  optionLabel = input<string>('label');
  optionValue = input<string>('value');
  filter = input<boolean>(false);
  showClear = input<boolean>(false);
  disabled = input<boolean>(false);
  hint = input<string>('');

  isFocused = signal<boolean>(false);

  focus = output<void>();
  blur = output<void>();
  change = output<any>();

  onFocus(event: any): void {
    this.isFocused.set(true);
    this.focus.emit();
  }

  onBlur(event: any): void {
    this.isFocused.set(false);
    this.blur.emit();
  }

  onChange(event: any): void {
    this.change.emit(event.value);
  }
}
