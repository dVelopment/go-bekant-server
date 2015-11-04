package main

import (
  "fmt"
  "net/http"
  "encoding/json"
  "io/ioutil"
  "os"
  "github.com/gorilla/mux"
  "strconv"
  "time"
  "math"
  "github.com/dvelopment/go-bekant-server/bot"
  "github.com/dvelopment/go-bekant-server/desk"
  "flag"
)

var (
  settings ConfigType
)

type ConfigType struct {
  Host HostConfigType
  Sensor bot.SensorConfigType
  Desk bot.DeskConfigType
  Joystick bot.JoystickConfigType
  WebApp WebAppConfigType
}

type WebAppConfigType struct {
  HostName string
  Port int
  Secret string
}

type HostConfigType struct {
  HostName string
  Port int
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

  bot.Move(direction)

  result := MoveResultType{Direction: dir}

  js, err := json.Marshal(result)

  if (err != nil) {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  } else {
    w.Header().Set("Content-Type", "application/json")
    w.Write(js)
  }
}


func goHandler(w http.ResponseWriter, r *http.Request) {
  params := mux.Vars(r)

  target, err := strconv.ParseFloat(params["position"], 64)

  if (err != nil) {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }

  // get current position
  position := bot.ReadDistance()

  for (position == -1) {
    position = bot.ReadDistance()
  }

  delta := target - position

  fmt.Printf("delta: %.2fcm\ntarget: %.2fcm\n", delta, target)

  if (math.Abs(delta) > 0.5) {
    if (delta > 0) {
      bot.GoUpTo(target)
    } else {
      bot.GoDownTo(target)
    }
  }

  res := PositionResultType{Position: bot.ReadDistance()}

  js, err2 := json.Marshal(res)

  if (err2 != nil) {
    http.Error(w, err2.Error(), http.StatusInternalServerError)
  }

  w.Header().Set("Content-Type", "application/json")
  w.Write(js)

}

func primeHandler(w http.ResponseWriter, r *http.Request) {
  result := PositionResultType{Position: bot.ReadDistance()}
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

  res.Position = bot.ReadDistance()

  js, err := json.Marshal(res)

  if (err != nil) {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }

  w.Header().Set("Content-Type", "application/json")
  w.Write(js)
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
  // currently moving to a target?
  if (bot.IsMoving()) {
    // stop it
    bot.Interrupt()
  }

  bot.Stop()

  res := PositionResultType{Position: bot.ReadDistance()}

  js, err := json.Marshal(res)

  if (err != nil) {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }

  w.Header().Set("Content-Type", "application/json")
  w.Write(js)
}

func distanceHandler(w http.ResponseWriter, r *http.Request) {
  fmt.Println("GET /distance")
  res := DistanceResultType{Distance: bot.ReadDistance()}

  js, err := json.Marshal(res)

  if (err != nil) {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }

  fmt.Printf("measured distance: %.2f\n", res.Distance)

  w.Header().Set("Content-Type", "application/json")
  w.Write(js)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
  res := StatusResultType{IsPrimed: true}

  js, err := json.Marshal(res)

  if (err != nil) {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }

  w.Header().Set("Content-Type", "application/json")
  w.Write(js)
}

func notifyWebApp(path string, config WebAppConfigType) {
  client := &http.Client{}
  url := fmt.Sprintf("http://%s:%d/desk/%s", config.HostName, config.Port, path)
  req, err := http.NewRequest("POST", url, nil)
  req.Header.Set("Authorization", config.Secret)

  res, err := client.Do(req)

  if err != nil {
    fmt.Printf("[Server] Error notifying web app: %v\n", err)
  } else {
    res.Body.Close()
  }
}

func main() {
  settingsPtr := flag.String("settings", "settings.json", "Path to the settings file")
  flag.Parse()
  settings = readConfig(*settingsPtr)

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

  moving := make(chan desk.Direction, 1)
  stopped := make(chan bool, 1)
  preferences := make(chan desk.Direction, 1)

  bot.Init(settings.Joystick, settings.Desk, settings.Sensor, moving, stopped, preferences)
  go bot.Run()

  go http.ListenAndServe(fmt.Sprintf("%s:%d", settings.Host.HostName, settings.Host.Port), nil)

  time.Sleep(time.Millisecond)

  for {
    select {
    case dir := <-moving:
      fmt.Printf("[Server] desk moving: %v\n", dir)
      var direction string
      if dir == desk.Up {
        direction = "up"
      } else {
        direction = "down"
      }

      notifyWebApp(fmt.Sprintf("moving/%s", direction), settings.WebApp)
    case dir := <-preferences:
      fmt.Printf("[Server] desk next preference: %v\n", dir)
      var direction string
      if dir == desk.Up {
        direction = "up"
      } else {
        direction = "down"
      }

      notifyWebApp(fmt.Sprintf("preferences/%s", direction), settings.WebApp)
    case <-stopped:
      fmt.Println("[Server] desk stopped")
      notifyWebApp("stopped", settings.WebApp)
    }
  }
}
