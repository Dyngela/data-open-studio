import {Component, input, output, signal} from '@angular/core';
import {FormControl, ReactiveFormsModule} from "@angular/forms";
import { InputNumberModule } from 'primeng/inputnumber';
import {NgClass} from "@angular/common";
import {FloatLabel} from "primeng/floatlabel";

@Component({
  selector: 'kui-input-number',
  imports: [
    InputNumberModule,
    ReactiveFormsModule,
    NgClass,
    FloatLabel,
  ],
  templateUrl: './kui-input-number.html',
  styleUrl: './kui-input-number.scss'
})
export class InputNumber {
  // Inputs
  label = input<string>('Nombre');
  control = input.required<FormControl>();
  min = input<number | undefined>(undefined);
  max = input<number | undefined>(undefined);
  step = input<number>(1);
  disabled = input<boolean>(false);
  readonly = input<boolean>(false);
  mode = input<'decimal' | 'currency'>('decimal');
  minFractionDigits = input<number>(0);
  maxFractionDigits = input<number>(0);
  hint = input<string>('');


  isFocused = signal<boolean>(false);

  focus = output<void>();
  blur = output<void>();
  clear = output<void>();



  onFocus(event: Event): void {
    this.isFocused.set(true);
    this.focus.emit();
  }

  onBlur(event: Event): void {
    this.isFocused.set(false);
    this.blur.emit();
  }

  onClear(event: any): void {
    this.clear.emit();
    return
  }
}
