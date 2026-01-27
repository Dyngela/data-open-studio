import {inject, Injectable} from '@angular/core';
import {DbMetadata, SftpMetadata} from '../api/metadata.type';
import {MetadataService} from '../api/metadata.service';
import {ApiResult} from './base-api.type';
import {MessageService} from 'primeng/api';

/**
 * Service to manage local metadata for databases and SFTP connections
 */
@Injectable({
  providedIn: 'root',
})
export class MetadataLocalService {
  private metadataAPI = inject(MetadataService)
  private messageService = inject(MessageService)

  db!: ApiResult<DbMetadata[]>;
  sftp!: ApiResult<SftpMetadata[]>;

  public initialize(): void {
    this.db = this.metadataAPI.getAllDb()
    if (this.db.error()) {
      this.messageService.add({
        severity: 'error',
        summary: 'Erreur',
        detail: `Impossible de charger les métadonnées des bases de données : ${this.db.error()?.message}`,
      })
    }
    this.sftp = this.metadataAPI.getAllSftp()
    if (this.sftp.error()) {
      this.messageService.add({
        severity: 'error',
        summary: 'Erreur',
        detail: `Impossible de charger les métadonnées des s : ${this.sftp.error()?.message}`,
      })
    }
  }
}
