package ioc

import (
	"github.com/google/uuid"
	"holvit/logging"
	"holvit/utils"
	"reflect"
)

var RootScope *DependencyProvider

type ProviderFunc[T any] func(dc *DependencyProvider) T

func (pf ProviderFunc[T]) untyped() ProviderFunc[any] {
	return func(dc *DependencyProvider) any {
		return pf(dc)
	}
}

type CloseHandler[T any] func(T) error

func (ch CloseHandler[T]) untyped() CloseHandler[any] {
	return func(dependency any) error {
		return ch(dependency.(T))
	}
}

type DependencyProvider struct {
	id                   string
	parentScope          *DependencyProvider
	rootScope            *DependencyProvider
	dependencyCollection *dependencyCollection
	singletonInstances   map[reflect.Type]any
	scopedInstances      map[reflect.Type]any
	closeHandlers        map[reflect.Type]CloseHandler[any]
}

func (dp *DependencyProvider) Close() []error {
	var errors []error

	for chType, ch := range dp.closeHandlers {

		instance, ok := dp.scopedInstances[chType]

		if ok {
			err := ch(instance)

			if err != nil {
				errors = append(errors, err)
			}
		}

		if dp.singletonInstances != nil {
			instance, ok := dp.singletonInstances[chType]

			if ok {
				err := ch(instance)

				if err != nil {
					errors = append(errors, err)
				}
			}
		}
	}

	return errors
}

func newDependencyProvider(dependencies *dependencyCollection, closeHandlers map[reflect.Type]CloseHandler[any]) *DependencyProvider {

	singletonInstances := map[reflect.Type]any{}

	for t, p := range dependencies.singletonProviders {
		singletonInstances[t] = p(nil)
	}

	return &DependencyProvider{
		id:                   uuid.New().String(),
		parentScope:          nil,
		rootScope:            nil,
		dependencyCollection: dependencies,
		singletonInstances:   singletonInstances,
		scopedInstances:      map[reflect.Type]any{},
		closeHandlers:        closeHandlers,
	}
}

func (dp *DependencyProvider) NewScope() *DependencyProvider {
	rootScope := dp.rootScope
	if rootScope == nil {
		rootScope = dp
	}

	return &DependencyProvider{
		id:                   uuid.New().String(),
		parentScope:          dp,
		rootScope:            rootScope,
		dependencyCollection: dp.dependencyCollection,
		singletonInstances:   nil,
		scopedInstances:      map[reflect.Type]any{},
		closeHandlers:        dp.closeHandlers,
	}
}

func Get[TDependency any](dp *DependencyProvider) TDependency {
	dependencyType := utils.TypeOf[TDependency]()

	dependency, ok := dp.getDependency(dependencyType)
	if ok {
		return dependency.(TDependency)
	}

	dependency, ok = dp.getScopedDependency(dependencyType)
	if ok {
		return dependency.(TDependency)
	}

	dependency, ok = dp.getSingletonDependency(dependencyType)
	if ok {
		return dependency.(TDependency)
	}

	logging.Logger.Fatalf("Could not resolve dependency %s", dependencyType.Name())
	panic(dependencyType)
}

func (dp *DependencyProvider) getDependency(dependencyType reflect.Type) (any, bool) {
	provider, ok := dp.dependencyCollection.instanceProviders[dependencyType]
	if !ok {
		return nil, false
	}
	return provider(dp), true
}

func (dp *DependencyProvider) getScopedDependency(dependencyType reflect.Type) (any, bool) {
	dependency, ok := dp.scopedInstances[dependencyType]
	if ok {
		return dependency, true
	}

	provider, ok := dp.dependencyCollection.scopedProviders[dependencyType]
	if !ok {
		return nil, false
	}

	dependency = provider(dp)
	dp.scopedInstances[dependencyType] = dependency

	return dependency, true
}

func (dp *DependencyProvider) getSingletonDependency(dependencyType reflect.Type) (any, bool) {
	rootProvider := dp.rootScope

	if rootProvider == nil {
		rootProvider = dp
	}

	dependency, ok := rootProvider.singletonInstances[dependencyType]
	if ok {
		return dependency, true
	}

	provider, ok := rootProvider.dependencyCollection.singletonProviders[dependencyType]
	if !ok {
		return nil, false
	}

	dependency = provider(rootProvider)
	rootProvider.singletonInstances[dependencyType] = dependency

	return dependency, true
}

type dependencyCollection struct {
	singletonProviders map[reflect.Type]ProviderFunc[any]
	scopedProviders    map[reflect.Type]ProviderFunc[any]
	instanceProviders  map[reflect.Type]ProviderFunc[any]
}

func (dc *dependencyCollection) clone() *dependencyCollection {
	return &dependencyCollection{
		singletonProviders: cloneMap(dc.singletonProviders),
		scopedProviders:    cloneMap(dc.scopedProviders),
		instanceProviders:  cloneMap(dc.instanceProviders),
	}
}

type DependencyProviderBuilder struct {
	dependencyCollection *dependencyCollection
	closeHandlers        map[reflect.Type]CloseHandler[any]
}

func NewDependencyProviderBuilder() *DependencyProviderBuilder {
	return &DependencyProviderBuilder{
		dependencyCollection: &dependencyCollection{
			singletonProviders: map[reflect.Type]ProviderFunc[any]{},
			scopedProviders:    map[reflect.Type]ProviderFunc[any]{},
			instanceProviders:  map[reflect.Type]ProviderFunc[any]{},
		},
		closeHandlers: map[reflect.Type]CloseHandler[any]{},
	}
}

func (dpb *DependencyProviderBuilder) Build() *DependencyProvider {
	return newDependencyProvider(dpb.dependencyCollection.clone(), cloneMap(dpb.closeHandlers))
}

func cloneMap[TKey comparable, TValue any](m map[TKey]TValue) map[TKey]TValue {
	result := make(map[TKey]TValue)

	for key, value := range m {
		result[key] = value
	}

	return result
}

func Add[TDependency any](dpb *DependencyProviderBuilder, providerFunc ProviderFunc[TDependency]) {
	dpb.dependencyCollection.instanceProviders[utils.TypeOf[TDependency]()] = providerFunc.untyped()
}

func AddScoped[TDependency any](dpb *DependencyProviderBuilder, providerFunc ProviderFunc[TDependency]) {
	dpb.dependencyCollection.scopedProviders[utils.TypeOf[TDependency]()] = providerFunc.untyped()
}

func AddSingleton[TDependency any](dpb *DependencyProviderBuilder, providerFunc ProviderFunc[TDependency]) {
	dpb.dependencyCollection.singletonProviders[utils.TypeOf[TDependency]()] = providerFunc.untyped()
}

func AddCloseHandler[TDependency any](dpb *DependencyProviderBuilder, ch CloseHandler[TDependency]) {
	dpb.closeHandlers[utils.TypeOf[TDependency]()] = ch.untyped()
}
