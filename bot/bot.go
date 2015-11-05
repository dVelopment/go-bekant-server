package bot

import (
  "github.com/dvelopment/go-bekant-server/desk"
  "github.com/dvelopment/go-bekant-server/distance"
  "fmt"
  "github.com/stianeikeland/go-rpio"
  "time"
  "sort"
  "sync"
  "runtime"
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

  moving, preferences chan<- desk.Direction
  stopped chan<- bool

  interrupt, isMoving bool

  distanceMutex, interruptMutex *sync.Mutex
)

type ButtonState uint8

const (
  On ButtonState = iota
  Off
)

const DISTANCE_READS = 1

func IsMoving() (bool) {
  return isMoving
}

func Interrupt() {
  interruptMutex.Lock()
  interrupt = true
  desk.Stop()

  // make sure it's being registered
  time.Sleep(500 * time.Millisecond)
  interruptMutex.Unlock()
  runtime.Gosched()
}

func GoUpTo(target float64) {
  if (isMoving) {
    Stop()
  }
  interrupt = false
  Move(desk.Up)
  isMoving = true

  dist := ReadDistance()
  fmt.Printf("GoUpTo: %.2fcm - %.2fcm\n", dist, target)

  for (!interrupt && dist < target) {
    dist = ReadDistance()
    fmt.Printf("GoUpTo: %.2fcm - %.2fcm\n", dist, target)
    time.Sleep(50 * time.Millisecond)
  }
  Stop()
  isMoving = false

  // check accuracy
  if (!interrupt && (dist - target > 0.5)) {
    time.Sleep(500 * time.Millisecond)
    GoDownTo(target)
  }
}

func GoDownTo(target float64) {
  if (isMoving) {
    Stop()
  }
  interrupt = false
  Move(desk.Down)
  isMoving = true
  dist := ReadDistance()
  fmt.Printf("GoDownTo: %.2fcm - %.2fcm\n", dist, target)

  for (!interrupt && (dist == -1 || dist > target)) {
    dist = ReadDistance()
    fmt.Printf("GoDownTo: %.2fcm - %.2fcm\n", dist, target)
    time.Sleep(50 * time.Millisecond)
  }
  Stop()
  isMoving = false

  // check accuracy
  if (!interrupt && (target - dist > 0.5)) {
    time.Sleep(500 * time.Millisecond)
    GoUpTo(target)
  }
}

func Move(dir desk.Direction) {
  // currently moving to a target?
  if (isMoving) {
    // stop it
    Stop()
  }

  desk.Move(dir)
}

func Stop() {
  desk.Stop()
  Interrupt()
  isMoving = false
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
      state = ButtonChanged("left", tmpState)
      leftState = tmpState

      if (state == On) {
        preferences <- desk.Down
      }
    }

    tmpState = rightButton.Read()
    if (rightState != tmpState) {
      state = ButtonChanged("right", tmpState)
      rightState = tmpState

      if (state == On) {
        preferences <- desk.Up
      }
    }

    time.Sleep(10 * time.Millisecond)
  }
}

func Init(joystickConfig JoystickConfigType, deskConfig DeskConfigType, sensorConfig SensorConfigType, m chan<- desk.Direction, s chan<- bool, p chan<- desk.Direction) {
  desk.Init(deskConfig.UpPin, deskConfig.DownPin)
  distance.Init(sensorConfig.EchoPin, sensorConfig.TriggerPin)

  distanceMutex = &sync.Mutex{}
  interruptMutex = &sync.Mutex{}

  moving = m
  stopped = s
  preferences = p

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
