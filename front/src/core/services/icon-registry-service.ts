import { Injectable } from '@angular/core';



@Injectable({
  providedIn: 'root',
})
export class IconRegistryService {
  private registry = new Map<string, string>();

  registerIcons(icons: Record<string, string>) {
    Object.entries(icons).forEach(([name, content]) => this.registry.set(name, content));
  }

  getIcon(name: string): string {
    return this.registry.get(name) ?? '';
  }
}
