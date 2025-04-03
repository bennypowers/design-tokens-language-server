extern crate zed_extension_api;
use std::env;
use std::fs;
use std::path::Path;

use zed_extension_api::{self as zed, LanguageServerId};

struct DesignTokensLanguageserverExtension {
    cached_binary_path: Option<String>,
}

impl DesignTokensLanguageserverExtension {
    fn language_server_binary_path(
        &mut self,
        _id: &LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<String, String> {
        if let Some(path) = Some("/var/home/bennyp/.local/bin/design-tokens-language-server") {
            return Ok(path.to_string());
        }

        if let Some(path) = &self.cached_binary_path {
            if fs::metadata(path).map_or(false, |stat| stat.is_file()) {
                return Ok(path.clone());
            }
        }

        let result = self.copy_bin(worktree);
        match result {
            Ok(path) => {
                self.cached_binary_path = Some(path.clone());
                return Ok(path);
            }
            Err(err) => Err(err.to_string()),
        }
    }

    fn copy_bin(&mut self, worktree: &zed::Worktree) -> Result<String, std::io::Error> {
        let root_path = worktree.root_path();

        let home = env::var("XDG_STATE_HOME").unwrap_or("/var/home/bennyp/.local".to_string());

        let repo_path = Path::new("").join(&home).to_string_lossy().to_string();

        let binary_path = Path::new("")
            .join(root_path)
            .join("node_modules/.bin/design-tokens-language-server")
            .to_string_lossy()
            .to_string();

        let result = fs::copy(repo_path, binary_path.clone());

        match result {
            Ok(_u64) => Ok(binary_path.clone()),
            Err(e) => Err(e),
        }
    }
}

impl zed::Extension for DesignTokensLanguageserverExtension {
    fn new() -> Self {
        Self {
            cached_binary_path: None,
        }
    }

    fn language_server_command(
        &mut self,
        id: &LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<zed::Command, std::string::String> {
        let command = self.language_server_binary_path(id, worktree);
        match command {
            Ok(command) => Ok(zed::Command {
                command: command.to_string(),
                args: [].to_vec(),
                env: Default::default(),
            }),
            Err(err) => Err(err),
        }
    }
}

zed::register_extension!(DesignTokensLanguageserverExtension);
