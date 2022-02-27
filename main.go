package main

import (
	"fmt"
	"log"
	"os"
	"time"

	cache "github.com/patrickmn/go-cache"
	"golang.org/x/sync/errgroup"
	tele "gopkg.in/telebot.v3"
)

const (
	ERRORS           string        = "errors"
	ERROR_EXPIRATION time.Duration = 24 * time.Hour
)

var logCache *cache.Cache

func main() {
	logCache = cache.New(ERROR_EXPIRATION, 15*time.Minute)

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
