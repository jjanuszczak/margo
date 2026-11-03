import * as monaco from 'monaco-editor/editor/editor.api.js';
// The API entry point does not register editing/navigation key commands.
import 'monaco-editor/editor/browser/coreCommands.js';
import 'monaco-editor/editor/contrib/clipboard/browser/clipboard.js';
import 'monaco-editor/editor/contrib/wordOperations/browser/wordOperations.js';
import 'monaco-editor/editor/contrib/linesOperations/browser/linesOperations.js';
import 'monaco-editor/editor/contrib/unicodeHighlighter/browser/unicodeHighlighter.js';
import 'monaco-editor/editor/contrib/find/browser/findController.js';
import 'monaco-editor/editor/contrib/folding/browser/folding.js';
import 'monaco-editor/editor/contrib/bracketMatching/browser/bracketMatching.js';
import 'monaco-editor/editor/contrib/contextmenu/browser/contextmenu.js';
import { language as markdown, conf as markdownConfiguration } from 'monaco-editor/languages/definitions/markdown/markdown.js';
import { language as yaml, conf as yamlConfiguration } from 'monaco-editor/languages/definitions/yaml/yaml.js';
import EditorWorker from 'monaco-editor/editor/editor.worker.js?worker&inline';

// The worker is built into the application, with no CDN or remote language service.
self.MonacoEnvironment = { getWorker: () => new EditorWorker() };
for (const [id, language, configuration] of [['markdown', markdown, markdownConfiguration], ['yaml', yaml, yamlConfiguration]]) {
  monaco.languages.register({ id });
  monaco.languages.setMonarchTokensProvider(id, language);
  monaco.languages.setLanguageConfiguration(id, configuration);
}

export function createSourceEditor(element, onChange) {
  const sessions = new Map();
  let currentPath = '', loading = false;
  const editor = monaco.editor.create(element, {
    model: null,
    theme: 'vs-dark',
    automaticLayout: true,
    minimap: { enabled: false },
    fontFamily: '"SF Mono", Consolas, monospace',
    fontSize: 13,
    lineHeight: 22,
    wordWrap: 'on',
    lineNumbers: 'on',
    scrollBeyondLastLine: false,
    renderWhitespace: 'selection',
    padding: { top: 12, bottom: 12 },
    folding: true,
    links: false,
    ariaLabel: 'Markdown source editor',
    readOnly: true,
    fixedOverflowWidgets: true,
    stickyScroll: { enabled: false },
  });
  editor.onDidChangeModelContent(() => {
    if (!loading && currentPath) onChange(editor.getValue());
  });
  return {
    load(file, { preserveHistory = false } = {}) {
      if (currentPath && sessions.has(currentPath)) sessions.get(currentPath).viewState = editor.saveViewState();
      loading = true;
      try {
        let session = sessions.get(file.path);
        if (!preserveHistory || session?.model.getValue() !== file.content) {
          if (session) { if (editor.getModel() === session.model) editor.setModel(null); session.model.dispose(); }
          const language = /\.ya?ml$/i.test(file.path) ? 'yaml' : 'markdown';
          session = { model: monaco.editor.createModel(file.content, language, monaco.Uri.from({ scheme: 'inmemory', authority: 'margo', path: '/' + file.path })), viewState: null };
          sessions.set(file.path, session);
        }
        currentPath = file.path;
        editor.setModel(session.model);
        if (session.viewState) editor.restoreViewState(session.viewState);
        editor.updateOptions({ readOnly: false });
      } finally { loading = false; }
    },
    reset() {
      currentPath = '';
      editor.setModel(null);
      for (const session of sessions.values()) session.model.dispose();
      sessions.clear();
      editor.updateOptions({ readOnly: true });
    },
    insert(text) {
      editor.pushUndoStop();
      editor.executeEdits('margo-media', [{ range: editor.getSelection(), text, forceMoveMarkers: true }]);
      editor.pushUndoStop();
    },
    goToLine(line) {
      const model = editor.getModel();
      if (!model) return;
      const lineNumber = Math.max(1, Math.min(line || 1, model.getLineCount()));
      editor.setPosition({ lineNumber, column: 1 });
      editor.revealLineInCenter(lineNumber);
      editor.focus();
    },
    history(action) {
      if (editor.hasTextFocus() || editor.hasWidgetFocus()) {
        editor.trigger('margo-menu', action, null);
      } else {
        // Preserve native history for dialog fields and other text inputs.
        document.execCommand(action);
      }
    },
    setTheme(theme) { monaco.editor.setTheme(theme); },
    focus() { editor.focus(); },
    hasFocus() { return editor.hasTextFocus() || editor.hasWidgetFocus(); },
    selectedText() { const selection = editor.getSelection(); return selection && !selection.isEmpty() ? editor.getModel()?.getValueInRange(selection) || '' : ''; },
  };
}
