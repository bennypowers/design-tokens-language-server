extern crate zed_extension_api;
use std::fs;
use zed::LanguageServerId;
use zed_extension_api::{self as zed, Result};

struct DesignTokensExtension {
    cached_binary_path: Option<String>,
}

impl DesignTokensExtension {
    fn language_server_binary(
        &mut self,
        language_server_id: &LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<String> {
        if let Some(path) = worktree.which("design-tokens-language-server") {
            return Ok(path);
        }

        if let Some(path) = &self.cached_binary_path {
            if fs::metadata(path).map_or(false, |stat| stat.is_file()) {
                return Ok(path.clone());
            }
        }

        zed::set_language_server_installation_status(
            language_server_id,
            &zed::LanguageServerInstallationStatus::CheckingForUpdate,
        );
        let release = zed::latest_github_release(
            "bennypowers/design-tokens-language-server",
            zed::GithubReleaseOptions {
                require_assets: true,
                pre_release: false,
            },
        )?;

        let (platform, arch) = zed::current_platform();
        // Binary names for the design tokens language server:
        //  * - design-tokens-language-server-aarch64-apple-darwin
        //  * - design-tokens-language-server-aarch64-unknown-linux-gnu
        //  * - design-tokens-language-server-x86_64-apple-darwin
        //  * - design-tokens-language-server-x86_64-unknown-linux-gnu
        //  * - design-tokens-language-server-win-x64.exe
        //  * - design-tokens-language-server-win-arm64.exe
        let asset_name = match platform {
            // Windows uses simplified naming
            zed::Os::Windows => {
                let arch_name = match arch {
                    zed::Architecture::Aarch64 => "arm64",
                    zed::Architecture::X8664 => "x64",
                    zed::Architecture::X86 => todo!(),
                };
                format!("design-tokens-language-server-win-{}.exe", arch_name)
            }
            // Unix platforms use target triples
            _ => {
                let arch_name = match arch {
                    zed::Architecture::Aarch64 => "aarch64",
                    zed::Architecture::X8664 => "x86_64",
                    zed::Architecture::X86 => todo!(),
                };
                let os_name = match platform {
                    zed::Os::Mac => "apple-darwin",
                    zed::Os::Linux => "unknown-linux-gnu",
                    zed::Os::Windows => unreachable!(),
                };
                format!("design-tokens-language-server-{}-{}", arch_name, os_name)
            }
        };

        let asset = release
            .assets
            .iter()
            .find(|asset| asset.name == asset_name)
            .ok_or_else(|| format!("no asset found matching {:?}", asset_name))?;

        let version_dir = format!("design-tokens-language-server-{}", release.version);
        fs::create_dir_all(&version_dir)
            .map_err(|err| format!("failed to create directory '{version_dir}': {err}"))?;

        let binary_path = format!("{version_dir}/{asset_name}");

        if !fs::metadata(&binary_path).map_or(false, |stat| stat.is_file()) {
            zed::set_language_server_installation_status(
                language_server_id,
                &zed::LanguageServerInstallationStatus::Downloading,
            );

            zed::download_file(
                &asset.download_url,
                &binary_path,
                zed::DownloadedFileType::Uncompressed,
            )
            .map_err(|err| format!("failed to download file: {err}"))?;

            zed::make_file_executable(&binary_path)?;

            let entries = fs::read_dir(".")
                .map_err(|err| format!("failed to list working directory {err}"))?;
            for entry in entries {
                let entry = entry.map_err(|err| format!("failed to load directory entry {err}"))?;
                if entry.file_name().to_str() != Some(&version_dir) {
                    fs::remove_dir_all(entry.path()).ok();
                }
            }
        }

        self.cached_binary_path = Some(binary_path.clone());
        Ok(binary_path)
    }
}

impl zed::Extension for DesignTokensExtension {
    fn new() -> Self {
        Self {
            cached_binary_path: None,
        }
    }

    fn language_server_command(
        &mut self,
        language_server_id: &LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<zed::Command> {
        let dtls_binary = self.language_server_binary(language_server_id, worktree)?;
        // wrap the dtls_binary in `lsp-devtools agent -- ${dtls_binary}`
        // let command = format!("{} agent -- {}", worktree.which("lsp-devtools").unwrap()dtls_binary);
        Ok(zed::Command {
            command: dtls_binary,
            args: vec![],
            env: Default::default(),
        })
    }
}

zed::register_extension!(DesignTokensExtension);
