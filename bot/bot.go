package bot

import (
  "github.com/dvelopment/go-bekant-server/desk"
  "github.com/dvelopment/go-bekant-server/distance"
  "fmt"
  "github.com/stianeikeland/go-rpio"
  "time"
  "sort"
)

type JoystickConfigType struct {
  UpPin int
  DownPin int
  LeftPin int
  RightPin int
}

type SensorConfigType struct {
  EchoPin int
  TriggerPin int
}

type DeskConfigType struct {
  UpPin int
  DownPin int
}

var (
  upButton rpio.Pin
  downButton rpio.Pin
  leftButton rpio.Pin
  rightButton rpio.Pin

  upState, downState, leftState, rightState rpio.State

  moving chan<- desk.Direction
  stopped chan<- bool
)

type ButtonState uint8

const (
  On ButtonState = iota
  Off
)

const DISTANCE_READS = 11

func Move(dir desk.Direction) {
  desk.Move(dir)
}

func Stop() {
  desk.Stop()
}

func Close() {
  desk.Close()
  distance.Close()
}

func ReadDistance() (float64) {
  distances := []float64{}

  for i := 0; i < DISTANCE_READS; i++ {
    distances = append(distances, distance.ReadDistance())
    distance.Pause()
  }

  sort.Float64s(distances)

  return distances[DISTANCE_READS / 2]
}

func ButtonChanged(name string, state rpio.State) (ButtonState) {
  if state == rpio.Low {
    fmt.Printf("Button pressed: %s\n", name)
    return On
  } else {
    fmt.Printf("Button released: %s\n", name)
    return Off
  }
}

func Run() {
  var state ButtonState

  fmt.Println("[Bot] started")

  for true {
    tmpState := upButton.Read()
    if (upState != tmpState) {
      state = ButtonChanged("up", tmpState)
      upState = tmpState

      if (state == On) {
        Move(desk.Up)
        moving <- desk.Up
      } else {
        Stop()
        stopped <- true
      }
    }

    tmpState = downButton.Read()
    if (downState != tmpState) {
      state = ButtonChanged("down", tmpState)
      downState = tmpState

      if (state == On) {
        Move(desk.Down)
        moving <- desk.Down
      } else {
        Stop()
        stopped <- true
      }
    }

    tmpState = leftButton.Read()
    if (leftState != tmpState) {
      ButtonChanged("left", tmpState)
      leftState = tmpState
    }

    tmpState = rightButton.Read()
    if (rightState != tmpState) {
      ButtonChanged("right", tmpState)
      rightState = tmpState
    }

    time.Sleep(10 * time.Millisecond)
  }
}

func Init(joystickConfig JoystickConfigType, deskConfig DeskConfigType, m chan<- desk.Direction, s chan<- bool) {
  desk.Init(deskConfig.UpPin, deskConfig.DownPin)
  moving = m
  stopped = s

  upState = rpio.High
  downState = rpio.High
  leftState = rpio.High
  rightState = rpio.High

  upButton = rpio.Pin(joystickConfig.UpPin)
  downButton = rpio.Pin(joystickConfig.DownPin)
  leftButton = rpio.Pin(joystickConfig.LeftPin)
  rightButton = rpio.Pin(joystickConfig.RightPin)

  upButton.PullUp()
  downButton.PullUp()
  leftButton.PullUp()
  rightButton.PullUp()

  upButton.Input()
  downButton.Input()
  leftButton.Input()
  rightButton.Input()
}
