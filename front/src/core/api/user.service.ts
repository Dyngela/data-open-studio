import { inject, Injectable } from '@angular/core';
import { BaseApiService } from '../services/base-api.service';
import { SearchCriteria } from '../services/base-api.type';
import { User } from './job.type';

@Injectable({ providedIn: 'root' })
export class UserService {
  private api = inject(BaseApiService);

  searchUsers(query: string, onSuccess: (users: User[]) => void) {
    const criteria: SearchCriteria[] = [{ name: 'q', value: query }];
    this.api.get<User[]>('/users/search', criteria, onSuccess);
  }
}
