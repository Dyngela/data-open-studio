import { Component, computed, input, inject, ElementRef, AfterViewInit, OnDestroy, NgZone, ViewChild } from '@angular/core';
import { LayoutService } from '../../core/services/layout-service';
import { KuiModalHeader } from '../../ui/modal/kui-modal-header/kui-modal-header';
import { NodeInstance } from '../../core/nodes-services/node.type';
import { NodeGraphService } from '../../core/nodes-services/node-graph.service';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { ScriptService } from '../../core/api/script.service';

declare const monaco: any;

@Component({
  selector: 'app-script-modal',
  standalone: true,
  imports: [KuiModalHeader],
  templateUrl: './script.modal.html',
  styleUrl: './script.modal.css',
})
export class ScriptModal {
  @ViewChild('editorContainer') editorContainer!: ElementRef;
  private layout = inject(LayoutService);
  private nodeGraph = inject(NodeGraphService);
  node = input.required<NodeInstance>();
  modalTitle = computed(() => this.node().name ?? this.node().type.label);
  private scriptService = inject(ScriptService);

  onCancel() {
    this.layout.closeModal();
  }

  onTitleChange(value: string) {
    const trimmed = value.trim();
    this.nodeGraph.renameNode(this.node().id, trimmed || this.node().type.label);
  }

  private editor: any;
  output = '';
  isLoading = false;
  hasError = false;

  constructor(private http: HttpClient, private zone: NgZone) {}

  ngAfterViewInit() {
    this.loadMonaco();
  }

  private loadMonaco() {
    const win = window as any;

    // Indique à Monaco où trouver ses workers
    win.MonacoEnvironment = {
      getWorkerUrl: (_: any, label: string) => {
        return `assets/monaco-editor/min/vs/base/worker/workerMain.js`;
      }
    };

    // Charge le loader Monaco dynamiquement
    const script = document.createElement('script');
    script.src = 'assets/monaco-editor/min/vs/loader.js';
    script.onload = () => {
      win.require.config({
        paths: { vs: 'assets/monaco-editor/min/vs' }
      });
      win.require(['vs/editor/editor.main'], () => {
        this.zone.run(() => this.initEditor());
      });
    };
    document.head.appendChild(script);
  }

  private initEditor() {
    this.editor = monaco.editor.create(this.editorContainer.nativeElement, {
      value: `# Écrivez votre code Python ici\nprint("Hello, World!")`,
      language: 'python',
      theme: 'vs-dark',
      fontSize: 14,
      minimap: { enabled: false },
      scrollBeyondLastLine: false,
      automaticLayout: true,
    });
  }

  runCode() {
    if (!this.editor) return;

    this.isLoading = true;
    this.output = '';
    this.hasError = false;

    const code = this.editor.getValue();

    const payload = this.scriptService.executeCode(
      (response) => {
        this.output = response.output;
        this.isLoading = false;
        this.hasError = response.error;
      },
      (error) => {
        this.output = error?.message || 'Une erreur est survenue';
        this.isLoading = false;
        this.hasError = true;
      }
    );

    payload.execute({ code });
  }

  ngOnDestroy() {
    this.editor?.dispose();
  }
}

/*
    const mutation = this.dbNodeService.guessSchema(
      (response) => {
        this.guessedSchema.set(response.dataModels || []);
        this.isGuessingSchema.set(false);
      },
      (error) => {
        this.guessError.set(error?.message || 'Impossible de détecter le schéma');
        this.isGuessingSchema.set(false);
      },
    );

    mutation.execute({
      nodeId: String(this.node().id),
      query,
      connectionId: conn.id,
    });
*/