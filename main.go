package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/strava/go.strava"
)

const port = 8080

type settings struct {
	GoogleCom settingsGoogleCom `toml:"google_com"`
	StravaCom settingsStravaCom `toml:"strava_com"`
}

type settingsGoogleCom struct {
	Key string `toml:"key"`
}

type settingsStravaCom struct {
	ClientID     int    `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
}

var Authenticator *strava.OAuthAuthenticator
var Templates map[string]*template.Template
var Settings *settings

func init() {
	initSettings()
	initTemplates()
}

func initSettings() {
	Settings = &settings{}
	_, err := toml.DecodeFile("settings.toml", Settings)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

}

func initTemplates() {
	layouts, err := filepath.Glob("layout.html")
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	views, err := filepath.Glob("index.html")
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	Templates = make(map[string]*template.Template)
	for _, view := range views {
		files := append(layouts, view)
		parseFiles, err := template.New("layout").Funcs(template.FuncMap{}).ParseFiles(files...)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		Templates[view] = template.Must(parseFiles, err)
	}
}

func main() {
	strava.ClientId = Settings.StravaCom.ClientID
	strava.ClientSecret = Settings.StravaCom.ClientSecret

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	http.HandleFunc("/", index)

	Authenticator = &strava.OAuthAuthenticator{
		CallbackURL:            fmt.Sprintf("http://0.0.0.0:%d/exchange_token", port),
		RequestClientGenerator: nil,
	}
	path, err := Authenticator.CallbackPath()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	http.HandleFunc(path, Authenticator.HandlerFunc(success, failure))

	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func index(responseWriter http.ResponseWriter, request *http.Request) {
	data := map[string]interface{}{
		"google": Settings.GoogleCom.Key,
		"mode":   "index",
		"url":    Authenticator.AuthorizationURL("state1", strava.Permissions.Public, true),
	}
	err := Templates["index.html"].Execute(responseWriter, data)
	if err != nil {
		log.Println(err)
	}
}

func success(auth *strava.AuthorizationResponse, responseWriter http.ResponseWriter, request *http.Request) {
	client := strava.NewClient(auth.AccessToken)
	stats, _ := strava.NewAthletesService(client).Stats(auth.Athlete.Id).Do()
	activities, _ := strava.NewAthletesService(client).ListActivities(auth.Athlete.Id).PerPage(200).Do()
	routesList := [][][2]float64{}
	for _, activity := range activities {
		types := []strava.StreamType{
			strava.StreamTypes.Time,
			strava.StreamTypes.Location,
			strava.StreamTypes.Distance,
			strava.StreamTypes.Elevation,
			strava.StreamTypes.Speed,
			strava.StreamTypes.HeartRate,
			strava.StreamTypes.Cadence,
			strava.StreamTypes.Power,
			strava.StreamTypes.Temperature,
			strava.StreamTypes.Moving,
			strava.StreamTypes.Grade,
		}
		streams, _ := strava.NewActivityStreamsService(client).Get(activity.Id, types).Resolution("high").Do()
		route := [][2]float64{}
		for _, data := range streams.Location.Data {
			point := [2]float64{data[0], data[1]}
			route = append(route, point)
		}
		routesList = append(routesList, route)
	}
	routesBytes, _ := json.Marshal(routesList)
	routesString := template.JS(routesBytes)
	data := map[string]interface{}{
		"google":     Settings.GoogleCom.Key,
		"mode":       "success",
		"athlete":    auth.Athlete,
		"stats":      stats,
		"activities": activities,
		"routes":     routesString,
	}
	err := Templates["index.html"].Execute(responseWriter, data)
	if err != nil {
		log.Println(err)
	}
}

func failure(err error, responseWriter http.ResponseWriter, request *http.Request) {
	data := map[string]interface{}{
		"google": Settings.GoogleCom.Key,
		"mode":   "failure",
		"error":  err.Error(),
	}
	err = Templates["index.html"].Execute(responseWriter, data)
	if err != nil {
		log.Println(err)
	}
}
