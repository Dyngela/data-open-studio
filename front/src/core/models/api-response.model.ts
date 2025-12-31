/**
 * Standard API response wrapper
 */
export interface ApiResponse<T> {
  data: T;
  message?: string;
  success: boolean;
  timestamp?: string;
}

/**
 * Paginated response
 */

export interface Page<T> {
  data: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}


/**
 * API error response
 */
export interface ApiError<T> {
  message: string;
  data: T;
}

/**
 * Query parameters for list endpoints
 */
export interface QueryParams {
  page?: number;
  pageSize?: number;
  sort?: string;
  order?: 'asc' | 'desc';
  search?: string;
  [key: string]: any;
}
