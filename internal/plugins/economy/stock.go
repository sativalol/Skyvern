package economy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
	"github.com/bwmarrin/discordgo"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/storage"
)

type Stock struct {
	Symbol       string
	Price        float64
	OpenPrice    float64
	PriceHistory []float64
}

type YFQuoteResponse struct {
	QuoteResponse struct {
		Result []struct {
			Symbol                     string  `json:"symbol"`
			LongName                   string  `json:"longName"`
			RegularMarketPrice         float64 `json:"regularMarketPrice"`
			RegularMarketChangePercent float64 `json:"regularMarketChangePercent"`
			RegularMarketPreviousClose float64 `json:"regularMarketPreviousClose"`
			RegularMarketOpen          float64 `json:"regularMarketOpen"`
		} `json:"result"`
	} `json:"quoteResponse"`
}

type YFChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol        string  `json:"symbol"`
				PreviousClose float64 `json:"previousClose"`
			} `json:"meta"`
			Indicators struct {
				Quote []struct {
					Close []float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}

var (
	yfCrumb  string
	yfJar, _ = cookiejar.New(nil)
	yfClient = &http.Client{
		Timeout: 5 * time.Second,
		Jar:     yfJar,
	}
)

func initYFinanceSession() error {
	req, err := http.NewRequest("GET", "https://fc.yahoo.com", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	resp, err := yfClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	req, err = http.NewRequest("GET", "https://query2.finance.yahoo.com/v1/test/getcrumb", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	resp, err = yfClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get crumb: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	yfCrumb = string(bytes.TrimSpace(body))
	return nil
}

func fetchYFQuote(symbols []string) (*YFQuoteResponse, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("empty symbols")
	}
	if yfCrumb == "" {
		_ = initYFinanceSession()
	}
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v7/finance/quote?symbols=%s", strings.Join(symbols, ","))
	if yfCrumb != "" {
		url += fmt.Sprintf("&crumb=%s", yfCrumb)
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	resp, err := yfClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		yfCrumb = ""
		return nil, fmt.Errorf("http error %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error %d", resp.StatusCode)
	}

	var res YFQuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	if len(res.QuoteResponse.Result) == 0 {
		return nil, fmt.Errorf("no quote results")
	}
	return &res, nil
}

func getQuoteWithRetry(symbols []string) (*YFQuoteResponse, error) {
	res, err := fetchYFQuote(symbols)
	if err != nil && (strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") || yfCrumb == "") {
		if err = initYFinanceSession(); err == nil {
			return fetchYFQuote(symbols)
		}
	}
	return res, err
}

func fetchYFChart(symbol string) ([]float64, float64, error) {
	if yfCrumb == "" {
		_ = initYFinanceSession()
	}
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v8/finance/chart/%s?range=1d&interval=15m", symbol)
	if yfCrumb != "" {
		url += fmt.Sprintf("&crumb=%s", yfCrumb)
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	resp, err := yfClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		yfCrumb = ""
		return nil, 0, fmt.Errorf("http error %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("http error %d", resp.StatusCode)
	}

	var res YFChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, 0, err
	}
	if len(res.Chart.Result) == 0 {
		return nil, 0, fmt.Errorf("no chart result")
	}
	resObj := res.Chart.Result[0]
	if len(resObj.Indicators.Quote) == 0 {
		return nil, 0, fmt.Errorf("no chart quote data")
	}

	rawClose := resObj.Indicators.Quote[0].Close
	var history []float64
	for _, val := range rawClose {
		if val > 0 && !math.IsNaN(val) {
			history = append(history, val)
		}
	}
	return history, resObj.Meta.PreviousClose, nil
}

func getChartWithRetry(symbol string) ([]float64, float64, error) {
	history, prevClose, err := fetchYFChart(symbol)
	if err != nil && (strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") || yfCrumb == "") {
		if err = initYFinanceSession(); err == nil {
			return fetchYFChart(symbol)
		}
	}
	return history, prevClose, err
}

func countHolders(db *storage.DB, gid, sym string) int {
	count := 0
	_ = db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bktEcoAccts)
		if bkt == nil {
			return nil
		}
		c := bkt.Cursor()
		prefix := []byte(gid + ":")
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			var a EcoAccount
			if json.Unmarshal(v, &a) == nil {
				if shares, ok := a.Stocks[sym]; ok && shares > 0 {
					count++
				}
			}
		}
		return nil
	})
	return count
}

func getChangeStr(cur, prev float64) string {
	diff := cur - prev
	pct := (diff / prev) * 100.0
	if diff >= 0 {
		return fmt.Sprintf("+%.2f%%", pct)
	}
	return fmt.Sprintf("-%.2f%%", math.Abs(pct))
}

func stockCmds(p *EconomyPlugin) []*manager.Command {
	return []*manager.Command{
		{
			Trigger:     "stock",
			Aliases:     []string{"shares", "stocks", "stonks"},
			Name:        "stock",
			Description: "Trade actual stocks and cryptos with real-time charts using your wallet balance",
			Category:    "economy",
			Execute: func(ctx *manager.CommandContext) error {
				gid := ctx.GuildID()
				if gid == "" {
					return ctx.Reply("Must be used in a server.")
				}
				cfg := getCfg(ctx.DB, gid)
				if !cfg.Enabled {
					return ctx.Reply("Economy is disabled in this server.")
				}

				if len(ctx.Args) == 0 {
					return listStocks(ctx, cfg)
				}

				sub := strings.ToLower(ctx.Args[0])
				switch sub {
				case "list":
					return listStocks(ctx, cfg)
				case "view":
					return viewStock(ctx, cfg)
				case "buy":
					return buyStock(ctx, cfg)
				case "sell":
					return sellStock(ctx, cfg)
				case "portfolio", "port", "bal":
					return viewPortfolio(ctx, cfg)
				default:
					// fallback to view a specific symbol
					return viewStock(ctx, cfg)
				}
			},
		},
	}
}

func listStocks(ctx *manager.CommandContext, cfg EcoCfg) error {
	watchlist := []string{"AAPL", "MSFT", "NVDA", "TSLA", "BTC-USD", "ETH-USD"}
	res, err := getQuoteWithRetry(watchlist)
	if err != nil {
		return ctx.Reply(fmt.Sprintf("Failed to fetch stock data: %v", err))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-10s | %-12s | %-10s\n", "Symbol", "Price", "Change"))
	sb.WriteString(strings.Repeat("-", 38) + "\n")

	for _, s := range res.QuoteResponse.Result {
		change := s.RegularMarketPrice - s.RegularMarketPreviousClose
		pct := (change / s.RegularMarketPreviousClose) * 100.0
		sign := "+"
		if change < 0 {
			sign = ""
		}
		sb.WriteString(fmt.Sprintf("%-10s | $%-11.2f | %s%.2f%%\n",
			s.Symbol, s.RegularMarketPrice, sign, pct))
	}

	emb := config.Build(ctx.Cfg, config.EmbedOpt{
		Title:       "Live Stock Market Watchlist",
		Description: fmt.Sprintf("```\n%s```", sb.String()),
	})
	emb.Footer.Text = "Use .stock view <symbol> | Prices powered by Yahoo Finance"
	return ctx.Respond(emb)
}

func viewStock(ctx *manager.CommandContext, cfg EcoCfg) error {
	sym := ""
	if len(ctx.Args) > 1 {
		sym = strings.ToUpper(ctx.Args[1])
	} else if len(ctx.Args) == 1 {
		sym = strings.ToUpper(ctx.Args[0])
		if sym == "LIST" || sym == "PORTFOLIO" || sym == "PORT" || sym == "BAL" || sym == "BUY" || sym == "SELL" {
			return ctx.SendHelp("stock")
		}
	} else {
		return ctx.SendHelp("stock")
	}

	res, err := getQuoteWithRetry([]string{sym})
	if err != nil || len(res.QuoteResponse.Result) == 0 {
		return ctx.Reply(fmt.Sprintf("Stock symbol %s not found.", sym))
	}
	s := res.QuoteResponse.Result[0]

	history, prevClose, err := getChartWithRetry(sym)
	if err != nil || len(history) == 0 {
		history = []float64{s.RegularMarketPreviousClose, s.RegularMarketPrice}
		prevClose = s.RegularMarketPreviousClose
	}

	change := s.RegularMarketPrice - s.RegularMarketPreviousClose
	pct := (change / s.RegularMarketPreviousClose) * 100.0
	sign := "+"
	if change < 0 {
		sign = ""
	}

	name := s.LongName
	if name == "" {
		name = s.Symbol
	}

	holders := countHolders(ctx.DB, ctx.GuildID(), sym)
	uid := ctx.AuthorID()
	a := getAcct(ctx.DB, ctx.GuildID(), uid)
	sharesOwned := a.Stocks[sym]
	valueOwned := float64(sharesOwned) * s.RegularMarketPrice

	emb := config.Build(ctx.Cfg, config.EmbedOpt{
		Title: fmt.Sprintf("%s - %s", s.Symbol, name),
		Fields: []*discordgo.MessageEmbedField{
			config.Field("Market Price", fmt.Sprintf("$%.2f (%s%.2f%%)", s.RegularMarketPrice, sign, pct), true),
			config.Field("Market Open", fmt.Sprintf("$%.2f", s.RegularMarketOpen), true),
			config.Field("Previous Close", fmt.Sprintf("$%.2f", prevClose), true),
			config.Field("Your Position", fmt.Sprintf("%.4f Shares (Value: %s)", sharesOwned, fmtCoins(int64(valueOwned), cfg)), true),
			config.Field("Available Cash", fmtCoins(a.Wallet, cfg), true),
			config.Field("Holders (Server)", strconv.Itoa(holders), true),
		},
		ImageURL: "attachment://chart.png",
	})

	stockObj := &Stock{
		Symbol:       s.Symbol,
		Price:        s.RegularMarketPrice,
		OpenPrice:    prevClose,
		PriceHistory: history,
	}

	chartBytes := drawChart(stockObj)
	_, err = ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{emb},
		Files: []*discordgo.File{
			{
				Name:        "chart.png",
				ContentType: "image/png",
				Reader:      bytes.NewReader(chartBytes),
			},
		},
	})
	return err
}

func buyStock(ctx *manager.CommandContext, cfg EcoCfg) error {
	if len(ctx.Args) < 3 {
		return ctx.SendHelp("stock")
	}

	sym := strings.ToUpper(ctx.Args[1])
	res, err := getQuoteWithRetry([]string{sym})
	if err != nil || len(res.QuoteResponse.Result) == 0 {
		return ctx.Reply(fmt.Sprintf("Stock symbol %s not found.", sym))
	}
	s := res.QuoteResponse.Result[0]
	if s.RegularMarketPrice <= 0 {
		return ctx.Reply("Invalid stock price. Cannot trade this symbol right now.")
	}

	uid := ctx.AuthorID()
	gid := ctx.GuildID()
	a := getAcct(ctx.DB, gid, uid)

	var sh float64
	argLower := strings.ToLower(ctx.Args[2])
	if argLower == "all" || argLower == "max" {
		sh = float64(a.Wallet) / s.RegularMarketPrice
		sh = math.Floor(sh*10000.0) / 10000.0
		// calculate cost + fee and loop decrement
		for sh > 0 {
			cost := int64(math.Ceil(s.RegularMarketPrice * sh))
			fee := int64(math.Ceil(float64(cost) * 0.01))
			if cost+fee <= a.Wallet {
				break
			}
			sh -= 0.0001
		}
		if sh <= 0 {
			return ctx.Reply("You do not have enough funds to buy even 0.0001 shares of this stock including the 1% broker fee.")
		}
	} else {
		parsed, err := strconv.ParseFloat(ctx.Args[2], 64)
		if err != nil || parsed <= 0 || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return ctx.Reply("Please enter a valid positive number of shares.")
		}
		sh = parsed
	}

	rawCost := s.RegularMarketPrice * sh
	if rawCost >= float64(math.MaxInt64-1) || math.IsNaN(rawCost) || math.IsInf(rawCost, 0) {
		return ctx.Reply("The total cost is too high or invalid.")
	}
	cost := int64(math.Ceil(rawCost))
	if cost <= 0 {
		return ctx.Reply("The number of shares is too small; total cost must be at least 1 coin.")
	}

	fee := int64(math.Ceil(float64(cost) * 0.01))
	totalCost := cost + fee

	if a.Wallet < totalCost {
		return ctx.Reply(fmt.Sprintf("Insufficient funds to cover the purchase cost + 1%% broker commission. Total Needed: %s (Cost: %s + Fee: %s) | Wallet: %s",
			fmtCoins(totalCost, cfg), fmtCoins(cost, cfg), fmtCoins(fee, cfg), fmtCoins(a.Wallet, cfg)))
	}

	a.Wallet -= totalCost
	if a.Stocks == nil {
		a.Stocks = make(map[string]float64)
	}
	a.Stocks[sym] += sh

	_ = saveAcct(ctx.DB, gid, uid, a)

	return ctx.Reply(fmt.Sprintf("Successfully bought %.4f shares of %s at %s.\nBroker Commission (1%%): %s | Total Cost: %s\nNew Wallet Balance: %s | Total Position: %.4f shares.",
		sh, sym, fmtCoins(int64(s.RegularMarketPrice), cfg), fmtCoins(fee, cfg), fmtCoins(totalCost, cfg), fmtCoins(a.Wallet, cfg), a.Stocks[sym]))
}

func sellStock(ctx *manager.CommandContext, cfg EcoCfg) error {
	if len(ctx.Args) < 3 {
		return ctx.SendHelp("stock")
	}

	sym := strings.ToUpper(ctx.Args[1])
	res, err := getQuoteWithRetry([]string{sym})
	if err != nil || len(res.QuoteResponse.Result) == 0 {
		return ctx.Reply(fmt.Sprintf("Stock symbol %s not found.", sym))
	}
	s := res.QuoteResponse.Result[0]
	if s.RegularMarketPrice <= 0 {
		return ctx.Reply("Invalid stock price. Cannot trade this symbol right now.")
	}

	uid := ctx.AuthorID()
	gid := ctx.GuildID()
	a := getAcct(ctx.DB, gid, uid)

	owned := a.Stocks[sym]
	var sh float64
	argLower := strings.ToLower(ctx.Args[2])
	if argLower == "all" || argLower == "max" {
		sh = owned
		if sh <= 0 {
			return ctx.Reply(fmt.Sprintf("You do not own any shares of %s.", sym))
		}
	} else {
		parsed, err := strconv.ParseFloat(ctx.Args[2], 64)
		if err != nil || parsed <= 0 || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return ctx.Reply("Please enter a valid positive number of shares.")
		}
		sh = parsed
	}

	if owned < sh {
		return ctx.Reply(fmt.Sprintf("You do not own enough shares. Owned: %.4f | Selling: %.4f", owned, sh))
	}

	rawPayout := s.RegularMarketPrice * sh
	if rawPayout >= float64(math.MaxInt64-1) || math.IsNaN(rawPayout) || math.IsInf(rawPayout, 0) {
		return ctx.Reply("The total payout is too high or invalid.")
	}
	payout := int64(math.Floor(rawPayout))

	fee := int64(math.Ceil(float64(payout) * 0.01))
	netPayout := payout - fee

	if netPayout <= 0 {
		return ctx.Reply("The number of shares is too small; net payout must be at least 1 coin after 1% broker commission.")
	}

	a.Stocks[sym] -= sh
	if a.Stocks[sym] <= 0 {
		delete(a.Stocks, sym)
	}

	a.Wallet += netPayout

	_ = saveAcct(ctx.DB, gid, uid, a)

	return ctx.Reply(fmt.Sprintf("Successfully sold %.4f shares of %s at %s.\nBroker Commission (1%%): %s | Net Payout: %s\nNew Wallet Balance: %s | Remaining Position: %.4f shares.",
		sh, sym, fmtCoins(int64(s.RegularMarketPrice), cfg), fmtCoins(fee, cfg), fmtCoins(netPayout, cfg), fmtCoins(a.Wallet, cfg), a.Stocks[sym]))
}

func viewPortfolio(ctx *manager.CommandContext, cfg EcoCfg) error {
	uid := ctx.AuthorID()
	gid := ctx.GuildID()
	a := getAcct(ctx.DB, gid, uid)

	var symbols []string
	for sym, sh := range a.Stocks {
		if sh > 0 {
			symbols = append(symbols, sym)
		}
	}

	prices := make(map[string]float64)
	prevCloses := make(map[string]float64)

	if len(symbols) > 0 {
		res, err := getQuoteWithRetry(symbols)
		if err == nil {
			for _, item := range res.QuoteResponse.Result {
				prices[item.Symbol] = item.RegularMarketPrice
				prevCloses[item.Symbol] = item.RegularMarketPreviousClose
			}
		}
	}

	var holdingsText []string
	var totalStockValue float64

	for sym, sh := range a.Stocks {
		if sh <= 0 {
			continue
		}
		prc := prices[sym]
		val := sh * prc
		totalStockValue += val
		prev := prevCloses[sym]
		chg := ""
		if prev > 0 {
			chg = getChangeStr(prc, prev)
		}
		holdingsText = append(holdingsText, fmt.Sprintf("**%s** - %.4f shares (Value: %s) %s", sym, sh, fmtCoins(int64(val), cfg), chg))
	}

	desc := strings.Join(holdingsText, "\n")
	if desc == "" {
		desc = "*You do not own any stocks in this server.*"
	}

	totalNet := a.Wallet + a.Bank + int64(totalStockValue)

	emb := config.Build(ctx.Cfg, config.EmbedOpt{
		Title:       fmt.Sprintf("%s's Investment Portfolio", ctx.AuthorTag()),
		Description: desc,
		Fields: []*discordgo.MessageEmbedField{
			config.Field("Cash (Wallet)", fmtCoins(a.Wallet, cfg), true),
			config.Field("Stocks Value", fmtCoins(int64(totalStockValue), cfg), true),
			config.Field("Total Net Worth", fmtCoins(totalNet, cfg), true),
		},
	})
	return ctx.Respond(emb)
}

func absVal(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, col color.Color) {
	dx := absVal(x1 - x0)
	dy := absVal(y1 - y0)
	sx, sy := 1, 1
	if x0 >= x1 {
		sx = -1
	}
	if y0 >= y1 {
		sy = -1
	}
	err := dx - dy
	for {
		for io := -1; io <= 1; io++ {
			for jo := -1; jo <= 1; jo++ {
				cx, cy := x0+io, y0+jo
				if cx >= 0 && cx < img.Bounds().Dx() && cy >= 0 && cy < img.Bounds().Dy() {
					img.Set(cx, cy, col)
				}
			}
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func drawChart(s *Stock) []byte {
	w, h := 400, 150
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	bg := color.RGBA{0x1e, 0x1f, 0x22, 0xff}
	draw.Draw(img, img.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)
	history := s.PriceHistory
	if len(history) < 2 {
		var buf bytes.Buffer
		_ = png.Encode(&buf, img)
		return buf.Bytes()
	}
	minVal, maxVal := history[0], history[0]
	for _, val := range history {
		if val < minVal {
			minVal = val
		}
		if val > maxVal {
			maxVal = val
		}
	}
	diff := maxVal - minVal
	if diff == 0 {
		diff = 1.0
	}
	var pts []image.Point
	for i, val := range history {
		x := i * w / (len(history) - 1)
		y := h - 15 - int((val-minVal)/diff*float64(h-30))
		pts = append(pts, image.Pt(x, y))
	}
	var lineCol color.Color
	if s.Price >= s.OpenPrice {
		lineCol = color.RGBA{0x23, 0xa5, 0x5a, 0xff}
	} else {
		lineCol = color.RGBA{0xf2, 0x3f, 0x43, 0xff}
	}
	for i := 0; i < len(pts)-1; i++ {
		drawLine(img, pts[i].X, pts[i].Y, pts[i+1].X, pts[i+1].Y, lineCol)
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
