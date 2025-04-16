extern crate zed_extension_api;
use std::collections::HashMap;
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
        if let Some(path) = Some(&self.get_local_bin_path(worktree)) {
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

    fn get_local_bin_path(&mut self, worktree: &zed::Worktree) -> String {
        let env = worktree
            .shell_env()
            .into_iter()
            .map(|data| (data.0, data.1))
            .collect::<HashMap<String, String>>();

        let home = match env.get("HOME") {
            Some(h) => h,
            None => {
                panic!("No HOME env var")
            }
        };

        let state_home = match env.get("XDG_STATE_HOME") {
            Some(h) => h,
            None => &Path::new(&home)
                .join(".local")
                .to_string_lossy()
                .to_string(),
        };

        return Path::new(state_home)
            .join("bin")
            .join("design-tokens-language-server")
            .to_string_lossy()
            .to_string();
    }

    fn copy_bin(&mut self, worktree: &zed::Worktree) -> Result<String, std::io::Error> {
        let binary_path = Path::new(&worktree.root_path())
            .join("node_modules/.bin/design-tokens-language-server")
            .to_string_lossy()
            .to_string();

        let local_bin_path = self.get_local_bin_path(worktree);

        let result = fs::copy(&local_bin_path, binary_path.clone());

        match result {
            Ok(_u64) => Ok(binary_path),
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
