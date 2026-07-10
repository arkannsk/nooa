{{- /* method.go.tpl — template for a single HTTP method */ -}}

{{- $hasParams := false }}
{{- if .QueryParams }}{{ $hasParams = true }}{{ end }}
{{- if .HeaderParams }}{{ $hasParams = true }}{{ end }}
{{- if .RequestBody }}{{ $hasParams = true }}{{ end }}
{{- if .PathParams }}{{ $hasParams = true }}{{ end }}

{{- $hasResponse := false }}
{{- if .ResponseBody }}{{ $hasResponse = true }}{{ end }}

{{- define "retPrefix" -}}
{{- if . }}nil, {{ end -}}
{{- end -}}

{{- if $hasParams }}

// {{ .MethodName }}Request holds parameters for {{ .MethodName }}.
type {{ .MethodName }}Request struct {
{{- if .RequestBody }}
	// Body is the request body.
	Body {{ .RequestBody.GoType }}
{{- end }}
{{- range .QueryParams }}
	// {{ .Name }} — {{ .Description }}
{{- if .Required }}
	{{ toGoName .Name }} {{ .GoType }}
{{- else }}
	{{ toGoName .Name }} {{ .GoType }}
	{{ toGoName .Name }}Set bool
{{- end }}
{{- end }}
{{- range .HeaderParams }}
	// {{ .Name }} — {{ .Description }}
{{- if .Required }}
	{{ toGoName .Name }} {{ .GoType }}
{{- else }}
	{{ toGoName .Name }} {{ .GoType }}
	{{ toGoName .Name }}Set bool
{{- end }}
{{- end }}
{{- range .PathParams }}
	// {{ .Name }} — {{ .Description }} (path parameter)
	{{ toGoName .Name }} {{ .GoType }}
{{- end }}
}

{{- end }}

{{- if $hasResponse }}

// {{ .MethodName }}Response holds the response for {{ .MethodName }}.
type {{ .MethodName }}Response struct {
{{- range $code, $field := .ResponseBody.Fields }}
	// Status {{ $code }} — {{ $field.SchemaName }}
	Status{{ $code }} {{ $field.GoType }}
{{- end }}
	// StatusCode is the HTTP status code.
	StatusCode int
	// RawBody is the raw response body.
	RawBody []byte
}

{{- end }}

// {{ .MethodName }} — {{ .Summary }}
func (c *Client) {{ .MethodName }}(ctx context.Context{{ if $hasParams }}, input *{{ .MethodName }}Request{{ end }}) {{ if $hasResponse }}(*{{ .MethodName }}Response, error){{ else }}error{{ end }} {
{{- if .PathParams }}

	{{- range .PathParams }}
	if input == nil {
		return {{ template "retPrefix" $hasResponse }}fmt.Errorf("{{ .Name }} is required")
	}
	{{- end }}
{{- end }}

	u := c.BaseURL + {{ .Path | printf "%q" }}
{{- range .PathParams }}

	{{- $pname := toGoName .Name }}
	u = strings.ReplaceAll(u, "{{ printf "{%s}" .Name }}", fmt.Sprintf("%v", input.{{ $pname }}))
{{- end }}

{{- if .QueryParams }}

	// Query parameters
	query := url.Values{}
	{{- range .QueryParams }}

	{{- $pname := toGoName .Name }}
	if input != nil && input.{{ $pname }}Set {
		query.Set("{{ .Name }}", fmt.Sprintf("%v", input.{{ $pname }}))
	}
	{{- end }}
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
{{- end }}

	var body io.Reader
{{- if .RequestBody }}

	if input != nil {
		b, err := json.Marshal(input.Body)
		if err != nil {
			return {{ template "retPrefix" $hasResponse }}fmt.Errorf("marshal request body: %w", err)
		}
		body = bytes.NewReader(b)
	}
{{- end }}

	httpReq, err := http.NewRequestWithContext(ctx, "{{ .Method }}", u, body)
	if err != nil {
		return {{ template "retPrefix" $hasResponse }}fmt.Errorf("create request: %w", err)
	}
{{- if .RequestBody }}
	httpReq.Header.Set("Content-Type", "application/json")
{{- end }}
{{- if .HeaderParams }}

	{{- range .HeaderParams }}

	{{- $pname := toGoName .Name }}
	if input != nil && input.{{ $pname }}Set {
		httpReq.Header.Set("{{ .Name }}", fmt.Sprintf("%v", input.{{ $pname }}))
	}
	{{- end }}
{{- end }}
	if c.Token != "" {
		httpReq.Header.Set("Authorization", c.TokenPrefix+" "+c.Token)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return {{ template "retPrefix" $hasResponse }}fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return {{ template "retPrefix" $hasResponse }}fmt.Errorf("read response: %w", err)
	}

{{- if $hasResponse }}

	result := &{{ .MethodName }}Response{
		StatusCode: resp.StatusCode,
		RawBody:    raw,
	}
	{{- range $code, $field := .ResponseBody.Fields }}
	if resp.StatusCode == {{ $code }} {
		if err := json.Unmarshal(raw, &result.Status{{ $code }}); err != nil {
			return result, fmt.Errorf("unmarshal status {{ $code }}: %w", err)
		}
	}
	{{- end }}
	if resp.StatusCode >= 400 {
		return result, fmt.Errorf("request failed: status %d", resp.StatusCode)
	}
	return result, nil
{{- else }}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("request failed: status %d body: %s", resp.StatusCode, string(raw))
	}
	return nil
{{- end }}
}
