import {Component, input, output} from '@angular/core';
import {FormControl, ReactiveFormsModule} from "@angular/forms";
import {Slider} from "primeng/slider";
import {NgClass} from "@angular/common";

@Component({
  selector: 'kui-slider',
  imports: [
    Slider,
    ReactiveFormsModule,
    NgClass,
  ],
  templateUrl: './kui-slider.html',
  styleUrl: './kui-slider.css'
})
export class KuiSlider {
  // Inputs
  label = input<string>('Valeur');
  control = input.required<FormControl>();
  min = input<number>(0);
  max = input<number>(100);
  step = input<number>(1);
  disabled = input<boolean>(false);
  hint = input<string>('');

  change = output<number>();

  onChange(event: any): void {
    this.change.emit(event.value);
  }
}
