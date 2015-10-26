package main

import (
  "fmt"
  "net/http"
  "encoding/json"
  "io/ioutil"
  "os"
  "github.com/dvelopment/go-pi-distance"
)

var settings ConfigType

type ConfigType struct {
  Host HostConfigType
  Sensor SensorConfigType
  Desk DeskConfigType
}

type HostConfigType struct {
  HostName string
  Port int
}

type SensorConfigType struct {
  EchoPin int
  TriggerPin int
}

type DeskConfigType struct {
  UpPin int
  DownPin int
}

type DistanceResultType struct {
  Distance float64
}

func readConfig(fileName string) (ConfigType) {
  content, e := ioutil.ReadFile(fileName);

  if (e != nil) {
    fmt.Printf("File error: %v\n", e)
    os.Exit(1)
  }

  var c ConfigType
  json.Unmarshal(content, &c)

  return c
}

func distanceHandler(w http.ResponseWriter, r *http.Request) {
  var d DistanceResultType

  d.Distance = distance.ReadAverageDistance(10)

  js, err := json.Marshal(d)

  if (err != nil) {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }

  w.Header().Set("Content-Type", "application/json")
  w.Write(js)
}

func main() {
  settings = readConfig("settings.json")

  // initialize distance module
  err := distance.Init(settings.Sensor.EchoPin, settings.Sensor.TriggerPin)

  if (err != nil) {
    fmt.Printf("%v\n", err);
    os.Exit(1)
  }

  fmt.Printf("Listening on %s:%d\n", settings.Host.HostName, settings.Host.Port)

  http.HandleFunc("/distance", distanceHandler)
  http.ListenAndServe(fmt.Sprintf("%s:%d", settings.Host.HostName, settings.Host.Port), nil)
}
