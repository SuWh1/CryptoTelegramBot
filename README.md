# Crypto Bot

This is a simple Telegram bot that fetches real-time cryptocurrency data using the CoinGecko API. Users can type the name of any cryptocurrency to get its current price and 24-hour price change.

## How to Run

### 1. Clone the Repository

First, clone this repository to your local machine:

```bash
git clone https://github.com/your-username/crypto-bot.git
cd crypto-bot
```

### 2. Install Dependencies
Make sure you have Go installed on your machine. If you haven't installed it yet, you can download it from the official Go website.

After cloning the repository, install the required dependencies:

```bash
go mod tidy
```

### 3. Set Up the API Key
Create a file named .env in the root directory of the project and add your Telegram Bot API token:
```bash
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
```
Replace your_telegram_bot_token with the token you received from BotFather when you created your bot.
