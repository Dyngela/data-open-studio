import { Component, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { MetadataService } from '../../../core/api/metadata.service';
import { EmailMetadata, CreateEmailMetadataRequest, UpdateEmailMetadataRequest, TestEmailConnectionResult } from '../../../core/api/metadata.type';

@Component({
  selector: 'app-email-metadata-list',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  templateUrl: './email-metadata-list.html',
  styleUrl: './email-metadata-list.css',
})
export class EmailMetadataList {
  private metadataService = inject(MetadataService);
  private fb = inject(FormBuilder);

  // Data
  metadataResult = this.metadataService.getAllEmail();
  metadata = computed(() => this.metadataResult.data() ?? []);
  isLoading = this.metadataResult.isLoading;

  // Modal state
  showModal = signal(false);
  editingItem = signal<EmailMetadata | null>(null);
  isSubmitting = signal(false);

  // Test connection
  isTestingConnection = signal(false);
  connectionTestResult = signal<TestEmailConnectionResult | null>(null);

  // Form
  form: FormGroup = this.fb.group({
    name: ['', [Validators.maxLength(255)]],
    imapHost: ['', [
      Validators.required,
      Validators.maxLength(255),
      Validators.pattern(/^[a-zA-Z0-9]([a-zA-Z0-9\-\.]*[a-zA-Z0-9])?$/)
    ]],
    imapPort: [993, [
      Validators.required,
      Validators.min(1),
      Validators.max(65535)
    ]],
    smtpHost: ['', [
      Validators.maxLength(255),
      Validators.pattern(/^[a-zA-Z0-9]([a-zA-Z0-9\-\.]*[a-zA-Z0-9])?$/)
    ]],
    smtpPort: [587, [
      Validators.min(1),
      Validators.max(65535)
    ]],
    username: ['', [
      Validators.required,
      Validators.maxLength(255)
    ]],
    password: ['', [
      Validators.required,
      Validators.maxLength(256)
    ]],
    useTls: [true],
  });

  openCreateModal() {
    this.editingItem.set(null);
    this.connectionTestResult.set(null);
    this.form.reset({
      name: '',
      imapHost: '',
      imapPort: 993,
      smtpHost: '',
      smtpPort: 587,
      username: '',
      password: '',
      useTls: true,
    });
    this.showModal.set(true);
  }

  openEditModal(item: EmailMetadata) {
    this.editingItem.set(item);
    this.connectionTestResult.set(null);
    this.form.patchValue({
      name: item.name,
      imapHost: item.imapHost,
      imapPort: item.imapPort,
      smtpHost: item.smtpHost,
      smtpPort: item.smtpPort,
      username: item.username,
      password: item.password,
      useTls: item.useTls,
    });
    this.showModal.set(true);
  }

  closeModal() {
    this.showModal.set(false);
    this.editingItem.set(null);
    this.form.reset();
  }

  onSubmit() {
    if (this.form.invalid || this.isSubmitting()) {
      this.form.markAllAsTouched();
      return;
    }

    this.isSubmitting.set(true);
    const formValue = {
      ...this.form.value,
      imapPort: Number(this.form.value.imapPort),
      smtpPort: Number(this.form.value.smtpPort),
    };
    const editing = this.editingItem();

    if (editing) {
      const mutation = this.metadataService.updateEmail(
        editing.id,
        () => {
          this.isSubmitting.set(false);
          this.closeModal();
          this.metadataResult.refresh();
        },
        () => {
          this.isSubmitting.set(false);
        }
      );
      mutation.execute(formValue as UpdateEmailMetadataRequest);
    } else {
      const mutation = this.metadataService.createEmail(
        () => {
          this.isSubmitting.set(false);
          this.closeModal();
          this.metadataResult.refresh();
        },
        () => {
          this.isSubmitting.set(false);
        }
      );
      mutation.execute(formValue as CreateEmailMetadataRequest);
    }
  }

  onDelete(item: EmailMetadata) {
    if (!confirm(`Supprimer la connexion "${item.name || item.username}" ?`)) return;

    const mutation = this.metadataService.deleteEmail(
      item.id,
      () => {
        this.metadataResult.refresh();
      }
    );
    mutation.execute();
  }

  testConnection() {
    if (this.form.invalid || this.isTestingConnection()) return;

    this.isTestingConnection.set(true);
    this.connectionTestResult.set(null);

    const formValue = {
      ...this.form.value,
      imapPort: Number(this.form.value.imapPort),
      smtpPort: Number(this.form.value.smtpPort),
    };
    const mutation = this.metadataService.testEmailConnection(
      (result) => {
        this.isTestingConnection.set(false);
        this.connectionTestResult.set(result);
      },
      () => {
        this.isTestingConnection.set(false);
        this.connectionTestResult.set({
          imapSuccess: false,
          imapMessage: 'Erreur de connexion IMAP',
          smtpSuccess: false,
          smtpMessage: 'Erreur de connexion SMTP',
        });
      }
    );
    mutation.execute(formValue as CreateEmailMetadataRequest);
  }

  // Helper methods for template
  isFieldInvalid(fieldName: string): boolean {
    const field = this.form.get(fieldName);
    return !!(field && field.invalid && field.touched);
  }

  getFieldError(fieldName: string): string {
    const field = this.form.get(fieldName);
    if (!field || !field.errors) return '';

    if (field.errors['required']) return 'Ce champ est requis';
    if (field.errors['pattern']) {
      if (fieldName === 'imapHost' || fieldName === 'smtpHost') return 'Format de host invalide';
    }
    if (field.errors['min']) return 'Le port minimum est 1';
    if (field.errors['max']) return 'Le port maximum est 65535';
    if (field.errors['maxlength']) return 'Valeur trop longue';

    return 'Valeur invalide';
  }
}
