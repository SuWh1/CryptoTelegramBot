package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"unicode"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CryptoData struct {
	ID     string
	Price  float64
	Change float64
}

type CryptoListItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func Capitalize(name string) string {
	words := strings.Fields(name)

	for i, word := range words {
		if len(word) > 0 {
			words[i] = string(unicode.ToUpper(rune(word[0]))) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

var cryptoList []CryptoListItem

func fetchCryptoList() error {
	resp, err := http.Get("https://api.coingecko.com/api/v3/coins/list")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&cryptoList); err != nil {
		return err
	}
	return nil
}

func getCryptoDataByID(id string) (*CryptoData, error) {
	apiURL := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&ids=%s&order=market_cap_desc&per_page=1&page=1&sparkline=false", id)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Println("Market API Response:", string(body))

	var result []struct {
		ID     string  `json:"id"`
		Price  float64 `json:"current_price"`
		Change float64 `json:"price_change_percentage_24h"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("crypto not found")
	}

	return &CryptoData{
		ID:     result[0].ID,
		Price:  result[0].Price,
		Change: result[0].Change,
	}, nil
}

func findCryptoIDByName(name string) (string, error) {
	lowerName := strings.ToLower(name)
	for _, crypto := range cryptoList {
		if strings.ToLower(crypto.Name) == lowerName {
			return crypto.ID, nil
		}
	}
	return "", fmt.Errorf("cryptocurrency not found")
}

func sendTop5CryptoOptions(bot *tgbotapi.BotAPI, chatID int64) {
	resp, err := http.Get("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=5&page=1&sparkline=false")
	if err != nil {
		log.Println("Error fetching top 5 cryptos:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return
	}

	log.Println("API Response:", string(body))

	type MarketData struct {
		ID     string  `json:"id"`
		Name   string  `json:"name"`
		Price  float64 `json:"current_price"`
		Change float64 `json:"price_change_percentage_24h"`
	}

	var result []MarketData
	if err := json.Unmarshal(body, &result); err != nil {
		log.Println("Error decoding API response:", err)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "OR choose a cryptocurrency from the top 5 list 👇")
	var keyboardRows [][]tgbotapi.InlineKeyboardButton

	for _, crypto := range result {
		button := tgbotapi.NewInlineKeyboardButtonData(crypto.Name, crypto.ID)
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(button))
	}

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
	if _, err := bot.Send(msg); err != nil {
		log.Println("Error sending top 5 crypto options:", err)
	}
}

func handleCallbackQuery(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery) {
	callback := tgbotapi.NewCallback(callbackQuery.ID, "")
	if _, err := bot.Request(callback); err != nil {
		log.Println("Error acknowledging callback query:", err)
		return
	}

	cryptoID := callbackQuery.Data
	cryptoData, err := getCryptoDataByID(cryptoID)
	if err != nil {
		log.Println("Error fetching cryptocurrency data:", err)
		bot.Send(tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "❌ Error fetching data for this cryptocurrency."))
		return
	}

	msgText := fmt.Sprintf("💰 *%s*\n💵 Price: *$%.2f*\n📊 24h Change: *%+.2f%%*", Capitalize(cryptoID), cryptoData.Price, cryptoData.Change)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, msgText)
	msg.ParseMode = "Markdown"

	if _, err := bot.Send(msg); err != nil {
		log.Println("Error sending crypto information:", err)
	}
}

func main() {
	if err := fetchCryptoList(); err != nil {
		log.Fatalf("Error fetching crypto list: %v", err)
	}

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

	for update := range updates {
		if update.Message != nil {
			if update.Message.Text == "/start" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "👋 Welcome to crypto_bot! Type the name of any cryptocurrency to get its info.")
				bot.Send(msg)
				sendTop5CryptoOptions(bot, update.Message.Chat.ID)
			} else {
				cryptoName := update.Message.Text

				cryptoID, err := findCryptoIDByName(cryptoName)
				if err != nil {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Cryptocurrency not found. Please check the name and try again.")
					bot.Send(msg)
					continue
				}

				cryptoData, err := getCryptoDataByID(cryptoID)
				if err != nil {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Error fetching cryptocurrency data.")
					bot.Send(msg)
					continue
				}

				sign := "📈"
				if cryptoData.Change < 0 {
					sign = "📉"
				}

				msgText := fmt.Sprintf("💰 *%s*\n💵 Price: *$%.2f*\n📊 24h Change: *%s%.2f%%*",
					Capitalize(cryptoName), cryptoData.Price, sign, cryptoData.Change)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
				msg.ParseMode = "Markdown"
				bot.Send(msg)
			}
		}

		if update.CallbackQuery != nil {
			handleCallbackQuery(bot, update.CallbackQuery)
		}
	}
}
