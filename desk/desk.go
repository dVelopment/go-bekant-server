package desk

import (
  "github.com/stianeikeland/go-rpio"
)

type Direction uint8

var (
  upPin rpio.Pin
  downPin rpio.Pin
)

const (
  Down Direction = iota
  Up
)

func Move(dir Direction) {
  Stop()
  if (dir == Up) {
    upPin.High()
  } else {
    downPin.High()
  }
}

func Stop() {
  upPin.Low()
  downPin.Low()
}

func Init(up int, down int) (err error) {
  err = rpio.Open()

  if (err != nil) {
    return
  }

  upPin = rpio.Pin(up)
  upPin.Output()
  upPin.Low()

  downPin = rpio.Pin(down)
  downPin.Output()
  downPin.Low()

  return nil
}

func Close() {
  rpio.Close()
}
