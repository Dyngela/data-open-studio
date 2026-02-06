import { Component, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, FormGroup, Validators, AbstractControl, ValidationErrors } from '@angular/forms';
import { MetadataService } from '../../../core/api/metadata.service';
import { SftpMetadata, CreateSftpMetadataRequest, UpdateSftpMetadataRequest } from '../../../core/api/metadata.type';

@Component({
  selector: 'app-sftp-metadata-list',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  templateUrl: './sftp-metadata-list.html',
  styleUrl: './sftp-metadata-list.css',
})
export class SftpMetadataList {
  private metadataService = inject(MetadataService);
  private fb = inject(FormBuilder);

  // Data
  metadataResult = this.metadataService.getAllSftp();
  metadata = computed(() => this.metadataResult.data() ?? []);
  isLoading = this.metadataResult.isLoading;

  // Modal state
  showModal = signal(false);
  editingItem = signal<SftpMetadata | null>(null);
  isSubmitting = signal(false);
  authMode = signal<'password' | 'key'>('password');

  // Form
  form: FormGroup = this.fb.group({
    host: ['', [
      Validators.required,
      Validators.maxLength(255),
      Validators.pattern(/^[a-zA-Z0-9]([a-zA-Z0-9\-\.]*[a-zA-Z0-9])?$/)
    ]],
    port: [22, [
      Validators.required,
      Validators.min(1),
      Validators.max(65535)
    ]],
    user: ['', [
      Validators.required,
      Validators.maxLength(128)
    ]],
    password: ['', [Validators.maxLength(256)]],
    privateKey: ['', [Validators.maxLength(8192)]],
    basePath: ['', [
      Validators.maxLength(512),
      Validators.pattern(/^(\/[a-zA-Z0-9_\-\.]+)*\/?$/)
    ]],
    extra: ['', [Validators.maxLength(1024)]],
  }, {
    validators: [this.authValidator]
  });

  // Custom validator: require either password or privateKey
  authValidator(control: AbstractControl): ValidationErrors | null {
    const password = control.get('password')?.value;
    const privateKey = control.get('privateKey')?.value;

    if (!password && !privateKey) {
      return { authRequired: true };
    }
    return null;
  }

  openCreateModal() {
    this.editingItem.set(null);
    this.authMode.set('password');
    this.form.reset({
      host: '',
      port: 22,
      user: '',
      password: '',
      privateKey: '',
      basePath: '',
      extra: '',
    });
    this.showModal.set(true);
  }

  openEditModal(item: SftpMetadata) {
    this.editingItem.set(item);
    this.authMode.set(item.privateKey ? 'key' : 'password');
    this.form.patchValue({
      host: item.host,
      port: item.port,
      user: item.user,
      password: item.password,
      privateKey: item.privateKey,
      basePath: item.basePath,
      extra: item.extra,
    });
    this.showModal.set(true);
  }

  closeModal() {
    this.showModal.set(false);
    this.editingItem.set(null);
    this.form.reset();
  }

  switchAuthMode(mode: 'password' | 'key') {
    this.authMode.set(mode);
    // Clear the other auth field when switching
    if (mode === 'password') {
      this.form.patchValue({ privateKey: '' });
    } else {
      this.form.patchValue({ password: '' });
    }
  }

  onSubmit() {
    if (this.form.invalid || this.isSubmitting()) {
      this.form.markAllAsTouched();
      return;
    }

    this.isSubmitting.set(true);
    const formValue = { ...this.form.value, port: Number(this.form.value.port) };
    const editing = this.editingItem();

    if (editing) {
      const mutation = this.metadataService.updateSftp(
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
      mutation.execute(formValue as UpdateSftpMetadataRequest);
    } else {
      const mutation = this.metadataService.createSftp(
        () => {
          this.isSubmitting.set(false);
          this.closeModal();
          this.metadataResult.refresh();
        },
        () => {
          this.isSubmitting.set(false);
        }
      );
      mutation.execute(formValue as CreateSftpMetadataRequest);
    }
  }

  onDelete(item: SftpMetadata) {
    if (!confirm(`Supprimer la connexion SFTP "${item.host}" ?`)) return;

    const mutation = this.metadataService.deleteSftp(
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
      if (fieldName === 'basePath') return 'Chemin invalide (ex: /home/user)';
    }
    if (field.errors['min']) return 'Le port minimum est 1';
    if (field.errors['max']) return 'Le port maximum est 65535';
    if (field.errors['maxlength']) return 'Valeur trop longue';

    return 'Valeur invalide';
  }

  hasAuthError(): boolean {
    return !!(this.form.errors?.['authRequired'] && this.form.touched);
  }
}
