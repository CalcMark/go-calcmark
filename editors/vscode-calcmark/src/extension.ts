import { ExtensionContext, workspace, window } from "vscode";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

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

  const serverOptions: ServerOptions = {
    command: cmPath,
    args: ["lsp"],
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "calcmark" }],
    outputChannelName: "CalcMark LSP",
  };

  client = new LanguageClient(
    "calcmark",
    "CalcMark Language Server",
    serverOptions,
    clientOptions
  );

  await client.start();
}

export async function deactivate(): Promise<void> {
  if (client) {
    await client.stop();
  }
}

function findBinary(): string | undefined {
  // 1. Check user setting
  const config = workspace.getConfiguration("calcmark");
  const configured = config.get<string>("binaryPath");
  if (configured) {
    return configured;
  }

  // 2. Use "cm" from PATH
  return "cm";
}
