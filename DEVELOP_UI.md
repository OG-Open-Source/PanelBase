# Optional UI Settings

The following UI settings can be configured in the `configs/ui_settings.json` file or updated via the `PUT /api/v1/settings/ui` API endpoint. These settings are dynamically loaded by the HTML templates in the `web/` directory.

- **`title`**: (String, Required) The website title displayed in the browser tab and page heading.
  - Default: "PanelBase"
  - Example: `"title": "My Awesome Control Panel"`
- **`logo_url`**: (String, Optional) The URL path to the logo image displayed in the page header (or other designated areas).
  - If omitted or empty, the title text might be displayed as a fallback.
  - Example: `"logo_url": "/assets/images/logo.png"`
- **`favicon_url`**: (String, Optional) The URL path to the image used as the browser tab icon (favicon).
  - Example: `"favicon_url": "/assets/images/favicon.ico"`
- **`custom_css`**: (String, Optional) Custom CSS rules to be injected into a `<style>` tag within the HTML `<head>` section.
  - Can be used to override default styles or add custom themes.
  - Example: `"custom_css": "body { font-family: 'Arial', sans-serif; } h1 { color: blue; }"`
- **`custom_js`**: (String, Optional) Custom JavaScript code to be injected into a `<script>` tag at the bottom of the HTML `<body>`.
  - Can be used to add custom client-side behavior or integrate third-party scripts.
  - Example: `"custom_js": "console.log('Custom script loaded!'); alert('Welcome!');"`

**Template Integration:**

Within HTML files in `web/` (e.g., `web/index.html`), these settings can be accessed using Go template syntax, for example:

```html
<title>{{ .Title }}</title>
{{ if .FaviconURL }}<link rel="icon" href="{{ .FaviconURL }}">{{ end }}
{{ if .LogoURL }}<img src="{{ .LogoURL }}" alt="Logo">{{ else }}<h1>{{ .Title }}</h1>{{ end }}
{{ if .CustomCSS }}<style>{{ .CustomCSS }}</style>{{ end }}
{{ if .CustomJS }}<script>{{ .CustomJS }}</script>{{ end }}
```
