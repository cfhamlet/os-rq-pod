package serv

import (
	"github.com/cfhamlet/os-rq-pod/pkg/log"
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
	log.Logger.Debug("extensions setup")
	iter.Iter(
		func(item slicemap.Item) bool {
			ext := item.(*NamedExtension)
			log.Logger.Debugf("extension %s setup", ext.Name())
			err = ext.Setup()
			log.Logger.Debugf("extension %s setup finish %v", ext.Name(), err)
			return err != nil
		},
	)
	log.Logger.Debug("extensions setup finish")
	return
}

// Cleanup TODO
func (mgr *ExtensionManager) Cleanup() (err error) {
	iter := slicemap.NewReverseIter(mgr.extensions)
	log.Logger.Debug("extensions cleanup")
	iter.Iter(
		func(item slicemap.Item) bool {
			ext := item.(*NamedExtension)
			log.Logger.Debugf("extension %s cleanup", ext.Name())
			err = ext.Cleanup()
			log.Logger.Debugf("extension %s clearnup finish %v", ext.Name(), err)
			return err != nil
		},
	)
	log.Logger.Debug("extensions cleanup finish")
	return
}

// IExtension TODO
type IExtension interface {
	Setup() error
	Cleanup() error
}
