# nooa — not only openapi

Generate OpenAPI 3.0 specifications directly from Go types. Built on top of [elval-gen](https://github.com/arkannsk/elval), nooa inspects your request/response structs and produces valid, self-documenting OpenAPI documents — no manual YAML, no code generation step.

```go
spec := nooa.NewSpec(nooa.Info{
    Title:   "My API",
    Version: "1.0.0",
})

nooa.NewRoute[UserReq, UserRes]("POST", "/users", handleCreateUser).
    Summary("Create a user").
    Tags("Users").
    OnSuccess(201, "User created").
    RegisterSpecAndMux(mux, spec)

nooa.RegisterVersionedAPI("", spec, mux)
```

## Features

- **Zero manual specs** — OpenAPI generated from Go types via reflection
- **elval-gen integration** — field annotations (`@evl:validate`, `@oa:*`) drive schema constraints
- **Polymorphism** — `oneOf` with discriminator support
- **Generic types** — `Option[T]`, `Result[T, E]`, and custom generics
- **HTTP parameters** — query, path, and header parameters via `@oa:in` annotations
- **Security schemes** — Bearer, Basic, API Key, OAuth 2.0, OpenID Connect
- **Tag descriptions** — group routes with descriptive tags for documentation UIs
- **Error schemas** — register error models once, reference them per-route
- **Documentation UIs** — Swagger UI, Redoc, and Scalar mounted with one call
- **Versioned specs** — isolate multiple API versions in separate `*Spec` instances
- **Thread-safe** — lazy single-generation with `sync.RWMutex`

## Installation

```bash
go get github.com/arkannsk/nooa
```

Requires [elval-gen](https://github.com/arkannsk/elval) for schema generation from struct annotations.

## Quick Start

### 1. Define your types

```go
type CreateUserRequest struct {
    // @oa:description "User name"
    // @evl:validate required
    // @evl:validate min:1
    Name string `json:"name"`

    // @oa:description "Email address"
    // @evl:validate required
    // @evl:validate pattern:email
    Email string `json:"email"`
}

type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

### 2. Create a spec and define routes

```go
mux := http.NewServeMux()
spec := nooa.NewSpec(nooa.Info{
    Title:       "User API",
    Version:     "1.0.0",
    Description: "A simple user management API",
})

spec.AddTag("Users", "User management operations")

nooa.NewRoute[CreateUserRequest, User]("POST", "/users", handleCreateUser).
    Summary("Create a new user").
    Description("Creates a user and returns the created resource").
    Tags("Users").
    OnSuccess(201, "User created").
    RegisterSpecAndMux(mux, spec)

nooa.NewRoute[struct{}, User]("GET", "/users/{id}", handleGetUser).
    Summary("Get user by ID").
    Tags("Users").
    OnSuccess(200, "User found").
    RegisterSpecAndMux(mux, spec)
```

### 3. Mount documentation

```go
// Swagger UI (default) — serves /openapi.json and /docs/
nooa.RegisterVersionedAPI("", spec, mux)

// Optional: Redoc or Scalar UI
nooa.RegisterRedoc("", spec, mux)   // /redoc/
nooa.RegisterScalar("", spec, mux)  // /scalar/
```

### 4. Run

```go
log.Fatal(http.ListenAndServe(":8080", mux))
```

Visit `http://localhost:8080/docs/` for Swagger UI, or `http://localhost:8080/openapi.json` for raw JSON.

## API Reference

### Spec

The `*Spec` is an isolated OpenAPI specification generator. Create one per API version.

```go
spec := nooa.NewSpec(nooa.Info{
    Title:       "My API",
    Version:     "1.0.0",
    Description: "API description",
})
```

| Method                                 | Description                       |
| -------------------------------------- | --------------------------------- |
| `AddRoute(r RouteSpec)`                | Add a route to the specification  |
| `RegisterModel(name, instance)`        | Register a model schema           |
| `AddTag(name, description)`            | Register a tag with a description |
| `AddError(status, model, description)` | Register a global error schema    |
| `AddSecurityScheme(name, scheme)`      | Register a security scheme        |
| `DefaultSecurity(reqs...)`             | Set global security requirements  |
| `SetTransformers(fns...)`              | Add spec transformers             |
| `ServeHTTP(w, r)`                      | Serve the generated OpenAPI JSON  |

### Routes

Routes are built with a generic fluent builder:

```go
nooa.NewRoute[RequestType, ResponseType](method, path, handler).
    Summary("...").
    Description("...").
    Tags("tag1", "tag2").
    OperationID("customId").
    Secure("bearerAuth", "read", "write").
    OnSuccess(200, "OK").
    OnClientErr(400, "Bad request").
    OnServerErr(500, "Internal error").
    OnNoContent(204, "Deleted").
    PossibleErr(http.StatusBadRequest, http.StatusNotFound).
    Prefix("/api/v1").
    RegisterSpecAndMux(mux, spec)
```

| Method                             | Description                        |
| ---------------------------------- | ---------------------------------- |
| `Summary(s)`                       | Operation summary                  |
| `Description(s)`                   | Operation description              |
| `Tags(...)`                        | Assign tags to the route           |
| `OperationID(id)`                  | Custom operation ID                |
| `Deprecated()`                     | Mark as deprecated                 |
| `Secure(scheme, scopes...)`        | Per-route security requirement     |
| `RequestContentType(cts...)`       | Override request content types     |
| `RequestBodySchema(name)`          | Override request body schema name  |
| `ResponseSchema(status, name)`     | Bind a schema to a status code     |
| `OnSuccess(status, desc, ct...)`   | Success response                   |
| `OnClientErr(status, desc, ct...)` | 4xx error response                 |
| `OnServerErr(status, desc, ct...)` | 5xx error response                 |
| `OnNoContent(status, desc)`        | 204-like no-body response          |
| `PossibleErr(statuses...)`         | Reference global error schemas     |
| `Prefix(p)`                        | Add path prefix (e.g. `/api/v1`)   |
| `Extension(key, value)`            | Add vendor extension (`x-...`)     |
| `Register(mux)`                    | Register handler in mux (no spec)  |
| `RegisterSpec(spec)`               | Register in spec only (no handler) |
| `RegisterSpecAndMux(mux, spec)`    | Register both                      |

### Tags

Register tags with descriptions at the spec level. They appear in the OpenAPI `tags` array and are displayed in documentation UIs:

```go
spec.AddTag("Users", "User management operations")
spec.AddTag("Auth", "Authentication and authorization")
```

Routes reference tags by name:

```go
nooa.NewRoute[Req, Res]("GET", "/users", handler).
    Tags("Users").
    ...
```

### Error Schemas

Register error models once at the spec level, then reference them per-route:

```go
spec.AddError(http.StatusBadRequest, new(ValidationError), "Validation failed")
spec.AddError(http.StatusNotFound, new(NotFoundError), "Resource not found")

// Per-route: reference by status code
nooa.NewRoute[Req, Res]("GET", "/users/{id}", handler).
    PossibleErr(http.StatusBadRequest, http.StatusNotFound).
    ...
```

The error model is automatically registered in `components/schemas`.

### Security

#### Register schemes

```go
// Bearer token (JWT)
spec.AddSecurityScheme("bearerAuth", nooa.SecuritySchemeBearer("JWT authorization"))

// Basic auth
spec.AddSecurityScheme("basicAuth", nooa.SecuritySchemeBasic("HTTP Basic auth"))

// API key in header
spec.AddSecurityScheme("apiKey", nooa.SecuritySchemeAPIKey(
    "X-API-Key", nooa.SecurityInHeader, "API key authentication"))

// OAuth2 — structured config
spec.AddSecurityScheme("oauth2", nooa.SecuritySchemeOAuth2("oauth2", "OAuth2", nooa.OAuth2Config{
    Flow:     nooa.OAuth2FlowAuthorizationCode,
    AuthURL:  "https://auth.example.com/authorize",
    TokenURL: "https://auth.example.com/token",
    Scopes: []nooa.OAuth2Scope{
        {Name: "read", Description: "Read access"},
        {Name: "write", Description: "Write access"},
    },
}))

// OpenID Connect
spec.AddSecurityScheme("oidc", nooa.SecuritySchemeOpenIDConnect(
    "oidc", "https://accounts.google.com/.well-known/openid-configuration", "Google OIDC"))
```

#### Global security

```go
// All routes require bearerAuth with read scope by default
spec.DefaultSecurity(nooa.SecurityRequirement{
    Scheme: "bearerAuth",
    Scopes: []string{"read"},
})
```

#### Per-route security

```go
nooa.NewRoute[Req, Res]("POST", "/admin", handler).
    Secure("oauth2", "write").
    ...
```

To disable global security on a specific route, set an empty `Security` on the route spec.

#### Security types

| Type           | Helper                                         | Parameters                                                                       |
| -------------- | ---------------------------------------------- | -------------------------------------------------------------------------------- |
| Bearer         | `SecuritySchemeBearer(desc)`                   | description                                                                      |
| Basic          | `SecuritySchemeBasic(desc)`                    | description                                                                      |
| API Key        | `SecuritySchemeAPIKey(name, in, desc)`         | param name, `SecurityInHeader`/`SecurityInQuery`/`SecurityInCookie`, description |
| OAuth2         | `SecuritySchemeOAuth2(name, desc, cfg)`        | name, description, `OAuth2Config`                                                |
| OpenID Connect | `SecuritySchemeOpenIDConnect(name, url, desc)` | name, discovery URL, description                                                 |

**OAuth2 flows:** `OAuth2FlowImplicit`, `OAuth2FlowPassword`, `OAuth2FlowClientCredentials`, `OAuth2FlowAuthorizationCode`.

### Versioned APIs

Each version gets its own `*Spec` and its own mount path:

```go
v1Spec := nooa.NewSpec(nooa.Info{Title: "API v1", Version: "1.0.0"})
v2Spec := nooa.NewSpec(nooa.Info{Title: "API v2", Version: "2.0.0"})

nooa.RegisterVersionedAPI("v1", v1Spec, mux)
// JSON:  /v1/openapi.json
// Docs:  /docs/v1/

nooa.RegisterVersionedAPI("v2", v2Spec, mux)
// JSON:  /v2/openapi.json
// Docs:  /docs/v2/
```

### Content Type Constants

```go
nooa.CTJSON           // application/json
nooa.CTProblemJSON    // application/problem+json
nooa.CTXML            // application/xml
nooa.CTForm           // application/x-www-form-urlencoded
nooa.CTMultipart      // multipart/form-data
nooa.CTOctetStream    // application/octet-stream
nooa.CTPNG            // image/png
nooa.CTHTML           // text/html
nooa.CTPlainText      // text/plain
nooa.CTCSV            // text/csv
```

### Spec Transformers

Transform the generated spec before serialization:

```go
spec.SetTransformers(func(spec map[string]any) map[string]any {
    spec["x-custom"] = "value"
    return spec
})
```

## HTTP Parameters

Use `@oa:in` annotations to document query, path, and header parameters:

```go
type SearchRequest struct {
    // @oa:in query
    // @oa:description "Search keyword"
    Query string `json:"query"`

    // @oa:in query
    // @oa:description "Page number"
    Page int `json:"page"`

    // @oa:in header X-API-Key
    // @oa:description "API key"
    APIKey string
}
```

Models implementing `OaParams() []*oa.Parameter` have their parameters automatically collected and attached to routes. See example `08_http_params`.

## elval-gen Annotations

nooa relies on [elval-gen](https://github.com/arkannsk/elval) for schema generation. Annotations are written as **Go comments above fields or types**:

### Schema Annotations (`@oa:*`)

| Annotation                       | Description               | Example                                              |
| -------------------------------- | ------------------------- | ---------------------------------------------------- |
| `@oa:description`                | Field or type description | `// @oa:description "User name"`                     |
| `@oa:example`                    | Example value             | `// @oa:example "hello"`                             |
| `@oa:default`                    | Default value             | `// @oa:default "active"`                            |
| `@oa:enum`                       | Enum values               | `// @oa:enum active,inactive,pending`                |
| `@oa:in`                         | Parameter location        | `// @oa:in query` or `// @oa:in header X-API-Key`    |
| `@oa:ignore`                     | Exclude field from schema | `// @oa:ignore`                                      |
| `@oa:rewrite`                    | Override field type       | `// @oa:rewrite "string"`                            |
| `@oa:rewrite.ref`                | Override `$ref` target    | `// @oa:rewrite.ref "#/components/schemas/GeoPoint"` |
| `@oa:oneOf`                      | Polymorphic variants      | `// @oa:oneOf "CircleShape,RectangleShape"`          |
| `@oa:discriminator.propertyName` | Discriminator property    | `// @oa:discriminator.propertyName "type"`           |
| `@oa:discriminator.mapping`      | Discriminator mapping     | `// @oa:discriminator.mapping "circle:CircleShape"`  |

### Validation Annotations (`@evl:validate`)

| Constraint           | Description            | Example                                 |
| -------------------- | ---------------------- | --------------------------------------- |
| `required`           | Field is required      | `// @evl:validate required`             |
| `min:N`              | Minimum value / length | `// @evl:validate min:3`                |
| `max:N`              | Maximum value / length | `// @evl:validate max:50`               |
| `gt:N` / `gte:N`     | Greater than / equal   | `// @evl:validate gt:0`                 |
| `lt:N` / `lte:N`     | Less than / equal      | `// @evl:validate lte:999`              |
| `eq:N` / `neq:N`     | Equal / not equal      | `// @evl:validate eq:42`                |
| `pattern:regex`      | Regex pattern          | `// @evl:validate pattern:email`        |
| `enum:vals`          | Enum constraint        | `// @evl:validate enum:active,inactive` |
| `contains:sub`       | Must contain substring | `// @evl:validate contains:admin`       |
| `starts_with:prefix` | Must start with        | `// @evl:validate starts_with:https://` |
| `ends_with:suffix`   | Must end with          | `// @evl:validate ends_with:.com`       |
| `not-zero`           | Non-zero value         | `// @evl:validate not-zero`             |
| `len:N`              | Exact length (slices)  | `// @evl:validate len:3`                |

### Example

```go
type CreateUserRequest struct {
    // @oa:description "User name"
    // @evl:validate required
    // @evl:validate min:1
    // @evl:validate max:50
    Name string `json:"name"`

    // @oa:description "Email address"
    // @evl:validate required
    // @evl:validate pattern:email
    Email string `json:"email"`

    // @oa:in query
    // @oa:description "Search keyword"
    Query string `json:"query"`
}
```

## Examples

The [`examples/elval-integration/`](examples/elval-integration/) directory contains 13 integration examples demonstrating every feature:

| #   | Example                                                     | Description                                                       |
| --- | ----------------------------------------------------------- | ----------------------------------------------------------------- |
| 01  | [basic_types](examples/elval-integration/01_basic_types/)   | Primitive Go types mapped to OpenAPI schemas                      |
| 02  | [files_stream](examples/elval-integration/02_files_stream/) | File types, streams, `io.Reader`, `multipart.File`                |
| 03  | [nested](examples/elval-integration/03_nested/)             | Nested structs with automatic dependency discovery                |
| 04  | [slice_maps](examples/elval-integration/04_slice_maps/)     | Slices, maps, and fixed arrays                                    |
| 05  | [generics](examples/elval-integration/05_generics/)         | Generic types: `Option[T]`, `Result[T, E]`, custom generics       |
| 06  | [polymorphism](examples/elval-integration/06_polymorphism/) | `oneOf` with discriminator and polymorphic shapes                 |
| 07  | [rewrite](examples/elval-integration/07_rewrite/)           | Type rewriting with `@oa:rewrite` and `@oa:rewrite.ref`           |
| 08  | [http_params](examples/elval-integration/08_http_params/)   | Query, path, and header parameters with security                  |
| 09  | [ignore](examples/elval-integration/09_ignore/)             | Excluding fields and types with `@oa:ignore`                      |
| 10  | [validators](examples/elval-integration/10_validators/)     | Validation constraints: strings, numbers, enums, dates            |
| 11  | [edge_cases](examples/elval-integration/11_edge_cases/)     | Empty structs, pointer chains, circular refs, `nil`, interfaces   |
| 12  | [custom_types](examples/elval-integration/12_custom_types/) | Type aliases, custom readers, embedded structs                    |
| 13  | [mixed](examples/elval-integration/13_mixed/)               | Comprehensive example combining all features with OAuth2 security |

Run any example:

```bash
go run examples/elval-integration/01_basic_types/main.go
```

Then visit `http://localhost:9090/docs/` for Swagger UI.

## Validation

All generated specs pass [stoplightio/vacuum](https://github.com/stoplightio/vacuum) validation with Grade A/A+. Run:

```bash
make vacuum
```

## Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Go Structs  │────>│  elval-gen   │────>│  *oa.Schema  │
│  (models)    │     │  (codegen)   │     │  (schemas)   │
└──────────────┘     └──────────────┘     └──────┬───────┘
                                                  │
┌──────────────┐     ┌──────────────┐     ┌──────▼───────┐
│  Handlers    │────>│  RouteSpec   │────>│    Spec      │────> OpenAPI 3.0 JSON
│  (http)      │     │  (routes)    │     │  (generator) │
└──────────────┘     └──────────────┘     └──────────────┘
```

1. **elval-gen** generates `OaSchema()` and `OaParams()` methods on your structs
2. **nooa** inspects these via reflection to build `components/schemas` and operation parameters
3. **Spec** assembles everything into a valid OpenAPI 3.0 document on first request (lazy, cached)

## License

MIT
