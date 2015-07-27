package libvirt

import (
	"bytes"
	"fmt"
	"html/template"
	"net"
	"net/url"

	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	"gopkg.in/alexzorin/libvirt-go.v2"
)

type stepCreateDomain struct{}

func (s *stepCreateDomain) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	lv, err := libvirt.NewVirConnection(config.LibvirtUrl)
	if err != nil {
		err := fmt.Errorf("Error connecting to libvirt: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	defer lv.CloseConnection()
	if lvd, err := lv.LookupDomainByName(config.VMName); err != nil {

		domainXml := bytes.NewBuffer(nil)
		tmpl, err := template.New("domain").Parse(config.DomainXml)
		if err != nil {
			err := fmt.Errorf("Error creating domain: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		u, err := url.Parse(config.ISOUrl)
		if err != nil {
			err := fmt.Errorf("Error parse iso_url: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		h, p, err := net.SplitHostPort(u.Host)
		if p == "" {
			switch u.Scheme {
			case "https":
				p = "443"
			case "http":
				p = "80"
			}
		}

		data := struct {
			VMName      string
			DiskName    string
			DiskType    string
			PoolName    string
			MemorySize  uint
			ISOUrlProto string
			ISOUrlPath  string
			ISOUrlHost  string
			ISOUrlPort  string
			SSHPort     string
		}{
			config.VMName,
			config.DiskName,
			"raw",
			config.PoolName,
			config.MemorySize,
			u.Scheme,
			u.Path,
			h,
			p,
			"2022",
		}
		err = tmpl.Execute(domainXml, data)
		if err != nil {
			err := fmt.Errorf("Error creating domain: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		lvd, err = lv.DomainCreateXML(string(domainXml.Bytes()), 0)
		if err != nil {
			err := fmt.Errorf("Error creating domain: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	} else {
		defer lvd.Free()
	}
	return multistep.ActionContinue
}

func (s *stepCreateDomain) Cleanup(state multistep.StateBag) {

}
