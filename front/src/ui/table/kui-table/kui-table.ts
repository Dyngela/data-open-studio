import {Component, input, output, contentChild, TemplateRef, WritableSignal, signal, OnChanges} from '@angular/core';
import { CommonModule } from '@angular/common';
import { Table, TableModule, TablePageEvent, TableRowSelectEvent, TableRowUnSelectEvent } from 'primeng/table';
import { InputText } from 'primeng/inputtext';

export interface KuiTableColumn {
  field: string;
  header: string;
  sortable?: boolean;
  template?: TemplateRef<any>;
}

@Component({
  selector: 'app-kui-table',
  imports: [CommonModule, TableModule, InputText],
  templateUrl: './kui-table.html',
  styleUrl: './kui-table.css',
})
export class KuiTable {
  data = input.required<any[]>();
  columns = input.required<KuiTableColumn[]>();

  // Pagination
  paginator = input<boolean>(true);
  rows = input<number>(10);
  rowsPerPageOptions = input<number[]>([5, 10, 25, 50]);
  totalRecords = input<number>(0);
  lazy = input<boolean>(false);
  showCurrentPageReport = input<boolean>(true);
  currentPageReportTemplate = input<string>('Affichage de {first} à {last} sur {totalRecords} entrées');

  // Filtrage
  showGlobalFilter = input<boolean>(true);
  globalFilterPlaceholder = input<string>('Rechercher...');
  globalFilterFields = input<string[]>([]);
  filterDelay = input<number>(300);

  // Templates
  actionsTemplate = contentChild<TemplateRef<any>>('actionsTemplate');

  // UI
  caption = input<string>('');
  emptyMessage = input<string>('Aucune donnée disponible');
  loading = input<boolean>(false);
  customClass = input<string>('');
  private filterTimeout: WritableSignal<any> = signal(null);

  onPage = output<TablePageEvent>();
  onSort = output<any>();
  onFilter = output<string>();

  handlePage(event: TablePageEvent): void {
    this.onPage.emit(event);
  }

  handleSort(event: any): void {
    this.onSort.emit(event);
  }

  onGlobalFilter(event: Event): void {
    const input = event.target as HTMLInputElement;
    clearTimeout(this.filterTimeout()); // reset delay

    this.filterTimeout.set(setTimeout(() => {
      this.onFilter.emit(input.value);
    }, this.filterDelay()))
  }

  getNestedValue(obj: any, path: string): any {
    return path.split('.').reduce((acc, part) => acc && acc[part], obj);
  }

  getColspan(): number {
    let count = this.columns().length;
    if (this.actionsTemplate()) count++;
    return count;
  }
}
