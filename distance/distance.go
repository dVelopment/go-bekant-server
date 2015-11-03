package distance

import (
  "fmt"
  "time"
  "github.com/stianeikeland/go-rpio"
)

const SPEED_OF_SOUND = 34320.0 // in cm per second
const READ_INTERVAL = 10 * time.Microsecond // pause between two readings

var (
  echoPin rpio.Pin
  triggerPin rpio.Pin
  // default max distance: 150cm
  timeout = 300 * time.Second / SPEED_OF_SOUND
)

/**
 * sets the timeout based on the max distance in cm
 */
func SetMaxDistance(cm int) () {
  timeout = time.Duration(cm * 2) * time.Second / SPEED_OF_SOUND;
}

func Pause() {
  time.Sleep(READ_INTERVAL)
}

func ReadDistance() (float64) {
  var res rpio.State
  var start, echoTimeout time.Time

  // make sure trigger is low
  triggerPin.Low()
  time.Sleep(2 * time.Microsecond)

  // calculate timeout for echo
  to := time.Now().Add(timeout + (100 * time.Microsecond))

  // trigger a 10µs burst
  triggerPin.High()
  time.Sleep(10 * time.Microsecond)
  triggerPin.Low()

  // wait for echo
  res = echoPin.Read()
  for (res == rpio.Low) {
    res = echoPin.Read()
    start = time.Now()
    if (start.After(to)) {
      fmt.Println("timeout while waiting on echo")
      return -1
    }
  }

  echoTimeout = start
  echoTimeout = echoTimeout.Add(timeout)
  for (res == rpio.High) {
    res = echoPin.Read()
    if (time.Now().After(echoTimeout)) {
      fmt.Println("timeout while high")
      return -1
    }
  }

  duration := time.Since(start).Seconds()

  // fmt.Printf("duration: %.10fs\n", duration);

  distance := SPEED_OF_SOUND / 2.0 * duration

  // fmt.Printf("distance: %.2fcm\n", distance)

  if (distance < 0) {
    // something went wrong
    // try again
    Pause()
    return ReadDistance()
  }

  return distance
}

func Init(echo int, trigger int) (err error) {
  err = rpio.Open()

  if (err != nil) {
    return
  }

  fmt.Printf("echo pin: %d - trigger pin: %d - timeout: %.10fs\n", echo, trigger, timeout.Seconds())

  echoPin = rpio.Pin(echo)
  echoPin.Input()

  triggerPin = rpio.Pin(trigger)
  triggerPin.Output()

  // ensure that triggerPin is set to low
  triggerPin.Low()

  fmt.Println("Waiting for sensor to settle")
  time.Sleep(2 * time.Second)

  return nil
}

func Close() {
  rpio.Close()
}
