import {
  commands,
  ExtensionContext,
  ViewColumn,
  WebviewPanel,
  window,
  workspace,
} from "vscode";
import {
  CloseAction,
  ErrorAction,
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
} from "vscode-languageclient/node";
import { execFile } from "child_process";
import { promisify } from "util";

const execFileAsync = promisify(execFile);

let client: LanguageClient | undefined;
let previewPanel: WebviewPanel | undefined;
let activePreviewURI: string | undefined;
let lastRenderedHTML: Map<string, string> = new Map();

interface DocumentRenderedParams {
  readonly uri: string;
  readonly html: string;
}

export async function activate(context: ExtensionContext): Promise<void> {
  // Register commands first so they're always available in the command palette,
  // even if the binary isn't found or validation fails.
  const openPreview = commands.registerCommand(
    "calcmark.openPreview",
    () => {
      if (previewPanel) {
        previewPanel.reveal(ViewColumn.Beside);
        return;
      }

      previewPanel = window.createWebviewPanel(
        "calcmarkPreview",
        "CalcMark Preview",
        ViewColumn.Beside,
        { enableScripts: true, retainContextWhenHidden: true }
      );

      previewPanel.onDidDispose(() => {
        previewPanel = undefined;
        activePreviewURI = undefined;
      });

      // Show content for active editor
      const editor = window.activeTextEditor;
      if (editor?.document.languageId === "calcmark") {
        activePreviewURI = editor.document.uri.toString();
        const html = lastRenderedHTML.get(activePreviewURI);
        if (html) {
          updatePreview(html);
        }
      }
    }
  );

  // Track active editor to update preview
  const editorChange = window.onDidChangeActiveTextEditor((editor) => {
    if (
      !previewPanel ||
      !editor ||
      editor.document.languageId !== "calcmark"
    ) {
      return;
    }
    const uri = editor.document.uri.toString();
    activePreviewURI = uri;
    const html = lastRenderedHTML.get(uri);
    if (html) {
      updatePreview(html);
    }
  });

  // Scroll sync: editor-to-preview (unidirectional via postMessage)
  const scrollSync = window.onDidChangeTextEditorVisibleRanges((e) => {
    if (
      !previewPanel ||
      e.textEditor.document.languageId !== "calcmark" ||
      e.textEditor.document.uri.toString() !== activePreviewURI
    ) {
      return;
    }
    const topLine = (e.visibleRanges[0]?.start.line ?? 0) + 1; // 0-indexed to 1-indexed
    previewPanel.webview.postMessage({ type: "scrollTo", line: topLine });
  });

  // Don't persist the preview panel across sessions — VS Code's webview
  // restore is unreliable (known issue: blank panels after reload).
  // Instead, dispose any restored panel and let the user reopen it.
  // See: https://github.com/microsoft/vscode/issues/98746
  window.registerWebviewPanelSerializer("calcmarkPreview", {
    async deserializeWebviewPanel(panel: WebviewPanel) {
      panel.dispose();
    },
  });

  context.subscriptions.push(openPreview, editorChange, scrollSync);

  // Find and validate the cm binary, then start the LSP client.
  const cmPath = findBinary();
  if (!cmPath) {
    window.showErrorMessage(
      "CalcMark: Could not find the 'cm' binary. " +
        "Install CalcMark (brew install calcmark/tap/calcmark) " +
        "or set calcmark.binaryPath in settings."
    );
    return;
  }

  const valid = await validateBinary(cmPath);
  if (!valid) {
    return;
  }

  const serverOptions: ServerOptions = {
    command: cmPath,
    args: ["lsp"],
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "calcmark" }],
    outputChannelName: "CalcMark LSP",
    errorHandler: {
      error: () => ({ action: ErrorAction.Continue }),
      closed: () => ({ action: CloseAction.Restart }),
    },
  };

  client = new LanguageClient(
    "calcmark",
    "CalcMark Language Server",
    serverOptions,
    clientOptions
  );

  // Handle live preview notifications
  client.onNotification(
    "calcmark/documentRendered",
    (params: DocumentRenderedParams) => {
      if (!params?.uri || !params?.html) {
        return;
      }
      lastRenderedHTML.set(params.uri, params.html);

      if (!previewPanel) return;

      // If the preview was restored from a previous session, it won't have
      // an activePreviewURI yet. Auto-bind to the active editor's file.
      if (!activePreviewURI) {
        const editor = window.activeTextEditor;
        if (editor?.document.languageId === "calcmark") {
          activePreviewURI = editor.document.uri.toString();
        }
      }

      if (activePreviewURI === params.uri) {
        updatePreview(params.html);
      }
    }
  );

  await client.start();
}

export async function deactivate(): Promise<void> {
  if (client) {
    await client.stop();
  }
}

function findBinary(): string | undefined {
  const config = workspace.getConfiguration("calcmark");
  const configured = config.get<string>("binaryPath");
  if (configured) {
    return configured;
  }
  return "cm";
}

async function validateBinary(cmPath: string): Promise<boolean> {
  try {
    await execFileAsync(cmPath, ["version"]);
    return true;
  } catch {
    window.showErrorMessage(
      `CalcMark: '${cmPath}' does not appear to be a valid cm binary. ` +
        "Update to the latest version or set calcmark.binaryPath in settings."
    );
    return false;
  }
}

function updatePreview(html: string): void {
  if (!previewPanel) return;
  previewPanel.webview.html = getPreviewHTML(html);
}

function getPreviewHTML(html: string): string {
  return `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta http-equiv="Content-Security-Policy"
        content="default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline';">
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      background: var(--vscode-editor-background, white);
      color: var(--vscode-editor-foreground, #333);
      margin: 0;
      padding: 1rem 2rem;
      line-height: 1.6;
    }
    .calc-block {
      margin: 1.5em 0; padding: 1em;
      background: var(--vscode-textBlockQuote-background, #f8f9fa);
      border-left: 4px solid var(--vscode-textLink-foreground, #0066cc);
      border-radius: 4px;
    }
    .calc-line {
      display: flex; justify-content: space-between; align-items: baseline;
      margin: 0.25em 0;
    }
    .calc-source {
      font-family: var(--vscode-editor-font-family, monospace);
      font-size: 0.95em;
      background: none;
      padding: 0;
      border-radius: 0;
    }
    .calc-inline-result {
      font-weight: 600;
      color: var(--vscode-textLink-foreground, #0066cc);
      margin-left: 2em; font-size: 0.9em;
    }
    .calc-inline-result::before { content: "= "; }
    .calc-error {
      color: var(--vscode-errorForeground, #d73a49);
      background: var(--vscode-inputValidation-errorBackground, #ffeef0);
      padding: 0.5em; border-radius: 3px;
      border-left: 3px solid var(--vscode-errorForeground, #d73a49);
      margin-top: 0.5em;
    }
    .text-block { margin: 1.5em 0; }
    .text-block p { margin: 0.75em 0; }
    .text-block h1, .text-block h2, .text-block h3 {
      margin-top: 1.5em; margin-bottom: 0.5em;
    }
    .text-block code {
      background: var(--vscode-textCodeBlock-background, #f6f8fa);
      padding: 0.2em 0.4em; border-radius: 3px;
      font-family: var(--vscode-editor-font-family, monospace); font-size: 0.9em;
    }
    .text-block pre {
      background: var(--vscode-textCodeBlock-background, #f6f8fa);
      padding: 1em; border-radius: 6px; overflow-x: auto;
    }
    .text-block pre code { background: none; padding: 0; }
    .text-block blockquote {
      border-left: 3px solid var(--vscode-textLink-foreground, #0066cc);
      padding-left: 1em; color: var(--vscode-descriptionForeground, #57606a);
      margin: 1em 0;
    }
    .text-block table { border-collapse: collapse; width: 100%; margin: 1em 0; }
    .text-block th, .text-block td {
      border: 1px solid var(--vscode-panel-border, #d0d7de);
      padding: 0.5em 0.75em; text-align: left;
    }
    .text-block th { background: var(--vscode-textBlockQuote-background, #f0f4f8); font-weight: 600; }
    .frontmatter {
      margin-bottom: 2em; padding: 1em 1.5em;
      background: var(--vscode-textBlockQuote-background, #f0f4f8);
      border-radius: 6px; border: 1px solid var(--vscode-panel-border, #d0d7de);
    }
    .frontmatter h3 {
      margin: 0 0 0.75em 0; font-size: 0.9em;
      color: var(--vscode-descriptionForeground, #57606a);
      text-transform: uppercase; letter-spacing: 0.05em;
    }
    .frontmatter dl { margin: 0; display: grid; grid-template-columns: auto 1fr; gap: 0.25em 1em; }
    .frontmatter dt {
      font-family: var(--vscode-editor-font-family, monospace); font-size: 0.9em;
      color: var(--vscode-textLink-foreground, #0550ae);
    }
    .frontmatter dd {
      margin: 0; font-family: var(--vscode-editor-font-family, monospace); font-size: 0.9em;
    }
    .cm-interpolated { font-weight: 600; }
  </style>
</head>
<body>
${html}
<script>
  // Listen for scroll-to messages from the extension
  window.addEventListener('message', (event) => {
    const msg = event.data;
    if (msg.type === 'scrollTo') {
      // Find the closest element at or before the target line
      let best = null;
      document.querySelectorAll('[data-source-line]').forEach((el) => {
        const line = parseInt(el.getAttribute('data-source-line'), 10);
        if (line <= msg.line) {
          best = el;
        }
      });
      if (best) {
        best.scrollIntoView({ block: 'start', behavior: 'smooth' });
      }
    }
  });
</script>
</body>
</html>`;
}
