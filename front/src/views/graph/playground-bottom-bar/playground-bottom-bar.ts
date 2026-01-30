import { Component, signal, ViewChild, HostListener, output, model } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TabsModule } from 'primeng/tabs';
import { Console } from '../console/console';
import { JobConfig } from '../job-config/job-config';

@Component({
  selector: 'app-playground-bottom-bar',
  standalone: true,
  imports: [CommonModule, TabsModule, Console, JobConfig],
  templateUrl: './playground-bottom-bar.html',
  styleUrl: './playground-bottom-bar.css',
})
export class PlaygroundBottomBar {
  @ViewChild('consoleComponent') consoleComponent?: Console;

  // Output events (forwarded from console)
  onSave = output<void>();
  onExecute = output<void>();
  onStop = output<void>();

  // State
  height = signal(200);
  activeTab = model<string>('console');

  // Resize state
  private isResizing = false;
  private startY = 0;
  private startHeight = 0;
  private readonly MIN_HEIGHT = 100;
  private readonly MAX_HEIGHT = 600;

  // Resize methods
  onResizeStart(event: MouseEvent) {
    event.preventDefault();
    this.isResizing = true;
    this.startY = event.clientY;
    this.startHeight = this.height();
  }

  @HostListener('document:mousemove', ['$event'])
  onResizeMove(event: MouseEvent) {
    if (!this.isResizing) return;

    const deltaY = this.startY - event.clientY;
    const newHeight = Math.min(
      this.MAX_HEIGHT,
      Math.max(this.MIN_HEIGHT, this.startHeight + deltaY)
    );

    this.height.set(newHeight);
  }

  @HostListener('document:mouseup')
  onResizeEnd() {
    this.isResizing = false;
  }

  // Forward console events
  onConsoleSave() {
    this.onSave.emit();
  }

  onConsoleExecute() {
    this.onExecute.emit();
  }

  onConsoleStop() {
    this.onStop.emit();
  }

  // Public methods to access console
  getConsole(): Console | undefined {
    return this.consoleComponent;
  }
}
