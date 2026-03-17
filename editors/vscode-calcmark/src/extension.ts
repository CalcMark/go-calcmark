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
        { enableScripts: false, retainContextWhenHidden: true }
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

  // Scroll sync: editor-to-preview (unidirectional)
  const scrollSync = window.onDidChangeTextEditorVisibleRanges((e) => {
    if (
      !previewPanel ||
      e.textEditor.document.languageId !== "calcmark" ||
      e.textEditor.document.uri.toString() !== activePreviewURI
    ) {
      return;
    }
    const topLine = e.visibleRanges[0]?.start.line ?? 0;
    const html = lastRenderedHTML.get(activePreviewURI!);
    if (html) {
      updatePreviewWithAnchor(html, topLine + 1); // Convert 0-indexed to 1-indexed
    }
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

function updatePreviewWithAnchor(html: string, sourceLine: number): void {
  if (!previewPanel) return;
  // Add an anchor id to the element closest to the source line
  const anchored = html.replace(
    new RegExp(`data-source-line="${sourceLine}"`),
    `data-source-line="${sourceLine}" id="scroll-target"`
  );
  previewPanel.webview.html = getPreviewHTML(anchored, true);
}

function getPreviewHTML(html: string, hasAnchor = false): string {
  const scrollScript = hasAnchor
    ? '<script>document.getElementById("scroll-target")?.scrollIntoView({block:"start"});</script>'
    : "";
  return `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta http-equiv="Content-Security-Policy"
        content="default-src 'none'; style-src 'unsafe-inline'${hasAnchor ? "; script-src 'unsafe-inline'" : ""};">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; padding: 1rem; }
  </style>
</head>
<body>${html}${scrollScript}</body>
</html>`;
}
