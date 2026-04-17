package nooa

import (
	"sync"
)

var (
	registryMu    sync.RWMutex
	routeRegistry []RouteSpec
	registerHooks []RegisterHook
)

type RegisterHook func(spec RouteSpec) RouteSpec

// AddRegisterHook регистрирует функцию-хук для модификации роута перед добавлением.
func AddRegisterHook(hook RegisterHook) {
	registerHooks = append(registerHooks, hook)
}

// ClearHooks очищает список хуков (используется в тестах).
func ClearHooks() {
	registerHooks = nil
}

// addToRegistryInternal применяет хуки и добавляет роут в реестр.
func addToRegistryInternal(spec RouteSpec) {
	for _, hook := range registerHooks {
		spec = hook(spec)
	}
	routeRegistry = append(routeRegistry, spec)
}

// RegisteredRoutes возвращает копию всех зарегистрированных маршрутов.
func RegisteredRoutes() []RouteSpec {
	registryMu.RLock()
	defer registryMu.RUnlock()
	result := make([]RouteSpec, len(routeRegistry))
	copy(result, routeRegistry)
	return result
}

// ClearRegistry очищает реестр (используется в тестах).
func ClearRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	routeRegistry = nil
}
