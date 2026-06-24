# Gemini CLI Provider Plugin

This plugin adds Gemini CLI upstream provider support to CLIProxyAPI through the native plugin ABI. It does not restore the host-side `/v1internal:generateContent` or `/v1internal:streamGenerateContent` inbound routes; those endpoints are used only as upstream Cloud Code API targets.

## Capabilities

- Parses `gemini` and `gemini-cli` auth storage files.
- Expands one physical auth file into multiple virtual auths when `project_ids` contains more than one Google Cloud project.
- Supports host Web OAuth through the host `/v0/management/oauth-callback` flow.
- Supports command-line login through `--geminicli-login`.
- Executes generate, stream, and token count requests through the host HTTP client when the host invokes the executor.
- Translates OpenAI, Responses, Claude, Gemini, and Codex payloads to the Gemini CLI provider envelope.
- Applies Gemini CLI thinking config under `request.generationConfig.thinkingConfig`.

## Command-Line Flags

- `--geminicli-login`: starts an interactive Gemini CLI login.
- `--geminicli-project-id`: sets the preferred project for the saved auth.

Use the host `--no-browser` flag to skip opening the browser automatically during command-line login. OAuth login saves a single physical auth file; multiple projects are kept in `project_ids` and expanded as virtual auths when the file is loaded. OAuth login timeout and polling interval are fixed in code. Proxy handling comes from the host configuration; the plugin has no plugin-specific proxy option.

## Auth Storage

The plugin stores provider-owned auth JSON with `type: "gemini-cli"`. A single physical storage file can include:

```json
{
  "type": "gemini-cli",
  "email": "user@example.com",
  "project_id": "primary-project",
  "project_ids": ["primary-project", "secondary-project"],
  "access_token": "access-token",
  "refresh_token": "refresh-token"
}
```

The first auth is the physical auth. Additional projects are exposed as virtual auths with `metadata.virtual=true`, `metadata.parent_auth_id`, `attributes.project_id`, and `attributes.runtime_only=true`.

## Upstream Endpoints

The executor targets the Cloud Code upstream endpoints:

- `POST https://cloudcode-pa.googleapis.com/v1internal:generateContent`
- `POST https://cloudcode-pa.googleapis.com/v1internal:streamGenerateContent?alt=sse`
- `POST https://cloudcode-pa.googleapis.com/v1internal:countTokens`

The plugin injects `Authorization`, `User-Agent`, and `X-Goog-Api-Client` headers before dispatching through the host HTTP client.

## License

This project is licensed under the [MIT License](LICENSE).

Copyright (c) 2026.6-present Router-For.ME
