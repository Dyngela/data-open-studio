import {inject, Injectable} from '@angular/core';
import { BaseApiService } from '../services/base-api.service';
import { ApiMutation, ApiResult } from '../services/base-api.type';
import {
  DbMetadata,
  CreateDbMetadataRequest,
  UpdateDbMetadataRequest,
  SftpMetadata,
  CreateSftpMetadataRequest,
  UpdateSftpMetadataRequest,
  EmailMetadata,
  CreateEmailMetadataRequest,
  UpdateEmailMetadataRequest,
  DeleteResponse,
  TestConnectionResult,
  TestEmailConnectionResult,
} from './metadata.type';

@Injectable({ providedIn: 'root' })
export class MetadataService {

  private api = inject(BaseApiService)
  private readonly dbPath = '/metadata/db';
  private readonly sftpPath = '/metadata/sftp';
  private readonly emailPath = '/metadata/email';


  /**
   * Get all database metadata entries
   */
  getAllDb(): ApiResult<DbMetadata[]> {
    return this.api.get<DbMetadata[]>(this.dbPath);
  }

  /**
   * Get a single database metadata by ID
   */
  getDbById(id: number): ApiResult<DbMetadata> {
    return this.api.get<DbMetadata>(`${this.dbPath}/${id}`);
  }

  /**
   * Create a new database metadata entry
   */
  createDb(
    onSuccess?: (data: DbMetadata) => void,
    onError?: (error: any) => void
  ): ApiMutation<DbMetadata, CreateDbMetadataRequest> {
    return this.api.post<DbMetadata, CreateDbMetadataRequest>(
      this.dbPath,
      onSuccess,
      onError
    );
  }

  /**
   * Update an existing database metadata entry
   */
  updateDb(
    id: number,
    onSuccess?: (data: DbMetadata) => void,
    onError?: (error: any) => void
  ): ApiMutation<DbMetadata, UpdateDbMetadataRequest> {
    return this.api.put<DbMetadata, UpdateDbMetadataRequest>(
      `${this.dbPath}/${id}`,
      onSuccess,
      onError
    );
  }

  /**
   * Delete a database metadata entry
   */
  deleteDb(
    id: number,
    onSuccess?: (data: DeleteResponse) => void,
    onError?: (error: any) => void
  ): ApiMutation<DeleteResponse, void> {
    return this.api.delete<DeleteResponse, void>(
      `${this.dbPath}/${id}`,
      onSuccess,
      onError
    );
  }


  /**
   * Test a database connection using metadata form values
   */
  testDbConnection(
    onSuccess?: (data: TestConnectionResult) => void,
    onError?: (error: any) => void
  ): ApiMutation<TestConnectionResult, CreateDbMetadataRequest> {
    return this.api.post<TestConnectionResult, CreateDbMetadataRequest>(
      `${this.dbPath}/test-connection`,
      onSuccess,
      onError
    );
  }

  /**
   * Get all SFTP metadata entries
   */
  getAllSftp(): ApiResult<SftpMetadata[]> {
    return this.api.get<SftpMetadata[]>(this.sftpPath);
  }

  /**
   * Get a single SFTP metadata by ID
   */
  getSftpById(id: number): ApiResult<SftpMetadata> {
    return this.api.get<SftpMetadata>(`${this.sftpPath}/${id}`);
  }

  /**
   * Create a new SFTP metadata entry
   */
  createSftp(
    onSuccess?: (data: SftpMetadata) => void,
    onError?: (error: any) => void
  ): ApiMutation<SftpMetadata, CreateSftpMetadataRequest> {
    return this.api.post<SftpMetadata, CreateSftpMetadataRequest>(
      this.sftpPath,
      onSuccess,
      onError
    );
  }

  /**
   * Update an existing SFTP metadata entry
   */
  updateSftp(
    id: number,
    onSuccess?: (data: SftpMetadata) => void,
    onError?: (error: any) => void
  ): ApiMutation<SftpMetadata, UpdateSftpMetadataRequest> {
    return this.api.put<SftpMetadata, UpdateSftpMetadataRequest>(
      `${this.sftpPath}/${id}`,
      onSuccess,
      onError
    );
  }

  /**
   * Delete a SFTP metadata entry
   */
  deleteSftp(
    id: number,
    onSuccess?: (data: DeleteResponse) => void,
    onError?: (error: any) => void
  ): ApiMutation<DeleteResponse, void> {
    return this.api.delete<DeleteResponse, void>(
      `${this.sftpPath}/${id}`,
      onSuccess,
      onError
    );
  }

  /**
   * Get all email metadata entries
   */
  getAllEmail(): ApiResult<EmailMetadata[]> {
    return this.api.get<EmailMetadata[]>(this.emailPath);
  }

  /**
   * Get a single email metadata by ID
   */
  getEmailById(id: number): ApiResult<EmailMetadata> {
    return this.api.get<EmailMetadata>(`${this.emailPath}/${id}`);
  }

  /**
   * Create a new email metadata entry
   */
  createEmail(
    onSuccess?: (data: EmailMetadata) => void,
    onError?: (error: any) => void
  ): ApiMutation<EmailMetadata, CreateEmailMetadataRequest> {
    return this.api.post<EmailMetadata, CreateEmailMetadataRequest>(
      this.emailPath,
      onSuccess,
      onError
    );
  }

  /**
   * Update an existing email metadata entry
   */
  updateEmail(
    id: number,
    onSuccess?: (data: EmailMetadata) => void,
    onError?: (error: any) => void
  ): ApiMutation<EmailMetadata, UpdateEmailMetadataRequest> {
    return this.api.put<EmailMetadata, UpdateEmailMetadataRequest>(
      `${this.emailPath}/${id}`,
      onSuccess,
      onError
    );
  }

  /**
   * Delete an email metadata entry
   */
  deleteEmail(
    id: number,
    onSuccess?: (data: DeleteResponse) => void,
    onError?: (error: any) => void
  ): ApiMutation<DeleteResponse, void> {
    return this.api.delete<DeleteResponse, void>(
      `${this.emailPath}/${id}`,
      onSuccess,
      onError
    );
  }

  /**
   * Test an email connection using metadata form values
   */
  testEmailConnection(
    onSuccess?: (data: TestEmailConnectionResult) => void,
    onError?: (error: any) => void
  ): ApiMutation<TestEmailConnectionResult, CreateEmailMetadataRequest> {
    return this.api.post<TestEmailConnectionResult, CreateEmailMetadataRequest>(
      `${this.emailPath}/test-connection`,
      onSuccess,
      onError
    );
  }
}
