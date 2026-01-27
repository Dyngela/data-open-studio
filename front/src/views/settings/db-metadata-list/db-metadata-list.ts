import { Component, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { MetadataService } from '../../../core/api/metadata.service';
import { DbMetadata, CreateDbMetadataRequest, UpdateDbMetadataRequest } from '../../../core/api/metadata.type';

@Component({
  selector: 'app-db-metadata-list',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  templateUrl: './db-metadata-list.html',
  styleUrl: './db-metadata-list.css',
})
export class DbMetadataList {
  private metadataService = inject(MetadataService);
  private fb = inject(FormBuilder);

  // Data
  metadataResult = this.metadataService.getAllDb();
  metadata = computed(() => this.metadataResult.data() ?? []);
  isLoading = this.metadataResult.isLoading;

  // Modal state
  showModal = signal(false);
  editingItem = signal<DbMetadata | null>(null);
  isSubmitting = signal(false);

  // Form
  form: FormGroup = this.fb.group({
    host: ['', [
      Validators.required,
      Validators.maxLength(255),
      Validators.pattern(/^[a-zA-Z0-9]([a-zA-Z0-9\-\.]*[a-zA-Z0-9])?$/)
    ]],
    port: ['', [
      Validators.required,
      Validators.pattern(/^\d+$/),
      Validators.min(1),
      Validators.max(65535)
    ]],
    user: ['', [
      Validators.required,
      Validators.maxLength(128)
    ]],
    password: ['', [
      Validators.required,
      Validators.maxLength(256)
    ]],
    databaseName: ['', [
      Validators.required,
      Validators.maxLength(128),
      Validators.pattern(/^[a-zA-Z_][a-zA-Z0-9_-]*$/)
    ]],
    sslMode: ['disable', Validators.required],
  });

  openCreateModal() {
    this.editingItem.set(null);
    this.form.reset({
      host: '',
      port: '5432',
      user: '',
      password: '',
      databaseName: '',
      sslMode: 'disable',
    });
    this.showModal.set(true);
  }

  openEditModal(item: DbMetadata) {
    this.editingItem.set(item);
    this.form.patchValue({
      host: item.host,
      port: item.port,
      user: item.user,
      password: item.password,
      databaseName: item.databaseName,
      sslMode: item.sslMode || 'disable',
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
    const formValue = this.form.value;
    const editing = this.editingItem();

    if (editing) {
      const mutation = this.metadataService.updateDb(
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
      mutation.execute(formValue as UpdateDbMetadataRequest);
    } else {
      const mutation = this.metadataService.createDb(
        () => {
          this.isSubmitting.set(false);
          this.closeModal();
          this.metadataResult.refresh();
        },
        () => {
          this.isSubmitting.set(false);
        }
      );
      mutation.execute(formValue as CreateDbMetadataRequest);
    }
  }

  onDelete(item: DbMetadata) {
    if (!confirm(`Supprimer la connexion "${item.databaseName}" ?`)) return;

    const mutation = this.metadataService.deleteDb(
      item.id,
      () => {
        this.metadataResult.refresh();
      }
    );
    mutation.execute();
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
      if (fieldName === 'host') return 'Format de host invalide';
      if (fieldName === 'port') return 'Le port doit être un nombre';
      if (fieldName === 'databaseName') return 'Nom de base invalide (lettres, chiffres, _, -)';
    }
    if (field.errors['min']) return 'Le port minimum est 1';
    if (field.errors['max']) return 'Le port maximum est 65535';
    if (field.errors['maxlength']) return 'Valeur trop longue';

    return 'Valeur invalide';
  }
}
