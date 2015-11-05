package distance

import (
  "fmt"
  "time"
  "github.com/stianeikeland/go-rpio"
  "sync"
  "runtime"
)

const SPEED_OF_SOUND = 34320.0 // in cm per second

var (
  echoPin rpio.Pin
  triggerPin rpio.Pin

  timeout = 300 * time.Second / SPEED_OF_SOUND // default max distance: 150cm
  mutex *sync.Mutex
)

/**
 * sets the timeout based on the max distance in cm
 */
func SetMaxDistance(cm int) () {
  timeout = time.Duration(cm * 2) * time.Second / SPEED_OF_SOUND;
}

func Pause() {
  time.Sleep(timeout * 2)
}

func ReadDistance() (float64) {
  var res rpio.State
  var start, echoTimeout time.Time

  mutex.Lock()

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
      mutex.Unlock()
      runtime.Gosched()
      return -1
    }
  }

  echoTimeout = start
  echoTimeout = echoTimeout.Add(timeout)
  for (res == rpio.High) {
    res = echoPin.Read()
    if (time.Now().After(echoTimeout)) {
      fmt.Printf("timeout while high\nstart: %d:%d:%d.%d\ntimeout:  %d:%d:%d.%d\necho: %d\n\n",
        start.Hour(), start.Minute(), start.Second(), start.Nanosecond(),
        echoTimeout.Hour(), echoTimeout.Minute(), echoTimeout.Second(), echoTimeout.Nanosecond(),
        echoPin.Read(),
      )
      mutex.Unlock()
      runtime.Gosched()
      return -1
    }
  }

  duration := time.Since(start).Seconds()
  distance := SPEED_OF_SOUND / 2.0 * duration

  mutex.Unlock()
  runtime.Gosched()

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

  mutex = &sync.Mutex{}

  fmt.Println("Waiting for sensor to settle")
  time.Sleep(2 * time.Second)


  return nil
}

func Close() {
  rpio.Close()
}
