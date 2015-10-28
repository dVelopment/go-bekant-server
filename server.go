package main

import (
  "fmt"
  "net/http"
  "encoding/json"
  "io/ioutil"
  "os"
  "github.com/dvelopment/go-bekant"
  "github.com/dvelopment/go-pi-distance"
  "github.com/gorilla/mux"
  "strconv"
  "time"
  "math"
)

var (
  settings ConfigType
  interrupt, moving bool
)

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

type PositionResultType struct {
  Position float64
}

type MoveResultType struct {
  Direction string
}

type DistanceResultType struct {
  Distance float64
}

type StatusResultType struct {
  IsPrimed bool
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

func moveHandler(w http.ResponseWriter, r *http.Request) {
  params := mux.Vars(r)
  dir := params["direction"]

  var direction desk.Direction

  if (dir == "up") {
    direction = desk.Up
  } else {
    direction = desk.Down
  }

  desk.Move(direction)

  result := MoveResultType{Direction: dir}

  js, err := json.Marshal(result)

  if (err != nil) {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  } else {
    w.Header().Set("Content-Type", "application/json")
    w.Write(js)
  }
}

func GoUpTo(target float64) {
  interrupt = false
  desk.Move(desk.Up)
  moving = true

  dist := distance.ReadDistance()
  fmt.Printf("GoUpTo: %.2fcm - %.2fcm\n", dist, target)

  for (!interrupt && dist < target) {
    dist = distance.ReadDistance()
    fmt.Printf("GoUpTo: %.2fcm - %.2fcm\n", dist, target)
    time.Sleep(50 * time.Millisecond)
  }
  desk.Stop()
  moving = false

  // check accuracy
  if (dist - target > 0.5) {
    time.Sleep(400 * time.Millisecond)
    GoDownTo(target)
  }
}

func GoDownTo(target float64) {
  interrupt = false
  desk.Move(desk.Down)
  moving = true
  dist := distance.ReadDistance()
  fmt.Printf("GoDownTo: %.2fcm - %.2fcm\n", dist, target)

  for (!interrupt && (dist == -1 || dist > target)) {
    dist = distance.ReadDistance()
    fmt.Printf("GoDownTo: %.2fcm - %.2fcm\n", dist, target)
    time.Sleep(50 * time.Millisecond)
  }
  desk.Stop()
  moving = false

  // check accuracy
  if (target - dist > 0.5) {
    time.Sleep(400 * time.Millisecond)
    GoUpTo(target)
  }
}


func goHandler(w http.ResponseWriter, r *http.Request) {
  params := mux.Vars(r)

  target, err := strconv.ParseFloat(params["position"], 64)

  if (err != nil) {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }

  // currently moving to a target?
  if (moving) {
    // stop it
    interrupt = true
    // make sure it's being registered
    time.Sleep(500 * time.Millisecond)
  }

  // get current position
  position := distance.ReadDistance()

  for (position == -1) {
    position = distance.ReadDistance()
  }

  delta := target - position

  fmt.Printf("delta: %.2fcm\ntarget: %.2fcm\n", delta, target)

  if (math.Abs(delta) > 0.5) {
    if (delta > 0) {
      GoUpTo(target)
    } else {
      GoDownTo(target)
    }
  }

  res := PositionResultType{Position: distance.ReadDistance()}

  js, err2 := json.Marshal(res)

  if (err2 != nil) {
    http.Error(w, err2.Error(), http.StatusInternalServerError)
  }

  w.Header().Set("Content-Type", "application/json")
  w.Write(js)

}

func primeHandler(w http.ResponseWriter, r *http.Request) {
  desk.Prime()

  result := PositionResultType{Position: distance.ReadDistance()}
  js, err := json.Marshal(result)

  if (err != nil) {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  } else {
    w.Header().Set("Content-Type", "application/json")
    w.Write(js)
  }
}

func positionHandler(w http.ResponseWriter, r *http.Request) {
  var res PositionResultType

  res.Position = distance.ReadDistance()

  js, err := json.Marshal(res)

  if (err != nil) {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }

  w.Header().Set("Content-Type", "application/json")
  w.Write(js)
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
  desk.Stop()

  res := PositionResultType{Position: distance.ReadDistance()}

  js, err := json.Marshal(res)

  if (err != nil) {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }

  w.Header().Set("Content-Type", "application/json")
  w.Write(js)
}

func distanceHandler(w http.ResponseWriter, r *http.Request) {
  fmt.Println("GET /distance")
  res := DistanceResultType{Distance: distance.ReadDistance()}

  js, err := json.Marshal(res)

  if (err != nil) {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }

  fmt.Printf("measured distance: %.2f\n", res.Distance)

  w.Header().Set("Content-Type", "application/json")
  w.Write(js)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
  res := StatusResultType{IsPrimed: desk.IsPrimed()}

  js, err := json.Marshal(res)

  if (err != nil) {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }

  w.Header().Set("Content-Type", "application/json")
  w.Write(js)
}

func main() {
  settings = readConfig("settings.json")

  // initialize distance module
  err := desk.Init(settings.Desk.UpPin, settings.Desk.DownPin)

  if (err != nil) {
    fmt.Printf("%v\n", err);
    os.Exit(1)
  }

  err = distance.Init(settings.Sensor.EchoPin, settings.Sensor.TriggerPin)

  if (err != nil) {
    fmt.Printf("%v\n", err);
    os.Exit(1)
  }

  rtr := mux.NewRouter()

  fmt.Printf("Listening on %s:%d\n", settings.Host.HostName, settings.Host.Port)

  rtr.HandleFunc("/position", positionHandler).Methods("GET")
  rtr.HandleFunc("/distance", distanceHandler).Methods("GET")
  rtr.HandleFunc("/status", statusHandler).Methods("GET")
  rtr.HandleFunc("/move/{direction:(up|down)}", moveHandler).Methods("POST")
  rtr.HandleFunc("/go/{position:[0-9.]+}", goHandler).Methods("POST")
  rtr.HandleFunc("/prime", primeHandler).Methods("POST")
  rtr.HandleFunc("/stop", stopHandler).Methods("POST")
  http.Handle("/", rtr)

  http.ListenAndServe(fmt.Sprintf("%s:%d", settings.Host.HostName, settings.Host.Port), nil)
}
