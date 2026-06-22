package nooa

// SecuritySchemeIn — где размещён API key.
type SecuritySchemeIn int

const (
	SecurityInHeader SecuritySchemeIn = iota
	SecurityInQuery
	SecurityInCookie
)

// OAuth2Flow — тип OAuth2 flow.
type OAuth2Flow int

const (
	OAuth2FlowImplicit OAuth2Flow = iota
	OAuth2FlowPassword
	OAuth2FlowClientCredentials
	OAuth2FlowAuthorizationCode
)

// OAuth2Scope — одна запись scope для OAuth2.
type OAuth2Scope struct {
	Name        string
	Description string
}

// OAuth2Config — конфигурация OAuth2-схемы безопасности.
//
//	// Authorization Code flow
//	nooa.OAuth2Config{
//	    Flow:     nooa.OAuth2FlowAuthorizationCode,
//	    AuthURL:  "https://auth.example.com/authorize",
//	    TokenURL: "https://auth.example.com/token",
//	    Scopes: []nooa.OAuth2Scope{
//	        {Name: "read", Description: "Read access"},
//	        {Name: "write", Description: "Write access"},
//	    },
//	}
type OAuth2Config struct {
	Flow     OAuth2Flow
	AuthURL  string
	TokenURL string
	Scopes   []OAuth2Scope
}

// SecurityScheme — описание схемы безопасности для OpenAPI spec.
// Регистрируется в components/securitySchemes.
//
// Примеры:
//
//	// Bearer token (JWT)
//	spec.AddSecurityScheme("bearerAuth", nooa.SecuritySchemeBearer("JWT authorization"))
//
//	// API key в заголовке
//	spec.AddSecurityScheme("apiKey", nooa.SecuritySchemeAPIKey("X-API-Key", nooa.SecurityInHeader, "API key"))
//
//	// OAuth2 authorizationCode
//	spec.AddSecurityScheme("oauth2", nooa.SecuritySchemeOAuth2("oauth2", "OAuth2", nooa.OAuth2Config{
//	    Flow:     nooa.OAuth2FlowAuthorizationCode,
//	    AuthURL:  "https://auth.example.com/authorize",
//	    TokenURL: "https://auth.example.com/token",
//	    Scopes: []nooa.OAuth2Scope{
//	        {Name: "read", Description: "Read access"},
//	        {Name: "write", Description: "Write access"},
//	    },
//	}))
type SecurityScheme struct {
	// Name — ключ в components/securitySchemes.
	Name string
	// Description — описание схемы.
	Description string

	// --- HTTP / HTTP Bearer ---
	HTTPScheme string // "basic" или "bearer"

	// --- API Key ---
	APIKeyIn    SecuritySchemeIn
	APIKeyParam string // имя заголовка или query-параметра

	// --- OAuth2 ---
	OAuth2 OAuth2Config

	// --- OpenID Connect ---
	OpenIDConnectURL string
}

// securitySchemeKind определяет, какой вид схемы передан.
type securitySchemeKind int

const (
	kindHTTPBearer securitySchemeKind = iota
	kindHTTPBasic
	kindAPIKey
	kindOAuth2
	kindOpenIDConnect
)

func (s SecurityScheme) kind() securitySchemeKind {
	if s.OpenIDConnectURL != "" {
		return kindOpenIDConnect
	}
	if len(s.OAuth2.Scopes) > 0 || s.OAuth2.AuthURL != "" || s.OAuth2.TokenURL != "" {
		return kindOAuth2
	}
	if s.APIKeyParam != "" {
		return kindAPIKey
	}
	if s.HTTPScheme == "bearer" {
		return kindHTTPBearer
	}
	if s.HTTPScheme == "basic" {
		return kindHTTPBasic
	}
	return kindHTTPBearer // default
}

// SecuritySchemeBearer создаёт схему Bearer-аутентификации (HTTP Bearer).
func SecuritySchemeBearer(description string) SecurityScheme {
	return SecurityScheme{
		Name:        "bearerAuth",
		Description: description,
		HTTPScheme:  "bearer",
	}
}

// SecuritySchemeBasic создаёт схему Basic-аутентификации.
func SecuritySchemeBasic(description string) SecurityScheme {
	return SecurityScheme{
		Name:        "basicAuth",
		Description: description,
		HTTPScheme:  "basic",
	}
}

// SecuritySchemeAPIKey создаёт схему API key.
// paramName — имя заголовка или query-параметра.
// in — SecurityInHeader, SecurityInQuery или SecurityInCookie.
func SecuritySchemeAPIKey(paramName string, in SecuritySchemeIn, description string) SecurityScheme {
	return SecurityScheme{
		Name:        "apiKey",
		Description: description,
		APIKeyIn:    in,
		APIKeyParam: paramName,
	}
}

// SecuritySchemeOAuth2 создаёт схему OAuth2.
// name — ключ в securitySchemes.
// description — описание.
// cfg — конфигурация OAuth2 (flow, URL, scopes).
func SecuritySchemeOAuth2(name, description string, cfg OAuth2Config) SecurityScheme {
	return SecurityScheme{
		Name:        name,
		Description: description,
		OAuth2:      cfg,
	}
}

// SecuritySchemeOpenIDConnect создаёт схему OpenID Connect.
func SecuritySchemeOpenIDConnect(name, url, description string) SecurityScheme {
	return SecurityScheme{
		Name:             name,
		Description:      description,
		OpenIDConnectURL: url,
	}
}

// securitySchemesToMap преобразует список SecurityScheme в map[string]any для components/securitySchemes.
func securitySchemesToMap(schemes []SecurityScheme) map[string]any {
	if len(schemes) == 0 {
		return nil
	}
	result := make(map[string]any, len(schemes))
	for _, s := range schemes {
		result[s.Name] = securitySchemeToOpenAPI(s)
	}
	return result
}

// securitySchemeToOpenAPI преобразует SecurityScheme в OpenAPI security scheme object.
func securitySchemeToOpenAPI(s SecurityScheme) map[string]any {
	switch s.kind() {
	case kindHTTPBearer, kindHTTPBasic:
		return map[string]any{
			"type":        "http",
			"scheme":      s.HTTPScheme,
			"description": s.Description,
		}

	case kindAPIKey:
		inStr := "header"
		switch s.APIKeyIn {
		case SecurityInQuery:
			inStr = "query"
		case SecurityInCookie:
			inStr = "cookie"
		}
		return map[string]any{
			"type":        "apiKey",
			"in":          inStr,
			"name":        s.APIKeyParam,
			"description": s.Description,
		}

	case kindOAuth2:
		obj := map[string]any{
			"type":        "oauth2",
			"description": s.Description,
		}
		flowObj := map[string]any{}
		flowKey := oauth2FlowKey(s.OAuth2.Flow)

		switch s.OAuth2.Flow {
		case OAuth2FlowImplicit:
			flowObj["authorizationUrl"] = s.OAuth2.AuthURL
		case OAuth2FlowPassword, OAuth2FlowClientCredentials:
			flowObj["tokenUrl"] = s.OAuth2.TokenURL
		case OAuth2FlowAuthorizationCode:
			flowObj["authorizationUrl"] = s.OAuth2.AuthURL
			flowObj["tokenUrl"] = s.OAuth2.TokenURL
		}
		if len(s.OAuth2.Scopes) > 0 {
			scopesMap := make(map[string]string, len(s.OAuth2.Scopes))
			for _, sc := range s.OAuth2.Scopes {
				scopesMap[sc.Name] = sc.Description
			}
			flowObj["scopes"] = scopesMap
		}
		obj["flows"] = map[string]any{flowKey: flowObj}
		return obj

	case kindOpenIDConnect:
		return map[string]any{
			"type":             "openIdConnect",
			"openIdConnectUrl": s.OpenIDConnectURL,
			"description":      s.Description,
		}
	}
	return nil
}

func oauth2FlowKey(flow OAuth2Flow) string {
	switch flow {
	case OAuth2FlowImplicit:
		return "implicit"
	case OAuth2FlowPassword:
		return "password"
	case OAuth2FlowClientCredentials:
		return "clientCredentials"
	case OAuth2FlowAuthorizationCode:
		return "authorizationCode"
	}
	return "authorizationCode"
}

// securityRequirementsToSlice преобразует список SecurityRequirement в OpenAPI security array.
// Для OAuth2 scopes обязательны — передаются как есть.
// Для HTTP Bearer/Basic/API Key scopes не требуются — передаётся пустой массив.
func securityRequirementsToSlice(reqs []SecurityRequirement) []any {
	if len(reqs) == 0 {
		return nil
	}
	result := make([]any, 0, len(reqs))
	for _, r := range reqs {
		entry := map[string]any{}
		if len(r.Scopes) > 0 {
			entry[r.Scheme] = r.Scopes
		} else {
			// OpenAPI требует массив; null вызывает ошибку валидации
			entry[r.Scheme] = []string{}
		}
		result = append(result, entry)
	}
	return result
}

// buildSpecSecuritySection добавляет security блок и components/securitySchemes в spec.
// explicitSchemes — явно зарегистрированные схемы (spec.AddSecurityScheme).
// defaultSecurity — глобальные security требования (spec.DefaultSecurity).
func buildSpecSecuritySection(spec map[string]any, explicitSchemes []SecurityScheme, defaultSecurity []SecurityRequirement, routes []RouteSpec) {
	// Глобальный security блок
	if len(defaultSecurity) > 0 {
		spec["security"] = securityRequirementsToSlice(defaultSecurity)
	}

	// components/securitySchemes
	if len(explicitSchemes) == 0 {
		return
	}

	schemesMap := securitySchemesToMap(explicitSchemes)
	if schemesMap == nil {
		return
	}

	// Добавляем в components
	components, ok := spec["components"].(map[string]any)
	if !ok {
		components = map[string]any{}
		spec["components"] = components
	}
	components["securitySchemes"] = schemesMap
}
