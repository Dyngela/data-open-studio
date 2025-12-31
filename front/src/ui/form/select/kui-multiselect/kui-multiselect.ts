import {Component, input, output, signal} from '@angular/core';
import {FormControl, ReactiveFormsModule} from "@angular/forms";
import {MultiSelect} from "primeng/multiselect";
import {NgClass} from "@angular/common";
import {FloatLabel} from "primeng/floatlabel";

@Component({
  selector: 'kui-multiselect',
  imports: [
    MultiSelect,
    ReactiveFormsModule,
    NgClass,
    FloatLabel,
  ],
  templateUrl: './kui-multiselect.html',
  styleUrl: './kui-multiselect.css'
})
export class KuiMultiselect {
  // Inputs
  label = input<string>('Sélections multiples');
  control = input.required<FormControl>();
  options = input.required<any[]>();
  optionLabel = input<string>('label');
  optionValue = input<string>('value');
  filter = input<boolean>(true);
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
