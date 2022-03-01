package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	cache "github.com/patrickmn/go-cache"
	"golang.org/x/sync/errgroup"
	tele "gopkg.in/telebot.v3"
)

const (
	ERRORS             string        = "errors"
	ERROR_EXPIRATION   time.Duration = 24 * time.Hour
	WEATHER            string        = "weather"
	WEATHER_EXPIRATION time.Duration = 30 * time.Minute
)

const (
	GOOD_WEATHER  = 1
	BAD_WEATHER   = 2
	CHECK_WEATHER = 3
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
	Main        Main `json:"main,omitempty"`
	Wind        Wind `json:"wind,omitempty"`
	LastRequest time.Time
}

type Config struct {
	Url   string
	APPID string
	Lat   string
	Lon   string
}

var logCache *cache.Cache
var weatherCache *cache.Cache

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
	time := fmt.Sprintf("%v:%v", w.LastRequest.Hour(), w.LastRequest.Minute())
	return fmt.Sprintf("🚶 %s Сейчас %.1fC и ветер %.1f м/c. По состоянию на %s",
		text, w.Main.Temp, w.Wind.Speed, time)
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

	if w, ok := weatherCache.Get(WEATHER); ok {
		go func(ch <-chan Weather) {
			defer close(out)
			out <- w.(Weather)
		}(out)

		return out
	}

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

		weather.LastRequest = time.Now()

		weatherCache.Set(WEATHER, weather, WEATHER_EXPIRATION)

		out <- weather

		return nil
	})

	return out
}

func main() {
	logCache = cache.New(ERROR_EXPIRATION, 15*time.Minute)
	weatherCache = cache.New(WEATHER_EXPIRATION, 5*time.Minute)

	cfg := &Config{
		Url:   os.Getenv("WEATHER_URL"),
		APPID: os.Getenv("WEATHER_APPID"),
		Lat:   os.Getenv("WEATHER_LAT"),
		Lon:   os.Getenv("WEATHER_LON"),
	}

	err := cfg.Check()
	if err != nil {
		log.Fatal(err)
		return
	}

	botPref := tele.Settings{
		Token:  os.Getenv("WEATHER_BOT_TOKEN"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := tele.NewBot(botPref)
	if err != nil {
		log.Fatal(err)
		return
	}

	menu := &tele.ReplyMarkup{ResizeKeyboard: true}
	selector := &tele.ReplyMarkup{}
	btnCheck := menu.Text("Прогулка на текущее время")

	menu.Reply(
		menu.Row(btnCheck),
	)

	selector.Inline(
		selector.Row(btnCheck),
	)

	bot.Handle("/start", func(c tele.Context) error {
		return c.Send("☀️ Добро пожаловать!\nЭтот бот помогает определить, будет ли прогулка в детском саду \"Гусельки\".\n"+
			"Нажмите на кнопку \"Прогулка на текущее время\" в меню для информации о прогулке.", menu)
	})

	bot.Handle(&btnCheck, func(c tele.Context) error {
		errGroup := new(errgroup.Group)

		defer func() {
			if err := errGroup.Wait(); err != nil {
				addLogError(err)
			}
		}()

		weatherChan := getWeather(errGroup, cfg)
		if weather, ok := <-weatherChan; ok {
			_, text := walkResult(&weather)
			return c.Send(text, menu)
		}

		return nil
	})

	bot.Handle("/errors", func(c tele.Context) error {
		return c.Send(getErrors(), menu)
	})

	bot.Start()
}

func addLogError(err error) {
	str, ok := logCache.Get(ERRORS)
	if !ok {
		logCache.Add(ERRORS, err.Error(), ERROR_EXPIRATION)
		return
	}

	text := str.(string)
	text = fmt.Sprintf("%s\n%s", text, err.Error())
	logCache.Set(ERRORS, text, ERROR_EXPIRATION)
}

func getErrors() string {
	errors, ok := logCache.Get(ERRORS)
	if !ok {
		return "erros list is empty"
	}

	return errors.(string)
}
