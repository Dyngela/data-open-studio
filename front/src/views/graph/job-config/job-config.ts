import { Component, effect, inject, input, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AutoComplete } from 'primeng/autocomplete';
import { Button } from 'primeng/button';
import { Tooltip } from 'primeng/tooltip';
import { MessageService } from 'primeng/api';
import { Toast } from 'primeng/toast';
import { JobWithNodes, NotificationContact, User } from '../../../core/api/job.type';
import { JobService } from '../../../core/api/job.service';
import { UserService } from '../../../core/api/user.service';

@Component({
  selector: 'app-job-config',
  standalone: true,
  imports: [CommonModule, AutoComplete, Button, Tooltip, Toast],
  providers: [MessageService],
  templateUrl: './job-config.html',
  styleUrl: './job-config.css',
})
export class JobConfig {
  job = input<JobWithNodes | null>(null);

  private jobService = inject(JobService);
  private userService = inject(UserService);
  private messageService = inject(MessageService);

  contacts = signal<NotificationContact[]>([]);
  filteredUsers = signal<User[]>([]);

  constructor() {
    effect(() => {
      const job = this.job();
      if (job) {
        this.contacts.set(job.notificationContacts || []);
      } else {
        this.contacts.set([]);
      }
    });
  }

  searchUsers(event: { query: string }) {
    this.userService.searchUsers(event.query, (users) => {
      const existingIds = new Set(this.contacts().map(c => c.id));
      this.filteredUsers.set(users.filter(u => !existingIds.has(u.id)));
    });
  }

  addContact(user: User) {
    const job = this.job();
    if (!job) return;

    const mutation = this.jobService.addNotificationContact(
      job.id,
      (updatedJob) => {
        this.contacts.set(updatedJob.notificationContacts || []);
        this.messageService.add({
          severity: 'success',
          summary: 'Succès',
          detail: `${user.prenom} ${user.nom} ajouté aux alertes`,
        });
      },
      () => {
        this.messageService.add({
          severity: 'error',
          summary: 'Erreur',
          detail: 'Impossible d\'ajouter le contact',
        });
      }
    );
    mutation.execute({ userId: user.id });
  }

  removeContact(contact: NotificationContact) {
    const job = this.job();
    if (!job) return;

    const mutation = this.jobService.removeNotificationContact(
      job.id,
      contact.id,
      (updatedJob) => {
        this.contacts.set(updatedJob.notificationContacts || []);
        this.messageService.add({
          severity: 'success',
          summary: 'Succès',
          detail: `${contact.prenom} ${contact.nom} retiré des alertes`,
        });
      },
      () => {
        this.messageService.add({
          severity: 'error',
          summary: 'Erreur',
          detail: 'Impossible de retirer le contact',
        });
      }
    );
    mutation.execute();
  }
}
