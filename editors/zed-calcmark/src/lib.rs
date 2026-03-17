use zed_extension_api as zed;

struct CalcMarkExtension;

impl zed::Extension for CalcMarkExtension {
    fn new() -> Self {
        CalcMarkExtension
    }

    fn language_server_command(
        &mut self,
        _language_server_id: &zed::LanguageServerId,
        worktree: &zed::Worktree,
    ) -> zed::Result<zed::Command> {
        // Look for cm binary in PATH via the worktree's which()
        let cm_path = worktree
            .which("cm")
            .ok_or_else(|| "cm binary not found in PATH. Install CalcMark: https://github.com/CalcMark/go-calcmark".to_string())?;

        Ok(zed::Command {
            command: cm_path,
            args: vec!["lsp".to_string()],
            env: vec![],
        })
    }
}

zed::register_extension!(CalcMarkExtension);
