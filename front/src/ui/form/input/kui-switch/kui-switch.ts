import {Component, input, output} from '@angular/core';
import {FormControl, ReactiveFormsModule} from "@angular/forms";
import {ToggleSwitch} from "primeng/toggleswitch";
import {NgClass} from "@angular/common";

@Component({
  selector: 'kui-switch',
  imports: [
    ToggleSwitch,
    ReactiveFormsModule,
    NgClass,
  ],
  templateUrl: './kui-switch.html',
  styleUrl: './kui-switch.scss'
})
export class KuiSwitch {
  // Inputs
  label = input<string>('Activer');
  control = input.required<FormControl>();
  disabled = input<boolean>(false);
  hint = input<string>('');

  change = output<boolean>();

  onChange(event: any): void {
    this.change.emit(event);
  }
}
