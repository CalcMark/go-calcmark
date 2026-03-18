# Releasing the VS Code Extension

**Marketplace:** [marketplace.visualstudio.com/items?itemName=calcmark.vscode-calcmark](https://marketplace.visualstudio.com/items?itemName=calcmark.vscode-calcmark)
**Publisher management:** [marketplace.visualstudio.com/manage/publishers/calcmark](https://marketplace.visualstudio.com/manage/publishers/calcmark)

## One-Time Setup

### 1. Create an Azure DevOps organization

Go to [dev.azure.com](https://dev.azure.com) and sign in with a Microsoft account. Create an organization if you don't have one.

### 2. Create a Personal Access Token (PAT)

1. In Azure DevOps, click your profile icon → **Personal access tokens**
2. Click **New Token**
3. Set:
   - **Name:** `vsce-publish`
   - **Organization:** All accessible organizations
   - **Scopes:** Custom defined → **Marketplace** → check **Manage**
   - **Expiration:** 1 year (maximum)
4. Click **Create** and copy the token immediately

### 3. Create a publisher

1. Go to [marketplace.visualstudio.com/manage](https://marketplace.visualstudio.com/manage)
2. Create a publisher with ID `calcmark` (must match `publisher` in package.json)

### 4. Store the token

For local publishing:

```bash
cd editors/vscode-calcmark
npx vsce login calcmark
# Paste your PAT when prompted
```

For GitHub Actions, add the PAT as a repository secret named `VSCE_PAT`:

1. Go to the repo → **Settings** → **Secrets and variables** → **Actions**
2. Click **New repository secret**
3. Name: `VSCE_PAT`, Value: your PAT

## Publishing

### Option A: Manual upload (recommended for now)

Package the extension and upload via the web:

```bash
cd editors/vscode-calcmark
npm install
npm run compile
npx vsce package
```

Then upload at [marketplace.visualstudio.com/manage/publishers/calcmark](https://marketplace.visualstudio.com/manage/publishers/calcmark):

1. Click your extension (or **New Extension** → **Visual Studio Code** for the first time)
2. Click **Update** (or drag the `.vsix` for a new extension)
3. Wait for validation to complete (~5 minutes)

**Important:** When creating a new extension, select **Visual Studio Code** — not "Visual Studio" or "Azure DevOps". Selecting the wrong type produces a confusing "Value cannot be null. Parameter name: v1" error.

### Option B: CLI publish (requires working PAT auth)

```bash
task publish:vscode        # stable
task publish:vscode:pre    # pre-release
```

### Option C: GitHub Actions (requires working PAT auth)

The workflow at `.github/workflows/vscode-extension.yml` publishes automatically:

- **Pre-release** — triggered by pushing a tag like `vscode-v0.1.0-pre.1`
- **Stable release** — triggered by pushing a tag like `vscode-v0.1.0`

```bash
# Pre-release
git tag vscode-v0.1.0-pre.1
git push origin vscode-v0.1.0-pre.1

# Stable
git tag vscode-v0.1.0
git push origin vscode-v0.1.0
```

**Note:** CI publishing via `VSCE_PAT` secret is not yet working (TF400813 auth error under investigation). Use Option A until resolved.

## Versioning

The extension version in `package.json` is independent of the `cm` binary version. Bump it before each publish:

```bash
cd editors/vscode-calcmark
npm version patch   # 0.1.0 → 0.1.1
npm version minor   # 0.1.1 → 0.2.0
npm version major   # 0.2.0 → 1.0.0
```

For pre-releases, the marketplace uses the same version number but marks it as pre-release via the `--pre-release` flag — no special version suffix needed.

## Packaging without publishing

To create a `.vsix` file for manual installation or testing:

```bash
cd editors/vscode-calcmark
npx vsce package              # stable
npx vsce package --pre-release  # pre-release
```

Install the `.vsix` in VS Code: **Cmd+Shift+P** → **Extensions: Install from VSIX...**
