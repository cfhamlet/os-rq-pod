package serv

import (
	"github.com/cfhamlet/os-rq-pod/pkg/slicemap"
	"github.com/segmentio/fasthash/fnv1a"
)

func idFromString(s string) uint64 {
	return fnv1a.HashString64(s)
}

// NamedExtension TODO
type NamedExtension struct {
	name string
	IExtension
}

// ItemID TODO
func (ext *NamedExtension) ItemID() uint64 {
	return idFromString(ext.name)
}

// Name TODO
func (ext *NamedExtension) Name() string {
	return ext.name
}

// Get TODO
func (ext *NamedExtension) Get() IExtension {
	return ext.IExtension
}

// ExtensionManager TODO
type ExtensionManager struct {
	serv       IServ
	extensions *slicemap.Map
}

// NewExtensionManager TODO
func NewExtensionManager(serv IServ) *ExtensionManager {
	return &ExtensionManager{serv, slicemap.New()}
}

// AddExtension TODO
func (mgr *ExtensionManager) AddExtension(name string, extension IExtension) bool {
	return mgr.extensions.Add(&NamedExtension{name, extension})
}

// GetExtension TODO
func (mgr *ExtensionManager) GetExtension(name string) IExtension {
	return mgr.extensions.Get(idFromString(name)).(*NamedExtension).Get()
}

// Setup TODO
func (mgr *ExtensionManager) Setup() (err error) {
	iter := slicemap.NewBaseIter(mgr.extensions)
	iter.Iter(
		func(item slicemap.Item) bool {
			ext := item.(IExtension)
			err = ext.Setup()
			return err != nil
		},
	)
	return
}

// Cleanup TODO
func (mgr *ExtensionManager) Cleanup() (err error) {
	iter := slicemap.NewReverseIter(mgr.extensions)
	iter.Iter(
		func(item slicemap.Item) bool {
			ext := item.(IExtension)
			err = ext.Cleanup()
			return err != nil
		},
	)
	return
}

// IExtension TODO
type IExtension interface {
	Setup() error
	Cleanup() error
}
