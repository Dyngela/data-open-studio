import {Injectable, signal, WritableSignal} from '@angular/core';
import {DbMetadata, SftpMetadata} from '../api/metadata.type';

/**
 * Service to manage local metadata for databases and SFTP connections
 */
@Injectable({
  providedIn: 'root',
})
export class MetadataLocalService {
  public databaseMetadata: WritableSignal<DbMetadata[]> = signal([]);
  public sftpMetadata: WritableSignal<SftpMetadata[]> = signal([]);

  public initialize(db?: DbMetadata[], sftp?: SftpMetadata[]): void {
    if (db) {
      this.databaseMetadata.set(db);
    }
    if (sftp) {
      this.sftpMetadata.set(sftp);
    }
  }
}
