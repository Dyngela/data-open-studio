// Enums
import {Connection} from '../nodes-services/node.type';
import type {ApiNodeType} from '../../nodes/node-definition.type';

export type JobVisibility = 'public' | 'private';
export type OwningRole = 'owner' | 'editor' | 'viewer';
export type NodeType = ApiNodeType;

// Node type
export interface Node {
  id: number;
  type: NodeType;
  name: string;
  xpos: number;
  ypos: number;
  data: unknown;
}

// SharedUser - a user with access to a job
export interface SharedUser {
  id: number;
  email: string;
  prenom: string;
  nom: string;
  role: OwningRole;
}

// NotificationContact - a user to notify on job failure
export interface NotificationContact {
  id: number;
  email: string;
  prenom: string;
  nom: string;
}

// Job response without nodes (for listing)
export interface Job {
  id: number;
  name: string;
  description: string;
  filePath: string;
  creatorId: number;
  active: boolean;
  visibility: JobVisibility;
  outputPath: string;
  createdAt: string;
  updatedAt: string;
  sharedUser: SharedUser[];
  notificationContacts: NotificationContact[];
}

export interface PrintCode {
  code: string;
  steps: any[][];
}

// Job response with nodes (for single job get)
export interface JobWithNodes extends Job {
  nodes: Node[];
  connexions?: Connection[];
}

// Request: Create a new job
export interface CreateJobRequest {
  name: string;
  description?: string;
  filePath?: string;
  outputPath?: string;
  active?: boolean;
  visibility?: JobVisibility;
  sharedWith?: number[];
}

// Request: Update an existing job
export interface UpdateJobRequest {
  name?: string;
  description?: string;
  filePath?: string;
  outputPath?: string;
  active?: boolean;
  visibility?: JobVisibility;
  sharedWith?: number[];
  nodes?: Node[];
  connexions?: Connection[];
}

// Request: Share/Unshare job
export interface ShareJobRequest {
  userIds: number[];
  role?: OwningRole;
}

// Request: Add notification contact
export interface AddNotificationContactRequest {
  userId: number;
}

// Response: Delete operation
export interface DeleteResponse {
  id: number;
  deleted: boolean;
}

// User type (mock for now)
export interface User {
  id: number;
  email: string;
  prenom: string;
  nom: string;
}
