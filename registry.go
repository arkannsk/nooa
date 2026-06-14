package nooa

import (
	"sync"
)

var (
	registryMu    sync.RWMutex
	routeRegistry []RouteSpec
	registerHooks []RegisterHook
)

// RegisterHook — функция-хук для модификации роута перед добавлением в глобальный реестр.
type RegisterHook func(spec RouteSpec) RouteSpec

// addToRegistryInternal применяет хуки и добавляет роут в глобальный реестр.
func addToRegistryInternal(spec RouteSpec) {
	for _, hook := range registerHooks {
		spec = hook(spec)
	}
	routeRegistry = append(routeRegistry, spec)
}

// RegisteredRoutes возвращает копию всех зарегистрированных маршрутов из глобального реестра.
func RegisteredRoutes() []RouteSpec {
	registryMu.RLock()
	defer registryMu.RUnlock()
	result := make([]RouteSpec, len(routeRegistry))
	copy(result, routeRegistry)
	return result
}

// ClearRegistry очищает глобальный реестр (используется в тестах).
func ClearRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	routeRegistry = nil
}
