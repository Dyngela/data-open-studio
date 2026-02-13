import { Injectable } from '@angular/core';



@Injectable({
  providedIn: 'root',
})
export class IconRegistryService {
  private icons = new Map<string, string>();

  registerIcons(icons: Record<string, string>) {
    Object.entries(icons).forEach(([name, path]) => this.icons.set(name, path));
  }

  getIcon(name: string): string {
    return this.icons.get(name) ?? '';
  }
}
