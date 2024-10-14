package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CryptoData struct {
	ID     string
	Price  float64
	Change float64
}

func getTopCryptos() (map[string]CryptoData, error) {
	resp, err := http.Get("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=20&page=1&sparkline=false")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result []struct {
		ID     string  `json:"id"`
		Name   string  `json:"name"`
		Price  float64 `json:"current_price"`
		Change float64 `json:"price_change_percentage_24h"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	cryptos := make(map[string]CryptoData)
	for _, crypto := range result {
		cryptos[crypto.Name] = CryptoData{
			ID:     crypto.ID,
			Price:  crypto.Price,
			Change: crypto.Change,
		}
	}

	return cryptos, nil
}

func getCryptoByName(name string) (*CryptoData, error) {
	apiURL := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&ids=%s&order=market_cap_desc&per_page=1&page=1&sparkline=false", name)
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result []struct {
		Name   string  `json:"id"`
		Price  float64 `json:"current_price"`
		Change float64 `json:"price_change_percentage_24h"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("crypto not found")
	}

	crypto := CryptoData{
		Price:  result[0].Price,
		Change: result[0].Change,
	}
	return &crypto, nil
}

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("Token is not set")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	cryptos, err := getTopCryptos()
	if err != nil {
		log.Fatalf("Error fetching crypto data: %v", err)
	}

	for update := range updates {
		if update.Message != nil {
			switch update.Message.Text {
			case "/start":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "👋 Welcome! Choose a top 20 of cryptocurrencies, or type the name of any cryptocurrency to get the info about it.✨")

				var keyboardRows [][]tgbotapi.InlineKeyboardButton
				for name := range cryptos {
					button := tgbotapi.NewInlineKeyboardButtonData(name, name)
					keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(button))
				}

				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
				bot.Send(msg)

			default:
				cryptoName := update.Message.Text
				var cryptoData *CryptoData

				if data, exists := cryptos[cryptoName]; exists {
					cryptoData = &data
				} else {
					var err error
					cryptoData, err = getCryptoByName(strings.ToLower(cryptoName))
					if err != nil {
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Cryptocurrency not found. Please check the name and try again.")
						bot.Send(msg)
						continue
					}
				}

				sign := "📈"
				if cryptoData.Change < 0 {
					sign = "📉"
				}

				msgText := fmt.Sprintf("💰 *%s*\n💵 Price: *$%.2f*\n📊 24h Change: *%s%.2f%%*",
					cryptoName, cryptoData.Price, sign, cryptoData.Change)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
				msg.ParseMode = "Markdown"
				bot.Send(msg)
			}
		} else if update.CallbackQuery != nil {
			cryptoName := update.CallbackQuery.Data
			if cryptoData, exists := cryptos[cryptoName]; exists {
				sign := "📈"
				if cryptoData.Change < 0 {
					sign = "📉"
				}

				msgText := fmt.Sprintf("💰 *%s*\n💵 Price: *$%.2f*\n📊 24h Change: *%s%.2f%%*",
					cryptoName, cryptoData.Price, sign, cryptoData.Change)
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, msgText)
				msg.ParseMode = "Markdown"
				bot.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "❌ Cryptocurrency data not found.")
				bot.Send(msg)
			}
		}
	}
}
