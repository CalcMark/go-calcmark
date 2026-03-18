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
  const cmPath = findBinary();
  if (!cmPath) {
    window.showErrorMessage(
      "CalcMark: Could not find the 'cm' binary. " +
        "Install CalcMark (brew install calcmark/tap/calcmark) " +
        "or set calcmark.binaryPath in settings."
    );
    return;
  }

  // Validate binary supports LSP
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

      if (previewPanel && activePreviewURI === params.uri) {
        updatePreview(params.html);
      }
    }
  );

  // Register preview command
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

  context.subscriptions.push(openPreview, editorChange, scrollSync);

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
    /* Override the CalcMark template styles for webview context */
    body {
      background: white;
      margin: 0;
      padding: 1rem 2rem;
    }
    .calc-block {
      background: #f8f9fa;
      border-radius: 4px;
    }
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
