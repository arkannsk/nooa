{{- /* method.go.tpl — template for a single HTTP method */ -}}

{{- $hasParams := false }}
{{- if .QueryParams }}{{ $hasParams = true }}{{ end }}
{{- if .HeaderParams }}{{ $hasParams = true }}{{ end }}
{{- if .RequestBody }}{{ $hasParams = true }}{{ end }}
{{- if .PathParams }}{{ $hasParams = true }}{{ end }}

{{- $hasResponse := false }}
{{- if .ResponseBody }}{{ $hasResponse = true }}{{ end }}

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

// {{ .MethodName }} — {{ .Summary }}
func (c *Client) {{ .MethodName }}(ctx context.Context{{ if $hasParams }}, input *{{ .MethodName }}Request{{ end }}) {{ if $hasResponse }}(*{{ .MethodName }}Response, error){{ else }}error{{ end }} {
{{- if .PathParams }}

	if input == nil {
		return {{ if $hasResponse }}nil, {{ end }}fmt.Errorf("input is required")
	}
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
			return {{ if $hasResponse }}nil, {{ end }}fmt.Errorf("marshal request body: %w", err)
		}
		body = bytes.NewReader(b)
	}
{{- end }}

	httpReq, err := http.NewRequestWithContext(ctx, "{{ .Method }}", u, body)
	if err != nil {
		return {{ if $hasResponse }}nil, {{ end }}fmt.Errorf("create request: %w", err)
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

	resp, err := c.HTTPClient.Do(ctx, httpReq)
	if err != nil {
		return {{ if $hasResponse }}nil, {{ end }}fmt.Errorf("do request: %w", err)
	}

{{- if $hasResponse }}

	result := &{{ .MethodName }}Response{
		Response: resp,
		client:   c,
	}
	if resp.StatusCode >= 400 {
		return result, fmt.Errorf("request failed: status %d", resp.StatusCode)
	}
	return result, nil
{{- else }}

	if resp.StatusCode >= 400 {
		_ = resp.Body.Close()
		return fmt.Errorf("request failed: status %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
	return nil
{{- end }}
}
