package elevator

import (
	"root/config"
	"root/elevio"
	"root/watchdog"
	"time"
)

type State struct {
	Obstructed bool
	Motorstop  bool
	Behaviour  Behaviour
	Floor      int
	Direction  Direction
}

type Behaviour int

const (
	Idle Behaviour = iota
	DoorOpen
	Moving
)

func (b Behaviour) ToString() string {
	return map[Behaviour]string{Idle: "idle", DoorOpen: "doorOpen", Moving: "moving"}[b]
}

func Elevator(newAssignmentC <-chan Assignments, newLocalElevStateC chan<- State, deliveredAssignmentC chan<- elevio.ButtonEvent) {
	doorOpenC := make(chan bool, 16)
	doorClosedC := make(chan bool, 16)
	floorEnteredC := make(chan int)
	obstructionC := make(chan bool, 16) // stuckC
	startMovingC := make(chan bool, 16)
	stopMovingC := make(chan bool, 16)

	go Door(doorClosedC, doorOpenC, obstructionC)
	go elevio.PollFloorSensor(floorEnteredC)
	go watchdog.MotorWatchdog(config.WatchdogTime, motorC, startMovingC, stopMovingC)

	elevio.SetMotorDirection(elevio.MD_Down)
	state := State{Direction: Down, Behaviour: Moving}

	var assignments Assignments

    motorTimer:= time.NewTimer(config.WatchdogTime)
	motorTimer.Stop()

	for {
		select {
		case <-doorClosedC:
			switch state.Behaviour {
			case DoorOpen:
				switch {
				case assignments.ReqInDirection(state.Floor, state.Direction):
					elevio.SetMotorDirection(state.Direction.toMD())
					state.Behaviour = Moving
					motorTimer = time.NewTimer(config.WatchdogTime)
					motorC <- false
					newLocalElevStateC <- state

				case assignments[state.Floor][state.Direction.toOpposite()]:
					doorOpenC <- true
					state.Direction = state.Direction.toOpposite()
					EmptyAssigner(state.Floor, state.Direction, assignments, deliveredAssignmentC)
					newLocalElevStateC <- state

				case assignments.ReqInDirection(state.Floor, state.Direction.toOpposite()):
					state.Direction = state.Direction.toOpposite()
					elevio.SetMotorDirection(state.Direction.toMD())
					state.Behaviour = Moving
					motorTimer = time.NewTimer(config.WatchdogTime)
			motorC <- false
					newLocalElevStateC <- state

				default:
					state.Behaviour = Idle
					newLocalElevStateC <- state
				}
			default:
				panic("DoorClosed in wrong state")
			}

		case state.Floor = <-floorEnteredC:
			elevio.SetFloorIndicator(state.Floor)
			stopMovingC <- true
			switch state.Behaviour {
			case Moving:
				switch {
				case assignments[state.Floor][state.Direction]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					doorOpenC <- true
					EmptyAssigner(state.Floor, state.Direction, assignments, deliveredAssignmentC)
					state.Behaviour = DoorOpen

				case assignments[state.Floor][elevio.BT_Cab] && assignments.ReqInDirection(state.Floor, state.Direction):
					elevio.SetMotorDirection(elevio.MD_Stop)
					doorOpenC <- true
					EmptyAssigner(state.Floor, state.Direction, assignments, deliveredAssignmentC)
					state.Behaviour = DoorOpen

				case assignments[state.Floor][elevio.BT_Cab] && !assignments[state.Floor][state.Direction.toOpposite()]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					doorOpenC <- true
					EmptyAssigner(state.Floor, state.Direction, assignments, deliveredAssignmentC)
					state.Behaviour = DoorOpen

				case assignments.ReqInDirection(state.Floor, state.Direction):
					motorTimer = time.NewTimer(config.WatchdogTime)
			motorC <- false

				case assignments[state.Floor][state.Direction.toOpposite()]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					doorOpenC <- true
					state.Direction = state.Direction.toOpposite()
					EmptyAssigner(state.Floor, state.Direction, assignments, deliveredAssignmentC)
					state.Behaviour = DoorOpen

				case assignments.ReqInDirection(state.Floor, state.Direction.toOpposite()):
					state.Direction = state.Direction.toOpposite()
					elevio.SetMotorDirection(state.Direction.toMD())
					motorTimer = time.NewTimer(config.WatchdogTime)
					motorC <- false

				default:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Behaviour = Idle
				}
			default:
				panic("FloorEntered in wrong state")
			}
			newLocalElevStateC <- state

		case assignments = <-newAssignmentC:
			switch state.Behaviour {
			case Idle:
				switch {
				case assignments[state.Floor][state.Direction] || assignments[state.Floor][elevio.BT_Cab]:
					doorOpenC <- true
					EmptyAssigner(state.Floor, state.Direction, assignments, deliveredAssignmentC)
					state.Behaviour = DoorOpen
					newLocalElevStateC <- state

				case assignments[state.Floor][state.Direction.toOpposite()]:
					doorOpenC <- true
					state.Direction = state.Direction.toOpposite()
					EmptyAssigner(state.Floor, state.Direction, assignments, deliveredAssignmentC)
					state.Behaviour = DoorOpen
					newLocalElevStateC <- state

				case assignments.ReqInDirection(state.Floor, state.Direction):
					elevio.SetMotorDirection(state.Direction.toMD())
					state.Behaviour = Moving
					newLocalElevStateC <- state
					motorTimer = time.NewTimer(config.WatchdogTime)
					motorC <- false

				case assignments.ReqInDirection(state.Floor, state.Direction.toOpposite()):
					state.Direction = state.Direction.toOpposite()
					elevio.SetMotorDirection(state.Direction.toMD())
					state.Behaviour = Moving
					newLocalElevStateC <- state
					motorTimer = time.NewTimer(config.WatchdogTime)
			motorC <- false
				default:
				}

			case DoorOpen:
				switch {
				case assignments[state.Floor][elevio.BT_Cab] || assignments[state.Floor][state.Direction]:
					doorOpenC <- true
					EmptyAssigner(state.Floor, state.Direction, assignments, deliveredAssignmentC)

				}

			case Moving:

			default:
				panic("Assignments in wrong state")
			}
		case motorStuck := <-motorTimer.C:
			if stuck != state.Stuck {
				state.Stuck = stuck
				newLocalElevStateC <- state
			}
		case obstruction := <-obstructionC:
			if obstruction != state.Stuck {
				state.Stuck = stuck
				newLocalElevStateC <- state
			}
		case motor := <-motorC:
			if motor != state.Motorstop {
				state.Stuck = stuck
				newLocalElevStateC <- state
			}
		}
	}
}
