package libvirt

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	"gopkg.in/alexzorin/libvirt-go.v2"
)

type stepCreateVolume struct{}

func (s *stepCreateVolume) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	var lvp libvirt.VirStoragePool
	lv, err := libvirt.NewVirConnection(config.LibvirtUrl)
	if err != nil {
		err := fmt.Errorf("Error connecting to libvirt: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	defer lv.CloseConnection()
	if lvp, err = lv.LookupStoragePoolByName(config.PoolName); err != nil {
		err := fmt.Errorf("Error getting pool %s: %s", config.PoolName, err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	volumeXml := bytes.NewBuffer(nil)
	tmpl, err := template.New("volume").Parse(config.VolumeXml)
	if err != nil {
		err := fmt.Errorf("Error creating volume: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	data := struct {
		DiskName string
		DiskSize uint
	}{
		config.DiskName,
		config.DiskSize,
	}
	err = tmpl.Execute(volumeXml, data)
	if err != nil {
		err := fmt.Errorf("Error creating volume: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if _, err := lvp.StorageVolCreateXML(string(volumeXml.Bytes()), 0); err != nil {
		err := fmt.Errorf("Error creating volume: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *stepCreateVolume) Cleanup(state multistep.StateBag) {

}