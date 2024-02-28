package lights

import (
	"root/driver/elevio"
	"root/elevator"
)

func SetLights(light_assignment elevator.Assingments) {
	for floor := 0; floor < 4; floor++ {
		for button := 0; button < 3; button++ {
			if light_assignment[floor][button] {
				elevio.SetButtonLamp( elevio.ButtonType(button), floor, true)
			} else {
				elevio.SetButtonLamp( elevio.ButtonType(button), floor, false)
			}
		}
	}
}

func Lights(lightsAssingmentChan <-chan elevator.Assingments) {
	for {
		select {
		case a := <-lightsAssingmentChan:
			SetLights(a)
		}
	}
}