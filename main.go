package main

import (
	"fmt"
	"os"

	"golang.org/x/sync/errgroup"
)

func main() {
	cfg := &Config{
		Url:   os.Getenv("WEATHER_URL"),
		APPID: os.Getenv("WEATHER_APPID"),
		Lat:   os.Getenv("WEATHER_LAT"),
		Lon:   os.Getenv("WEATHER_LON"),
	}

	err := cfg.Check()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	errGroup := new(errgroup.Group)
	c := getWeather(errGroup, cfg)
	if weather, ok := <-c; ok {
		fmt.Println(weather)
		fmt.Println(walkResult(&weather))
	}

	if err := errGroup.Wait(); err != nil {
		fmt.Println(err.Error())
	}
}
