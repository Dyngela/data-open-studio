import {
  Component,
  computed,
  input,
  inject,
  ElementRef,
  NgZone,
  ViewChild,
  signal,
  effect,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LayoutService } from '../../core/services/layout-service';
import { KuiModalHeader } from '../../ui/modal/kui-modal-header/kui-modal-header';
import { NodeInstance } from '../../core/nodes-services/node.type';
import { NodeGraphService } from '../../core/nodes-services/node-graph.service';
import { JobStateService } from '../../core/nodes-services/job-state.service';
import { DataModel } from '../../core/api/metadata.type';
import { isScriptConfig, ScriptNodeConfig } from './definition';
import { ScriptService } from '../../core/api/script.service';
import { ScriptCompilationError, ScriptResponse } from '../../core/api/script.type';

declare const monaco: any;

interface ScriptOutputField {
  name: string;
  type: string;
  nullable: boolean;
}

@Component({
  selector: 'app-script-modal',
  standalone: true,
  imports: [CommonModule, FormsModule, KuiModalHeader],
  templateUrl: './script.modal.html',
  styleUrl: './script.modal.css',
})
export class ScriptModal {
  @ViewChild('editorContainer') editorContainer!: ElementRef;

  private readonly compilationMarkerOwner = 'script-compilation';

  private layout = inject(LayoutService);
  private nodeGraph = inject(NodeGraphService);
  private jobState = inject(JobStateService);
  private zone = inject(NgZone);
  private scriptService = inject(ScriptService);

  node = input.required<NodeInstance>();
  modalTitle = computed(() => this.node().name ?? this.node().type.label);

  schemaRowExpanded = signal(true);

  protected upstreamInputs = computed(() => {
    return this.jobState.getUpstreamSchemas(this.node().id);
  });

  protected selectedInputPortId = signal<number | null>(null);
  protected outputFields = signal<ScriptOutputField[]>([{ name: '', type: '', nullable: false }]);

  protected selectedInput = computed(() => {
    const selectedPortId = this.selectedInputPortId();
    if (selectedPortId === null) return null;
    return this.upstreamInputs().find(inputValue => inputValue.portId === selectedPortId) ?? null;
  });

  protected canSave = computed(() => {
    const rows = this.outputFields();
    const filledRows = rows.filter(row => row.name.trim() || row.type.trim());

    if (filledRows.length === 0) return false;

    const hasInvalidRow = filledRows.some(row => !row.name.trim() || !row.type.trim());
    if (hasInvalidRow) return false;

    const normalizedNames = filledRows.map(row => row.name.trim().toLowerCase());
    const hasDuplicateName = new Set(normalizedNames).size !== normalizedNames.length;
    if (hasDuplicateName) return false;

    if (this.upstreamInputs().length > 0 && !this.selectedInput()) return false;

    // Require a successful compilation for the current editor content.
    if (!this.hasCompiledCurrentCode()) return false;

    // Block save when the last compilation run returned an error.
    if (this.hasError()) return false;

    return true;
  });

  // Mapping des types métier vers les types Python
  private readonly typeMappings: Record<string, { pyType: string; defaultVal: string }> = {
    // Chaînes
    string:    { pyType: 'str',   defaultVal: '""' },
    str:       { pyType: 'str',   defaultVal: '""' },
    text:      { pyType: 'str',   defaultVal: '""' },
    varchar:   { pyType: 'str',   defaultVal: '""' },
    char:      { pyType: 'str',   defaultVal: '""' },
    // Entiers
    int:       { pyType: 'int',   defaultVal: '0' },
    integer:   { pyType: 'int',   defaultVal: '0' },
    int8:      { pyType: 'int',   defaultVal: '0' },
    int32:     { pyType: 'int',   defaultVal: '0' },
    int64:     { pyType: 'int',   defaultVal: '0' },
    bigint:    { pyType: 'int',   defaultVal: '0' },
    smallint:  { pyType: 'int',   defaultVal: '0' },
    // Flottants
    float:     { pyType: 'float', defaultVal: '0.0' },
    float32:   { pyType: 'float', defaultVal: '0.0' },
    float64:   { pyType: 'float', defaultVal: '0.0' },
    double:    { pyType: 'float', defaultVal: '0.0' },
    decimal:   { pyType: 'float', defaultVal: '0.0' },
    numeric:   { pyType: 'float', defaultVal: '0.0' },
    real:      { pyType: 'float', defaultVal: '0.0' },
    // Booléens
    bool:      { pyType: 'bool',  defaultVal: 'False' },
    boolean:   { pyType: 'bool',  defaultVal: 'False' },
    // Dates / temps
    date:      { pyType: 'str',   defaultVal: '"2024-01-01"' },
    datetime:  { pyType: 'str',   defaultVal: '"2024-01-01T00:00:00"' },
    timestamp: { pyType: 'str',   defaultVal: '"2024-01-01T00:00:00"' },
    timestamptz: { pyType: 'str',   defaultVal: '"2024-01-01T00:00:00Z"' },
    time:      { pyType: 'str',   defaultVal: '"00:00:00"' },
  };

  private toPyType(rawType: string, nullable: boolean = false): { pyType: string; defaultVal: string } {
    const key = rawType.toLowerCase().trim();
    const mapped = this.typeMappings[key] ?? { pyType: 'Any', defaultVal: 'None' };
    return {
      pyType: nullable ? `Optional[${mapped.pyType}]` : mapped.pyType,
      defaultVal: nullable ? 'None' : mapped.defaultVal,
    };
  }

  // Code généré dynamiquement selon toutes les entrées upstream et la sortie, avec typage Python
  protected defaultCode = computed(() => {
    const inputs = this.upstreamInputs();
    const outputFields = this.outputFields().filter(f => f.name.trim() && f.type.trim());

    // Détermine si on a besoin d'Optional ou Any dans les imports
    const needsOptional = outputFields.some(f => f.nullable);
    const needsAny =
      inputs.some(inp => inp.schema.some(col => !this.typeMappings[col.type?.toLowerCase()])) ||
      outputFields.some(f => !this.typeMappings[f.type?.toLowerCase()]);

    const typingImports = ['TypedDict'];
    if (needsOptional) typingImports.push('Optional');
    if (needsAny) typingImports.push('Any');

    // Un TypedDict par connexion upstream
    const inputTypedDicts = inputs.map(inp => {
      const className = `${this.toPascalCase(inp.name)}Row`;
      const fields = inp.schema.map(col => {
        const { pyType } = this.toPyType(col.type);
        return `    ${col.name}: ${pyType}`;
      }).join('\n');
      return `class ${className}(TypedDict):\n${fields}`;
    });

    // TypedDict de sortie
    const outputTypedDict = outputFields.length
      ? (
          `class OutputRow(TypedDict):\n` +
          outputFields.map(f => {
            const { pyType } = this.toPyType(f.type, f.nullable);
            return `    ${f.name}: ${pyType}`;
          }).join('\n')
        )
      : '';

    // Un paramètre par connexion, en liste : users: List[UsersRow], tickets: List[TicketsRow]
    const inputArgs = inputs.length
      ? inputs.map(inp => `${inp.name}: List[${this.toPascalCase(inp.name)}Row]`).join(', ')
      : '';

    const outputRowDefault = outputFields.length
      ? (
          `        row: OutputRow = {\n` +
          outputFields.map(f => {
            const { defaultVal } = this.toPyType(f.type, f.nullable);
            return `            "${f.name}": ${defaultVal},`;
          }).join('\n') +
          `\n        }`
        )
      : `        row: dict = {}`;

    // Nom de la première entrée pour la boucle principale
    const firstInput = inputs[0]?.name ?? 'rows';

    return (
      `from typing import ${['List', ...typingImports].join(', ')}\n` +
      `\n` +
      (inputTypedDicts.length ? `${inputTypedDicts.join('\n\n')}\n\n` : '') +
      (outputTypedDict ? `${outputTypedDict}\n\n` : '') +
      `def transform(${inputArgs}) -> List[OutputRow]:\n` +
      `    results: List[OutputRow] = []\n` +
      `\n` +
      `    for item in ${firstInput}:\n` +
      `${outputRowDefault}\n` +
      `\n` +
      `        # Écrivez votre transformation ici\n` +
      `\n` +
      `        results.append(row)\n` +
      `\n` +
      `    return results\n`
    );
  });

  // Code sauvegardé depuis la config existante (prioritaire sur le code généré)
  private savedCode: string | null = null;

  protected output = signal('');
  protected isLoading = signal(false);
  protected hasError = signal(false);
  protected compilationError = signal<ScriptCompilationError | null>(null);
  protected editorVersion = signal(0);
  protected lastSuccessfulCompilationVersion = signal<number | null>(null);
  protected hasCompiledCurrentCode = computed(() => {
    return this.lastSuccessfulCompilationVersion() === this.editorVersion();
  });

  private editor: any;

  constructor() {
    // Met à jour l'éditeur quand l'entrée ou la sortie change,
    // uniquement si l'utilisateur n'a pas encore modifié le code manuellement.
    effect(() => {
      const newCode = this.defaultCode();

      if (!this.editor || this.savedCode !== null) return;

      const currentValue = this.editor.getValue();
      if (currentValue !== newCode) {
        this.editor.setValue(newCode);
      }
    });

    effect(() => {
      const inputs = this.upstreamInputs();
      const selectedId = this.selectedInputPortId();

      if (selectedId === null && inputs.length > 0) {
        this.selectedInputPortId.set(inputs[0].portId);
      }
    });
  }

  ngOnInit() {
    const config = this.jobState.getNodeConfig(this.node().id);

    if (isScriptConfig(config)) {
      if (config.dataModels?.length) {
        this.outputFields.set(
          config.dataModels.map(model => ({
            name: model.name,
            type: model.type,
            nullable: model.nullable ?? false,
          })),
        );
      }

      if (config.input?.portId !== undefined) {
        this.selectedInputPortId.set(config.input.portId);
      }

      // On conserve le code sauvegardé séparément pour ne pas écraser le computed
      if (config.code?.trim()) {
        this.savedCode = config.code;
      }
    }

    if (this.upstreamInputs().length > 0 && this.selectedInputPortId() === null) {
      this.selectedInputPortId.set(this.upstreamInputs()[0].portId);
    }
  }

  ngAfterViewInit() {
    this.loadMonaco();
  }

  onCancel() {
    this.layout.closeModal();
  }

  onTitleChange(value: string) {
    const trimmed = value.trim();
    this.nodeGraph.renameNode(this.node().id, trimmed || this.node().type.label);
  }

  onSelectInput(portId: number) {
    this.selectedInputPortId.set(portId);
  }

  addOutputField() {
    this.outputFields.update(fields => [...fields, { name: '', type: '', nullable: false }]);
  }

  removeOutputField(index: number) {
    this.outputFields.update(fields => {
      if (fields.length <= 1) {
        return [{ name: '', type: '', nullable: false }];
      }
      return fields.filter((_, i) => i !== index);
    });
  }

  updateOutputName(index: number, value: string) {
    this.outputFields.update(fields => {
      const next = [...fields];
      next[index] = { ...next[index], name: value };
      return next;
    });
  }

  updateOutputType(index: number, value: string) {
    this.outputFields.update(fields => {
      const next = [...fields];
      next[index] = { ...next[index], type: value };
      return next;
    });
  }

  toggleNullable(index: number, checked: boolean) {
    this.outputFields.update(fields => {
      const next = [...fields];
      next[index] = { ...next[index], nullable: checked };
      return next;
    });
  }

  onSave() {
    if (!this.canSave()) return;

    const selectedInput = this.selectedInput() ?? this.upstreamInputs()[0] ?? null;
    const dataModels: DataModel[] = this.outputFields()
      .filter(row => row.name.trim() || row.type.trim())
      .map(row => ({
        name: row.name.trim(),
        type: row.type.trim(),
        goType: '',
        nullable: row.nullable,
      }));

    const config: ScriptNodeConfig = {
      kind: 'script',
      code: this.getEditorValue(),
      dataModels,
    };

    if (selectedInput) {
      config.input = {
        name: selectedInput.name,
        portId: selectedInput.portId,
      };
    }

    this.jobState.setNodeConfig(this.node().id, config);
    this.layout.closeModal();
  }

  runCode() {
    if (!this.editor) return;

    this.isLoading.set(true);
    this.output.set('');
    this.hasError.set(false);
    this.compilationError.set(null);
    this.clearCompilationMarkers();

    const code = this.getEditorValue();
    const compiledVersion = this.editorVersion();

    const mutation = this.scriptService.executeCode(
      response => {
        this.handleCompilationResponse(response, compiledVersion);
      },
      error => {
        this.isLoading.set(false);

        // Ignore stale failures from an older compilation request.
        if (compiledVersion !== this.editorVersion()) {
          return;
        }

        this.output.set(error?.message || 'Une erreur est survenue');
        this.hasError.set(true);
        this.compilationError.set(null);
        this.lastSuccessfulCompilationVersion.set(null);
        this.clearCompilationMarkers();
      },
    );

    mutation.execute({ code });
  }

  private toPascalCase(str: string): string {
    return str
      .replace(/[^a-zA-Z0-9]+(.)/g, (_, chr) => chr.toUpperCase())
      .replace(/^(.)/, chr => chr.toUpperCase());
  }

  private loadMonaco() {
    const win = window as any;

    win.MonacoEnvironment = {
      getWorkerUrl: (_: any, label: string) => {
        return `assets/monaco-editor/min/vs/base/worker/workerMain.js`;
      },
    };

    const onMonacoReady = () => {
      win.require.config({
        paths: { vs: 'assets/monaco-editor/min/vs' },
      });
      win.require(['vs/editor/editor.main'], () => {
        this.zone.run(() => this.initEditor());
      });
    };

    if (win.monaco && win.require) {
      onMonacoReady();
      return;
    }

    const existingLoader = document.querySelector(
      'script[data-monaco-loader="true"]',
    ) as HTMLScriptElement | null;

    if (existingLoader) {
      if (win.require) {
        onMonacoReady();
      } else {
        existingLoader.addEventListener('load', onMonacoReady, { once: true });
      }
      return;
    }

    const script = document.createElement('script');
    script.src = 'assets/monaco-editor/min/vs/loader.js';
    script.setAttribute('data-monaco-loader', 'true');
    script.onload = onMonacoReady;
    document.head.appendChild(script);
  }

  private initEditor() {
    if (this.editor) return;

    // Le code sauvegardé est prioritaire, sinon on utilise le template généré
    const initialValue = this.savedCode ?? this.defaultCode();

    this.editor = monaco.editor.create(this.editorContainer.nativeElement, {
      value: initialValue,
      language: 'python',
      theme: 'vs-dark',
      fontSize: 14,
      minimap: { enabled: false },
      scrollBeyondLastLine: false,
      automaticLayout: true,
    });

    // Initial content starts at version 0.
    this.editorVersion.set(0);

    // Any edit invalidates the last successful compilation for save purposes.
    this.editor.onDidChangeModelContent(() => {
      this.editorVersion.update(version => version + 1);

      if (this.hasError()) {
        this.hasError.set(false);
        this.compilationError.set(null);
        this.clearCompilationMarkers();
      }
    });
  }

  private getEditorValue(): string {
    const value = this.editor?.getValue?.();
    if (typeof value === 'string' && value.length > 0) {
      return value;
    }
    return this.savedCode ?? this.defaultCode();
  }

  private handleCompilationResponse(response: ScriptResponse, compiledVersion: number) {
    this.isLoading.set(false);

    // Ignore stale responses when the editor content changed meanwhile.
    if (compiledVersion !== this.editorVersion()) {
      return;
    }

    this.output.set(this.formatCompilationOutput(response));
    this.hasError.set(response.error);
    this.compilationError.set(response.compilationError ?? null);
    this.updateCompilationMarkers(response.compilationError);

    if (response.error) {
      this.lastSuccessfulCompilationVersion.set(null);
      return;
    }

    this.lastSuccessfulCompilationVersion.set(compiledVersion);
  }

  private formatCompilationOutput(response: ScriptResponse): string {
    if (!response.compilationError) {
      return response.output;
    }

    const details = response.compilationError;
    const headerParts: string[] = [];

    if (details.type) {
      headerParts.push(details.type);
    }

    if (details.line) {
      const position = details.column
        ? `ligne ${details.line}, colonne ${details.column}`
        : `ligne ${details.line}`;
      headerParts.push(position);
    }

    const header = headerParts.length > 0 ? `${headerParts.join(' - ')}: ` : '';
    const hint = this.getCompilationHint(details);

    return `${header}${details.message}${hint}`;
  }

  private getCompilationHint(details: ScriptCompilationError): string {
    if (details.type !== 'IndentationError' && details.type !== 'TabError') {
      return '';
    }

    const lineCount = this.editor?.getModel?.()?.getLineCount?.();
    if (typeof lineCount === 'number' && details.line && details.line > lineCount) {
      return '\n\nPython peut signaler la ligne qui suit le vrai problème d\'indentation. Vérifie aussi les lignes juste au-dessus de la fin du script.';
    }

    return '\n\nPython signale souvent la ligne où l\'indentation devient incohérente, pas forcément la première ligne réellement mal indentée. Vérifie aussi les lignes juste au-dessus.';
  }

  private updateCompilationMarkers(compilationError?: ScriptCompilationError) {
    this.clearCompilationMarkers();

    if (!compilationError || !this.editor || !monaco?.editor) {
      return;
    }

    const model = this.editor.getModel?.();
    if (!model) {
      return;
    }

    const lineCount = model.getLineCount();
    const lineNumber = this.resolveCompilationLine(compilationError?.line, lineCount);
    if (!lineNumber) {
      return;
    }

    const lineMaxColumn = model.getLineMaxColumn(lineNumber);
    const startColumn = this.resolveCompilationColumn(compilationError.column, lineMaxColumn);
    const endColumn = compilationError.column
      ? Math.min(compilationError.column + 1, lineMaxColumn)
      : lineMaxColumn;
    const markerMessage = compilationError.type
      ? `${compilationError.type}: ${compilationError.message}`
      : compilationError.message;

    monaco.editor.setModelMarkers(model, this.compilationMarkerOwner, [{
      startLineNumber: lineNumber,
      endLineNumber: lineNumber,
      startColumn,
      endColumn: Math.max(startColumn + 1, endColumn),
      message: markerMessage,
      severity: monaco.MarkerSeverity.Error,
    }]);

    this.editor.revealLineInCenter(lineNumber);
    this.editor.setPosition({ lineNumber, column: startColumn });
  }

  private clearCompilationMarkers() {
    const model = this.editor?.getModel?.();
    if (!model || !monaco?.editor) {
      return;
    }

    monaco.editor.setModelMarkers(model, this.compilationMarkerOwner, []);
  }

  private resolveCompilationColumn(column: number | undefined, lineMaxColumn: number): number {
    if (!column) {
      return 1;
    }

    return Math.min(Math.max(column, 1), Math.max(lineMaxColumn - 1, 1));
  }

  private resolveCompilationLine(line: number | undefined, lineCount: number): number | null {
    if (!line) {
      return null;
    }

    if (line < 1) {
      return 1;
    }

    if (line > lineCount) {
      return lineCount;
    }

    return line;
  }

  ngOnDestroy() {
    this.clearCompilationMarkers();
    this.editor?.dispose();
  }
}