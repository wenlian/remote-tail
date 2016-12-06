package storage

import (
	"fmt"
	"sort"

	"github.com/wenlian/remote-tail/command"
)

type StorageDriver interface {
	AddStats(msg command.Message, brokers string) error
	Close() error
}

type StorageDriverFunc func() (StorageDriver, error)

var registeredPlugins = map[string](StorageDriverFunc){}

func RegisterStorageDriver(name string, f StorageDriverFunc) {
	registeredPlugins[name] = f
}

func New(name string) (StorageDriver, error) {
	if name == "" {
		return nil, nil
	}
	f, ok := registeredPlugins[name]
	if !ok {
		return nil, fmt.Errorf("unknown backend storage driver: %s", name)
	}
	return f()
}

func ListDrivers() []string {
	drivers := make([]string, 0, len(registeredPlugins))
	for name := range registeredPlugins {
		drivers = append(drivers, name)
	}
	sort.Strings(drivers)
	return drivers
}
