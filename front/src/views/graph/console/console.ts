import {Component, signal, ViewChild, ElementRef, HostListener, AfterViewChecked, OnDestroy, output, inject} from '@angular/core';
import { CommonModule, DatePipe, UpperCasePipe } from '@angular/common';
import {JobService} from '../../../core/api/job.service';
import {JobRealtimeService} from '../../../core/services/base-ws.service';

export interface LogEntry {
  id: string;
  timestamp: Date;
  level: 'info' | 'warn' | 'error' | 'success' | 'debug';
  message: string;
}

@Component({
  selector: 'app-console',
  standalone: true,
  imports: [CommonModule, DatePipe, UpperCasePipe],
  templateUrl: './console.html',
  styleUrl: './console.css',
})
export class Console implements AfterViewChecked, OnDestroy {
  private jobService = inject(JobService);
  private realtime = inject(JobRealtimeService);
  private unsubProgress: () => void;

  constructor() {
    this.unsubProgress = this.realtime.onProgress((progress) => {
      const statusMap: Record<string, LogEntry['level']> = {
        running: 'info',
        completed: 'success',
        failed: 'error',
      };

      const level = statusMap[progress.status] ?? 'info';
      const rowInfo = progress.rowCount > 0 ? ` (${progress.rowCount} rows)` : '';
      this.addLog(level, `[${progress.nodeName}] ${progress.message}${rowInfo}`);

      if (progress.status === 'failed') {
        this.markError(`Node "${progress.nodeName}" failed: ${progress.message}`);
      }
    });
  }

  ngOnDestroy(): void {
    this.unsubProgress();
  }

  @ViewChild('logContainer') logContainer?: ElementRef<HTMLDivElement>;

  // Output events
  onSave = output<void>();
  onExecute = output<void>();
  onStop = output<void>();

  // State
  height = signal(200);
  isCollapsed = signal(false);
  isRunning = signal(false);
  autoScroll = signal(true);
  logs = signal<LogEntry[]>([]);
  isSaving = signal(false);


  // Resize state
  private isResizing = false;
  private startY = 0;
  private startHeight = 0;
  private readonly MIN_HEIGHT = 100;
  private readonly MAX_HEIGHT = 600;
  private readonly COLLAPSED_HEIGHT = 40;

  private logIdCounter = 0;
  private shouldScrollToBottom = false;

  ngAfterViewChecked() {
    if (this.shouldScrollToBottom && this.autoScroll() && this.logContainer) {
      this.scrollToBottom();
      this.shouldScrollToBottom = false;
    }
  }

  // Resize methods
  onResizeStart(event: MouseEvent) {
    if (this.isCollapsed()) return;

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


  // Actions
  saveJob() {
    this.onSave.emit();
  }

  executeJob() {
    if (this.isRunning()) return;

    this.isRunning.set(true);
    this.addLog('info', 'Démarrage de l\'exécution du job...');
    this.onExecute.emit();
  }

  printCode() {
    this.jobService.printCode(1,
      (data) => { 
      console.log(data)
    }).execute()
  }

  stopJob() {
    if (!this.isRunning()) return;

    this.addLog('warn', 'Arrêt demandé...');
    this.onStop.emit();
    this.isRunning.set(false);
    this.addLog('info', 'Job arrêté.');
  }

  clearLogs() {
    this.logs.set([]);
  }

  toggleAutoScroll() {
    this.autoScroll.update(v => !v);
  }

  toggleCollapse() {
    this.isCollapsed.update(v => !v);
  }

  // Public methods for parent component
  addLog(level: LogEntry['level'], message: string) {
    const entry: LogEntry = {
      id: `log-${this.logIdCounter++}`,
      timestamp: new Date(),
      level,
      message,
    };

    this.logs.update(logs => [...logs, entry]);
    this.shouldScrollToBottom = true;
  }

  setRunning(running: boolean) {
    this.isRunning.set(running);
  }

  markSuccess() {
    this.addLog('success', 'Job terminé avec succès.');
    this.isRunning.set(false);
  }

  markError(error: string) {
    this.addLog('error', `Erreur: ${error}`);
    this.isRunning.set(false);
  }

  private scrollToBottom() {
    if (this.logContainer) {
      const el = this.logContainer.nativeElement;
      el.scrollTop = el.scrollHeight;
    }
  }
}
