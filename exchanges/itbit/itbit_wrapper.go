package itbit

import (
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the ItBit go routine
func (i *ItBit) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		i.Run()
		wg.Done()
	}()
}

// Run implements the ItBit wrapper
func (i *ItBit) Run() {
	if i.Verbose {
		log.Printf("%s polling delay: %ds.\n", i.GetName(), i.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", i.GetName(), len(i.EnabledPairs), i.EnabledPairs)
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (i *ItBit) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := i.GetTicker(exchange.FormatExchangeCurrency(i.Name,
		p).String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	tickerPrice.Last = tick.LastPrice
	tickerPrice.High = tick.High24h
	tickerPrice.Low = tick.Low24h
	tickerPrice.Volume = tick.Volume24h
	ticker.ProcessTicker(i.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(i.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (i *ItBit) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(i.GetName(), p, assetType)
	if err != nil {
		return i.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (i *ItBit) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(i.GetName(), p, assetType)
	if err != nil {
		return i.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (i *ItBit) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := i.GetOrderbook(exchange.FormatExchangeCurrency(i.Name,
		p).String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		price, err := strconv.ParseFloat(data[0], 64)
		if err != nil {
			log.Println(err)
		}
		amount, err := strconv.ParseFloat(data[1], 64)
		if err != nil {
			log.Println(err)
		}
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: amount, Price: price})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		price, err := strconv.ParseFloat(data[0], 64)
		if err != nil {
			log.Println(err)
		}
		amount, err := strconv.ParseFloat(data[1], 64)
		if err != nil {
			log.Println(err)
		}
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: amount, Price: price})
	}

	orderbook.ProcessOrderbook(i.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(i.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
//ItBit exchange - to-do
func (i *ItBit) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = i.GetName()
	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (i *ItBit) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (i *ItBit) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (i *ItBit) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var wallet string

	wallets, err := i.GetWallets(nil)
	if err != nil {
		return submitOrderResponse, err
	}

	// Determine what wallet ID to use if there is any actual available currency to make the trade!
	for _, i := range wallets {
		for j := range i.Balances {
			if i.Balances[j].Currency == p.FirstCurrency.String() && i.Balances[j].AvailableBalance >= amount {
				wallet = i.ID
			}
		}
	}

	if wallet == "" {
		return submitOrderResponse, fmt.Errorf("No wallet found with currency: %s with amount >= %v", p.FirstCurrency.String(), amount)
	}

	response, err := i.PlaceOrder(wallet, side.ToString(), orderType.ToString(), p.FirstCurrency.String(), amount, price, p.Pair().String(), "")

	if response.ID != "" {
		submitOrderResponse.OrderID = response.ID
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (i *ItBit) ModifyOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (i *ItBit) CancelOrder(order exchange.OrderCancellation) error {
	return i.CancelExistingOrder(order.WalletAddress, order.OrderID)
}

// CancelAllOrders cancels all orders associated with a currency pair
func (i *ItBit) CancelAllOrders(orderCancellation exchange.OrderCancellation) error {
	openOrders, err := i.GetOrders(orderCancellation.WalletAddress, "", "open", 0, 0)

	if err != nil {
		return err
	}

	for _, openOrder := range openOrders {
		err = i.CancelExistingOrder(orderCancellation.WalletAddress, openOrder.ID)

		if err != nil {
			return err
		}
	}

	return nil
}

// GetOrderInfo returns information on a current open order
func (i *ItBit) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (i *ItBit) GetDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (i *ItBit) WithdrawCryptocurrencyFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (i *ItBit) WithdrawFiatFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (i *ItBit) WithdrawFiatFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (i *ItBit) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (i *ItBit) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return i.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (i *ItBit) GetWithdrawCapabilities() uint32 {
	return i.GetWithdrawPermissions()
}
