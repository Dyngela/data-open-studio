import {AbstractControl, FormControl} from "@angular/forms";

export function fc<T>(control: AbstractControl<T> | null): FormControl<T> {
  return control as FormControl<T>;
}
