import { Component, input, output } from '@angular/core';
import { FormControl, ReactiveFormsModule } from '@angular/forms';
import { DatePicker } from 'primeng/datepicker';
import { FloatLabel } from 'primeng/floatlabel';

@Component({
  selector: 'app-kui-date-picker',
  imports: [DatePicker, FloatLabel, ReactiveFormsModule],
  templateUrl: './kui-date-picker.html',
  styleUrl: './kui-date-picker.scss',
})
export class KuiDatePicker {
  control = input.required<FormControl<Date | null>>();
  label = input.required<string>();
  inputId = input<string>('datepicker-' + Math.random().toString(36).substring(7));
  dateFormat = input<string>('dd/mm/yy');
  showIcon = input<boolean>(true);
  iconDisplay = input<'button' | 'input'>('input');
  showButtonBar = input<boolean>(true);
  customClass = input<string>('');

  onSelect = output<Date>();
  onClear = output<void>();

  handleSelect(date: Date): void {
    this.onSelect.emit(date);
  }

  handleClear(): void {
    this.onClear.emit();
  }
}
