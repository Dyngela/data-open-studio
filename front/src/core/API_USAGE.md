# API Usage Guide

## Architecture Overview

This project uses a modern, typed API interface built with Angular 20 features:

- **Base API Service**: Abstract class with common HTTP methods
- **Feature Services**: Extend base service for specific resources
- **Interceptors**: Handle auth, errors, and loading states
- **Models**: TypeScript interfaces for type safety
- **Signals**: Modern reactive state management

## Quick Start

### 1. Basic API Call

```typescript
import { Component, inject, OnInit } from '@angular/core';
import { UserService } from '../core/api/user.service';
import { User } from '../core/models';

@Component({
  selector: 'app-users',
  template: `
    @for (user of users(); track user.id) {
      <div>{{ user.name }}</div>
    }
  `
})
export class UsersComponent implements OnInit {
  private userService = inject(UserService);
  users = signal<User[]>([]);

  ngOnInit() {
    this.userService.getUsers({ page: 1, pageSize: 10 })
      .subscribe(response => {
        this.users.set(response.data);
      });
  }
}
```

### 2. Create a New Resource

```typescript
createUser() {
  const newUser: CreateUserDto = {
    name: 'John Doe',
    email: 'john@example.com',
    password: 'securePassword123'
  };

  this.userService.createUser(newUser)
    .subscribe({
      next: (user) => {
        console.log('User created:', user);
        this.messageService.add({
          severity: 'success',
          summary: 'Succès',
          detail: 'Utilisateur créé avec succès'
        });
      },
      error: (error) => {
        // Error is automatically handled by errorInterceptor
        console.error('Failed to create user:', error);
      }
    });
}
```

### 3. Update a Resource

```typescript
updateUser(userId: number) {
  const updates: UpdateUserDto = {
    name: 'Jane Doe',
    email: 'jane@example.com'
  };

  this.userService.updateUser(userId, updates)
    .subscribe(updatedUser => {
      console.log('User updated:', updatedUser);
    });
}
```

### 4. Delete a Resource

```typescript
deleteUser(userId: number) {
  this.userService.deleteUser(userId)
    .subscribe(() => {
      console.log('User deleted');
      // Refresh user list
      this.loadUsers();
    });
}
```

## Creating a New API Service

### Step 1: Define Your Models

```typescript
// src/core/models/product.model.ts
export interface Product {
  id: number;
  name: string;
  price: number;
  description?: string;
  createdAt: string;
}

export interface CreateProductDto {
  name: string;
  price: number;
  description?: string;
}

export interface UpdateProductDto {
  name?: string;
  price?: number;
  description?: string;
}
```

### Step 2: Create Your Service

```typescript
// src/core/api/product.service.ts
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { BaseApiService } from './base-api.service';
import { Product, CreateProductDto, UpdateProductDto, PaginatedResponse } from '../models';

@Injectable({
  providedIn: 'root'
})
export class ProductService extends BaseApiService {
  private endpoint = 'products';

  getProducts(params?: QueryParams): Observable<PaginatedResponse<Product>> {
    return this.getList<Product>(this.endpoint, params);
  }

  getProductById(id: number): Observable<Product> {
    return this.getWithResponse<Product>(`${this.endpoint}/${id}`);
  }

  createProduct(data: CreateProductDto): Observable<Product> {
    return this.postWithResponse<CreateProductDto, Product>(this.endpoint, data);
  }

  updateProduct(id: number, data: UpdateProductDto): Observable<Product> {
    return this.putWithResponse<UpdateProductDto, Product>(
      `${this.endpoint}/${id}`,
      data
    );
  }

  deleteProduct(id: number): Observable<void> {
    return this.deleteWithResponse<void>(`${this.endpoint}/${id}`);
  }
}
```

### Step 3: Use in Component

```typescript
import { Component, inject, signal } from '@angular/core';
import { ProductService } from '../core/api/product.service';
import { Product } from '../core/models';

@Component({
  selector: 'app-products',
  template: `
    <button (click)="loadProducts()">Load Products</button>

    @for (product of products(); track product.id) {
      <div>
        <h3>{{ product.name }}</h3>
        <p>{{ product.price | currency:'EUR' }}</p>
      </div>
    }
  `
})
export class ProductsComponent {
  private productService = inject(ProductService);
  products = signal<Product[]>([]);

  loadProducts() {
    this.productService.getProducts({ page: 1, pageSize: 20 })
      .subscribe(response => {
        this.products.set(response.data);
      });
  }
}
```

## Using Authentication

```typescript
import { Component, inject } from '@angular/core';
import { AuthService } from '../core/api/auth.service';
import { Router } from '@angular/router';

@Component({
  selector: 'app-login',
  template: `
    <form (ngSubmit)="login()">
      <input [(ngModel)]="email" type="email" />
      <input [(ngModel)]="password" type="password" />
      <button type="submit">Login</button>
    </form>
  `
})
export class LoginComponent {
  private authService = inject(AuthService);
  private router = inject(Router);

  email = '';
  password = '';

  login() {
    this.authService.login({ email: this.email, password: this.password })
      .subscribe({
        next: (response) => {
          console.log('Logged in as:', response.user);
          this.router.navigate(['/dashboard']);
        },
        error: (error) => {
          // Error automatically handled by interceptor
          console.error('Login failed:', error);
        }
      });
  }

  // Access current user (signal)
  get currentUser() {
    return this.authService.currentUser();
  }
}
```

## Using Loading State

```typescript
import { Component, inject } from '@angular/core';
import { LoadingService } from '../core/services/loading.service';

@Component({
  selector: 'app-root',
  template: `
    @if (loadingService.isLoading()) {
      <div class="loading-overlay">
        <app-kui-global-loading-spinner />
      </div>
    }

    <router-outlet />
  `
})
export class AppComponent {
  loadingService = inject(LoadingService);
}
```

## File Upload

```typescript
uploadFile(file: File) {
  this.productService.uploadFile('products/upload', file, { category: 'images' })
    .subscribe({
      next: (response) => {
        console.log('File uploaded:', response);
      },
      error: (error) => {
        console.error('Upload failed:', error);
      }
    });
}
```

## Advanced: Custom Query Parameters

```typescript
searchProducts() {
  const params: QueryParams = {
    page: 1,
    pageSize: 20,
    sort: 'price',
    order: 'desc',
    search: 'laptop',
    minPrice: 500,
    maxPrice: 2000,
    inStock: true
  };

  this.productService.getProducts(params)
    .subscribe(response => {
      this.products.set(response.data);
    });
}
```

## Environment Configuration

Update `src/environments/environment.ts` for your API:

```typescript
export const environment = {
  production: false,
  apiUrl: 'http://localhost:3000/api',
  apiVersion: 'v1'
};
```

## Error Handling

Errors are automatically handled by the `errorInterceptor`:
- Shows toast notifications
- Logs errors to console
- Redirects to login on 401
- User-friendly error messages in French

To handle specific errors:

```typescript
this.userService.getUserById(userId)
  .subscribe({
    next: (user) => {
      console.log('User loaded:', user);
    },
    error: (error: HttpErrorResponse) => {
      // Custom error handling
      if (error.status === 404) {
        this.router.navigate(['/not-found']);
      }
    }
  });
```

## Best Practices

1. **Use Signals**: For reactive state management
2. **Type Everything**: Use TypeScript interfaces
3. **Handle Subscriptions**: Use `takeUntilDestroyed()` or unsubscribe
4. **Error Handling**: Let interceptors handle common errors
5. **Loading States**: Use LoadingService for global loading
6. **Separate Concerns**: One service per resource
7. **DTOs**: Use separate DTOs for create/update operations

## Example: Complete CRUD Component

```typescript
import { Component, inject, signal, OnInit } from '@angular/core';
import { UserService } from '../core/api/user.service';
import { User, CreateUserDto, UpdateUserDto } from '../core/models';
import { MessageService } from 'primeng/api';

@Component({
  selector: 'app-user-management',
  template: `
    <app-kui-table
      [data]="users()"
      [columns]="columns"
      (onPage)="loadUsers($event)"
    >
      <ng-template #actionsTemplate let-user>
        <button (click)="editUser(user)">Edit</button>
        <button (click)="deleteUser(user.id)">Delete</button>
      </ng-template>
    </app-kui-table>
  `
})
export class UserManagementComponent implements OnInit {
  private userService = inject(UserService);
  private messageService = inject(MessageService);

  users = signal<User[]>([]);
  columns = [
    { field: 'name', header: 'Name', sortable: true },
    { field: 'email', header: 'Email', sortable: true },
    { field: 'role', header: 'Role' }
  ];

  ngOnInit() {
    this.loadUsers();
  }

  loadUsers(event?: any) {
    const page = event?.first / event?.rows + 1 || 1;
    const pageSize = event?.rows || 10;

    this.userService.getUsers({ page, pageSize })
      .subscribe(response => {
        this.users.set(response.data);
      });
  }

  createUser(userData: CreateUserDto) {
    this.userService.createUser(userData)
      .subscribe(user => {
        this.messageService.add({
          severity: 'success',
          summary: 'Success',
          detail: 'User created successfully'
        });
        this.loadUsers();
      });
  }

  editUser(user: User) {
    const updates: UpdateUserDto = { name: 'Updated Name' };

    this.userService.updateUser(user.id, updates)
      .subscribe(updatedUser => {
        this.messageService.add({
          severity: 'success',
          summary: 'Success',
          detail: 'User updated successfully'
        });
        this.loadUsers();
      });
  }

  deleteUser(userId: number) {
    this.userService.deleteUser(userId)
      .subscribe(() => {
        this.messageService.add({
          severity: 'success',
          summary: 'Success',
          detail: 'User deleted successfully'
        });
        this.loadUsers();
      });
  }
}
```
