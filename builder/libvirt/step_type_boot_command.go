package libvirt

import (
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/alexzorin/libvirt-go"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
)

const KeyLeftShift uint32 = 0xFFE1

type bootCommandTemplateData struct {
	HTTPIP   string
	HTTPPort uint
	Name     string
}

// This step "types" the boot command into the VM over VNC.
//
// Uses:
//   config *config
//   http_port int
//   ui     packer.Ui
//
// Produces:
//   <nothing>
type stepTypeBootCommand struct{}

func (s *stepTypeBootCommand) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	//	httpPort := state.Get("http_port").(uint)
	//	hostIp := state.Get("host_ip").(string)
	ui := state.Get("ui").(packer.Ui)

	var lvd libvirt.VirDomain
	lv, err := libvirt.NewVirConnection(config.LibvirtUrl)
	if err != nil {
		err := fmt.Errorf("Error connecting to libvirt: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	defer lv.CloseConnection()
	if lvd, err = lv.LookupDomainByName(config.VMName); err != nil {
		err := fmt.Errorf("Error lookup domain: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	defer lvd.Free()

	//	tplData := &bootCommandTemplateData{
	//		hostIp,
	//		httpPort,
	//		config.VMName,
	//	}

	ui.Say("Typing the boot command...")
	for _, command := range config.BootCommand {
		//		command, err := config.tpl.Process(command, tplData)
		//		if err != nil {
		//			err := fmt.Errorf("Error preparing boot command: %s", err)
		//			state.Put("error", err)
		//			ui.Error(err.Error())
		//			return multistep.ActionHalt
		//		}

		// Check for interrupts between typing things so we can cancel
		// since this isn't the fastest thing.
		if _, ok := state.GetOk(multistep.StateCancelled); ok {
			return multistep.ActionHalt
		}

		vncSendString(lvd, command)
	}

	return multistep.ActionContinue
}

func (*stepTypeBootCommand) Cleanup(multistep.StateBag) {}

func vncSendString(d libvirt.VirDomain, original string) {
	special := make(map[string]uint32)
	special["<bs>"] = 0xFF08
	special["<del>"] = 0xFFFF
	special["<enter>"] = 0xFF0D
	special["<esc>"] = 0xFF1B
	special["<f1>"] = 0xFFBE
	special["<f2>"] = 0xFFBF
	special["<f3>"] = 0xFFC0
	special["<f4>"] = 0xFFC1
	special["<f5>"] = 0xFFC2
	special["<f6>"] = 0xFFC3
	special["<f7>"] = 0xFFC4
	special["<f8>"] = 0xFFC5
	special["<f9>"] = 0xFFC6
	special["<f10>"] = 0xFFC7
	special["<f11>"] = 0xFFC8
	special["<f12>"] = 0xFFC9
	special["<return>"] = 0xFF0D
	special["<tab>"] = 0xFF09

	shiftedChars := "~!@#$%^&*()_+{}|:\"<>?"

	// TODO(mitchellh): Ripe for optimizations of some point, perhaps.
	for len(original) > 0 {
		var keyCode uint32
		keyShift := false

		if strings.HasPrefix(original, "<wait>") {
			log.Printf("Special code '<wait>' found, sleeping one second")
			time.Sleep(1 * time.Second)
			original = original[len("<wait>"):]
			continue
		}

		if strings.HasPrefix(original, "<wait5>") {
			log.Printf("Special code '<wait5>' found, sleeping 5 seconds")
			time.Sleep(5 * time.Second)
			original = original[len("<wait5>"):]
			continue
		}

		if strings.HasPrefix(original, "<wait10>") {
			log.Printf("Special code '<wait10>' found, sleeping 10 seconds")
			time.Sleep(10 * time.Second)
			original = original[len("<wait10>"):]
			continue
		}

		for specialCode, specialValue := range special {
			if strings.HasPrefix(original, specialCode) {
				log.Printf("Special code '%s' found, replacing with: %d", specialCode, specialValue)
				keyCode = specialValue
				original = original[len(specialCode):]
				break
			}
		}

		if keyCode == 0 {
			r, size := utf8.DecodeRuneInString(original)
			original = original[size:]
			keyCode = uint32(r)
			keyShift = unicode.IsUpper(r) || strings.ContainsRune(shiftedChars, r)

			log.Printf("Sending char '%c', code %d, shift %v", r, keyCode, keyShift)
		}

		//		if keyShift {
		//			c.KeyEvent(KeyLeftShift, true)
		//		}

		//		time.Sleep(5 * time.Millisecond)
		//VIR_KEYCODE_SET_LINUX, VIR_KEYCODE_SET_USB, VIR_KEYCODE_SET_RFB, VIR_KEYCODE_SET_WIN32, VIR_KEYCODE_SET_XT_KBD
		d.SendKey(libvirt.VIR_KEYCODE_SET_XT_KBD, 50, []uint{uint(keyCode)}, 0)
		//		c.KeyEvent(keyCode, true)
		//		time.Sleep(5 * time.Millisecond)
		//		c.KeyEvent(keyCode, false)

		//		if keyShift {
		//			c.KeyEvent(KeyLeftShift, false)
		//		}
	}
}
