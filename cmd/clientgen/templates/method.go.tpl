{{- /* method.go.tpl — template for a single HTTP method */ -}}

{{- $hasParams := false }}
{{- if .QueryParams }}{{ $hasParams = true }}{{ end }}
{{- if .HeaderParams }}{{ $hasParams = true }}{{ end }}
{{- if .PathParams }}{{ $hasParams = true }}{{ end }}

{{- $hasResponse := false }}
{{- if .ResponseBody }}{{ $hasResponse = true }}{{ end }}

{{- $usesModel := .UsesModelAsInput }}
{{- $inputGoType := .ModelInputGoType }}

{{- if $hasResponse }}

// {{ .MethodName }}Response holds the response for {{ .MethodName }}.
type {{ .MethodName }}Response struct {
	*http.Response
	client *Client
{{- range $code, $field := .ResponseBody.Fields }}
	status{{ $code }} *{{ $field.GoType }}
{{- end }}
}

{{- range $code, $field := .ResponseBody.Fields }}

func (r *{{ $.MethodName }}Response) Status{{ toGoTypeName $code }}() (*{{ $field.GoType }}, error) {
	if r.StatusCode != {{ $code }} {
		return nil, fmt.Errorf("expected status {{ $code }}, got %d", r.StatusCode)
	}
	if r.status{{ $code }} != nil {
		return r.status{{ $code }}, nil
	}
	r.status{{ $code }} = new({{ $field.GoType }})
	if err := client.UnmarshalResponse(r.Response, r.client.Codec, r.status{{ $code }}); err != nil {
		return nil, err
	}
	return r.status{{ $code }}, nil
}

{{- end }}
{{- end }}

{{- if and $hasParams (not $usesModel) }}

// {{ .MethodName }}Request holds parameters for {{ .MethodName }}.
type {{ .MethodName }}Request struct {
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

// {{ .MethodName }} — {{ .Summary }}
// {{ .Method }} {{ .Path }}
func (c *Client) {{ .MethodName }}(ctx context.Context{{ if $usesModel }}, input *{{ $inputGoType }}{{ else if $hasParams }}, input *{{ .MethodName }}Request{{ else }}, opts ...client.RequestOption{{ end }}) {{ if $hasResponse }}(*{{ .MethodName }}Response, error){{ else }}error{{ end }} {
{{- if and $hasParams .PathParams }}

	if input == nil {
		return {{ if $hasResponse }}nil, {{ end }}fmt.Errorf("input is required")
	}
{{- end }}

	u := c.BaseURL + {{ .Path | printf "%q" }}
{{- if or $hasParams $usesModel }}

{{ range .PathParams }}

	{{ if $usesModel }}
	u = strings.ReplaceAll(u, "{{ printf "{%s}" .Name }}", fmt.Sprintf("%v", input.{{ .GoFieldName }}))
	{{ else }}
	{{ $pname := toGoName .Name }}
	u = strings.ReplaceAll(u, "{{ printf "{%s}" .Name }}", fmt.Sprintf("%v", input.{{ $pname }}))
	{{ end }}
{{ end }}

{{- if .QueryParams }}
	// Query parameters
	query := url.Values{}
	{{- range .QueryParams }}

	{{- if $usesModel }}
	query.Set("{{ .Name }}", fmt.Sprintf("%v", input.{{ .GoFieldName }}))
	{{- else }}
	{{- $pname := toGoName .Name }}
	if input.{{ $pname }}Set {
		query.Set("{{ .Name }}", fmt.Sprintf("%v", input.{{ $pname }}))
	}
	{{- end }}
	{{- end }}
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
{{- end }}
{{- end }}

	var body io.Reader
{{- if .RequestBody }}

{{- if $usesModel }}
	b, err := json.Marshal(input)
	if err != nil {
		return {{ if $hasResponse }}nil, {{ end }}fmt.Errorf("marshal request body: %w", err)
	}
	body = bytes.NewReader(b)
{{- else }}
	if input != nil {
		b, err := json.Marshal(input)
		if err != nil {
			return {{ if $hasResponse }}nil, {{ end }}fmt.Errorf("marshal request body: %w", err)
		}
		body = bytes.NewReader(b)
	}
{{- end }}
{{- end }}

	httpReq, err := http.NewRequestWithContext(ctx, "{{ .Method }}", u, body)
	if err != nil {
		return {{ if $hasResponse }}nil, {{ end }}fmt.Errorf("create request: %w", err)
	}
{{- if .RequestBody }}
	httpReq.Header.Set("Content-Type", "application/json")
{{- end }}
{{- if $usesModel }}
{{- if .HeaderParams }}

	{{- range .HeaderParams }}

	httpReq.Header.Set("{{ .Name }}", fmt.Sprintf("%v", input.{{ .GoFieldName }}))
	{{- end }}
{{- end }}
{{- else if $hasParams }}
{{- if .HeaderParams }}

	{{- range .HeaderParams }}

	{{- $pname := toGoName .Name }}
	if input.{{ $pname }}Set {
		httpReq.Header.Set("{{ .Name }}", fmt.Sprintf("%v", input.{{ $pname }}))
	}
	{{- end }}
{{- end }}
{{- else }}
	// Apply options
	for _, opt := range opts {
		opt(httpReq)
	}
{{- end }}

	resp, err := c.HTTPClient.Do(ctx, httpReq)
	if err != nil {
		return {{ if $hasResponse }}nil, {{ end }}fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(resp.Body)
		return {{ if $hasResponse }}nil, {{ end }}fmt.Errorf("request failed: status %d, body: %s", resp.StatusCode, string(rb))
	}

{{- if $hasResponse }}
	return &{{ .MethodName }}Response{
		Response: resp,
		client:   c,
	}, nil
{{- else }}
	return nil
{{- end }}
}
