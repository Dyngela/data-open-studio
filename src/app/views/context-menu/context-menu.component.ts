import {Component, ElementRef, Input, OnChanges, OnInit, SimpleChanges, ViewChild} from '@angular/core';
import {ContextMenuItem} from "../../models/context-menu.model";

@Component({
  selector: 'app-context-menu',
  templateUrl: './context-menu.component.html',
  styleUrls: ['./context-menu.component.css']
})
export class ContextMenuComponent implements OnChanges {
  @Input() items: ContextMenuItem[] = [];
  @Input() position: { x: number, y: number } | null = null;
  @ViewChild('searchBox') searchBox!: ElementRef;

  filteredItems: ContextMenuItem[] = [];

  ngOnChanges(changes: SimpleChanges) {
    if (changes['items']) {
      this.filteredItems = this.items;
    }
    if (changes['position'] && this.position) {
      // Focus the search box when position is set
      setTimeout(() => {
        this.searchBox.nativeElement.focus();
      }, 0);
    }
  }

  filterItems(event: Event) {
    const searchTerm = (event.target as HTMLInputElement).value.toLowerCase();
    this.filteredItems = this.items.filter(item => item.label.toLowerCase().includes(searchTerm));
  }

  onSelect(item: ContextMenuItem) {
    item.action();
  }
}
