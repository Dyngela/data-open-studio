import { Component } from '@angular/core';
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';

@Component({
  selector: 'app-settings',
  standalone: true,
  imports: [RouterOutlet, RouterLink, RouterLinkActive],
  templateUrl: './settings.html',
  styleUrl: './settings.css',
})
export class Settings {
  tabs = [
    { label: 'Connexions DB', path: 'db', icon: 'pi pi-database' },
    { label: 'Connexions SFTP', path: 'sftp', icon: 'pi pi-cloud' },
    { label: 'Connexions Email', path: 'email', icon: 'pi pi-envelope' },
  ];
}