package model

import (
	"log"
	vbox "github.com/rmxymh/go-virtualbox"
)

type Instance struct {
	Name string

	lastBootOrder		[]string
	nextBootOrder		[]string
	needToRestoreBootOrder	bool
	changeBootOrder		bool
}

var instances map[string]Instance

func init() {
	instances = make(map[string]Instance)
}

func AddInstnace(name string) {
	newInstance := Instance {
		Name: name,
	}
	instances[name] = newInstance
	log.Println("Add instance ", name)
}

func DeleteInstance(name string) {
	_, ok := instances[name]
	if ok {
		delete(instances, name)
	}
	log.Println("Remove instance ", name)
}

func GetInstance(name string) (instance Instance, ok bool) {
	instance, ok = instances[name]
	return instance, ok
}

func (instance *Instance)IsRunning() bool {
	machine, err := vbox.GetMachine(instance.Name)

	if err == nil && machine.State == vbox.Running {
		return true
	}
	return false
}

func (instance *Instance)SetBootDevice(dev string) {
	machine, err := vbox.GetMachine(instance.Name)

	if err != nil {
		log.Fatalf("    Instance: Failed to set BootDevice to VM %s: %s", instance.Name, err.Error())
		return
	}

	newBootOrder := []string{dev}
	for _, d := range machine.BootOrder {
		if d != dev {
			newBootOrder = append(newBootOrder, d)
		}
	}

	instance.nextBootOrder = newBootOrder
	instance.changeBootOrder = true
}

func (instance *Instance)PowerOff() {
	machine, err := vbox.GetMachine(instance.Name)

	if err != nil {
		log.Fatalf("    Instance: Failed to find VM %s and power off it: %s", instance.Name, err.Error())
		return
	}

	machine.Stop()

	if instance.needToRestoreBootOrder {
		machine.BootOrder = instance.lastBootOrder
		machine.Modify()
		instance.lastBootOrder = make([]string, 4)
		instance.needToRestoreBootOrder = false
	}

}

func (instance *Instance)PowerOn() {
	machine, err := vbox.GetMachine(instance.Name)

	if err != nil {
		log.Fatalf("    Instance: Failed to find VM %s and power on it: %s", instance.Name, err.Error())
		return
	}

	if instance.changeBootOrder {
		instance.lastBootOrder = machine.BootOrder
		machine.BootOrder = instance.nextBootOrder
		machine.Modify()
		instance.nextBootOrder = make([]string, 4)
		instance.changeBootOrder = false
		instance.needToRestoreBootOrder = true
	}

	machine.Start()
}