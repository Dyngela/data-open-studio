import { Component, input, output, inject } from '@angular/core';
import { FileUpload, FileUploadModule, FileSelectEvent, FileRemoveEvent, FileUploadEvent, FileUploadHandlerEvent, FileUploadErrorEvent } from 'primeng/fileupload';
import { MessageService } from 'primeng/api';

@Component({
  selector: 'app-kui-file-upload',
  imports: [FileUploadModule],
  templateUrl: './kui-file-upload.html',
  styleUrl: './kui-file-upload.scss',
})
export class KuiFileUpload {
  private messageService = inject(MessageService);

  multiple = input<boolean>(true);
  accept = input<string>('*');
  maxFileSize = input<number>(10000000); // 10MB par défaut
  disabled = input<boolean>(false);
  showCancelButton = input<boolean>(true);
  chooseLabel = input<string>('Choisir');
  uploadLabel = input<string>('');
  cancelLabel = input<string>('Annuler');
  emptyMessage = input<string>('Glissez et déposez vos fichiers ici');
  customClass = input<string>('');
  invalidFileSizeMessage = input<string>('Le fichier {0} dépasse la taille maximale de {1}');

  onSelect = output<FileSelectEvent>();
  onRemove = output<FileRemoveEvent>();
  onClear = output<void>();
  onUpload = output<FileUploadEvent>();
  onCustomUpload = output<FileUploadHandlerEvent>();
  onError = output<FileUploadErrorEvent>();

  handleSelect(event: FileSelectEvent): void {
    console.log(event)
    if (event.files.length > 0 && this.accept() != '*') {
      for (let file of event.files) {
        const extension = file.name.split('.').pop() ?? '';
        if (!this.accept().includes(extension)) {
          this.messageService.add({
            severity: 'warn',
            summary: 'invalid file type',
            detail: `Le type de fichier: ${extension} n'est pas autorisé`,
          })
          return;
        }
      }
    }
    this.onSelect.emit(event);
  }

  handleRemove(event: FileRemoveEvent): void {
    console.log("remove")
    this.onRemove.emit(event);
  }

  handleClear(): void {
    console.log("clear")
    this.onClear.emit();
  }

  handleUpload(event: FileUploadEvent): void {
    this.onUpload.emit(event);
  }

  handleCustomUpload(event: FileUploadHandlerEvent): void {
    this.onCustomUpload.emit(event);
  }

  handleError(event: FileUploadErrorEvent): void {
    this.messageService.add({
      severity: 'error',
      summary: 'Erreur',
      detail: 'Une erreur est survenue lors du téléchargement du fichier',
      life: 5000
    });
    this.onError.emit(event);
  }

  formatSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }
}
