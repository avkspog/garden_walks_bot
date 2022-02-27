package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

type Main struct {
	Temp      float32 `json:"temp,omitempty"`
	FeelsLike float32 `json:"feels_like,omitempty"`
	TempMin   float32 `json:"temp_min,omitempty"`
	TempMax   float32 `json:"temp_max,omitempty"`
	Pressure  int     `json:"pressure,omitempty"`
	Humidity  int     `json:"humidity,omitempty"`
}

type Wind struct {
	Speed float32 `json:"speed,omitempty"`
	Deg   float32 `json:"deg,omitempty"`
}

type Weather struct {
	Main Main `json:"main,omitempty"`
	Wind Wind `json:"wind,omitempty"`
}

type Config struct {
	Url   string
	APPID string
	Lat   string
	Lon   string
}

const (
	GOOD_WEATHER  = 1
	BAD_WEATHER   = 2
	CHECK_WEATHER = 3
)

func walkResult(w *Weather) (int8, string) {
	t := w.Main.Temp
	v := w.Wind.Speed

	if t >= 0 {
		return CHECK_WEATHER, w.Text("Прогулка не отменяется.\n☀️")
	}

	if t >= -14 && t < 0 {
		return GOOD_WEATHER, w.Text("Прогулка не отменяется.\n🌡️")
	}

	if t >= -30 && t <= -15 && v > 7 {
		return BAD_WEATHER, w.Text("Прогулка отменяется.\n☃️")
	}

	if t >= -30 && t <= -15 && v <= 7 {
		return GOOD_WEATHER, w.Text("Прогулка не отменяется.\n🌡️")
	}

	if t >= -15 && t < 0 && v >= 7 {
		return GOOD_WEATHER, w.Text("Прогулка не отменяется.\n🌡️")
	}

	if t < -30 {
		return BAD_WEATHER, w.Text("Прогулка отменяется.\n❄️")
	}

	return CHECK_WEATHER, "Данные о температуре не определены."
}

func (w *Weather) Text(text string) string {
	return fmt.Sprintf("🚶 %s Сейчас %.1fC и ветер %.1f м/c.",
		text, w.Main.Temp, w.Wind.Speed)
}

func (c *Config) Check() error {
	if c.Url == "" {
		return errors.New("URL must be specified")
	}

	if c.APPID == "" {
		return errors.New("APPID must be specified")
	}

	if c.Lat == "" {
		return errors.New("Latitude must be specified")
	}

	if c.Lon == "" {
		return errors.New("Langitide must be specified")
	}

	return nil
}

func (c *Config) GetURL() string {
	var b bytes.Buffer

	b.WriteString(c.Url)
	b.WriteString("?lat=")
	b.WriteString(c.Lat)
	b.WriteString("&lon=")
	b.WriteString(c.Lon)
	b.WriteString("&appid=")
	b.WriteString(c.APPID)
	b.WriteString("&units=metric")

	return b.String()
}

func getWeather(group *errgroup.Group, cfg *Config) <-chan Weather {
	out := make(chan Weather)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	group.Go(func() error {
		defer close(out)

		resp, err := client.Get(cfg.GetURL())
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		var weather Weather

		err = json.Unmarshal(data, &weather)
		if err != nil {
			return err
		}

		out <- weather

		return nil
	})

	return out
}
