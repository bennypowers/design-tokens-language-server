extern crate zed_extension_api;
use std::path::Path;
use zed_extension_api::{self as zed, LanguageServerId};

struct DesignTokensLanguageserverExtension {
    // ... state
}

impl zed::Extension for DesignTokensLanguageserverExtension {
    fn new() -> Self {
        Self {}
    }

    fn language_server_command(
        &mut self,
        language_server_id: &LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<zed::Command, std::string::String> {
        Ok(zed::Command {
            // command: "/var/home/bennyp/.local/bin/design-tokens-language-server".to_owned(),
            command: Path::new(worktree.root_path().as_str())
                .join("bin/design-tokens-language-server")
                .into_os_string()
                .into_string()
                .unwrap(),
            args: [].to_vec(),
            env: Default::default(),
        })
    }
}

zed::register_extension!(DesignTokensLanguageserverExtension);
