package exchange

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	warningBase64DecryptSecretKeyFailed = "WARNING -- Exchange %s unable to base64 decode secret key.. Disabling Authenticated API support."
	// WarningAuthenticatedRequestWithoutCredentialsSet error message for authenticated request without credentials set
	WarningAuthenticatedRequestWithoutCredentialsSet = "WARNING -- Exchange %s authenticated HTTP request called but not supported due to unset/default API keys."
	// ErrExchangeNotFound is a stand for an error message
	ErrExchangeNotFound = "Exchange not found in dataset"
	// DefaultHTTPTimeout is the default HTTP/HTTPS Timeout for exchange requests
	DefaultHTTPTimeout = time.Second * 15
)

// FeeType custom type for calculating fees based on method
type FeeType string

// InternationalBankTransactionType custom type for calculating fees based on fiat transaction types
type InternationalBankTransactionType string

// Const declarations for fee types
const (
	BankFee                        FeeType = "bankFee"
	InternationalBankDepositFee    FeeType = "internationalBankDepositFee"
	InternationalBankWithdrawalFee FeeType = "internationalBankWithdrawalFee"
	CryptocurrencyTradeFee         FeeType = "cryptocurrencyTradeFee"
	CyptocurrencyDepositFee        FeeType = "cyptocurrencyDepositFee"
	CryptocurrencyWithdrawalFee    FeeType = "cryptocurrencyWithdrawalFee"
)

// Const declarations for international transaction types
const (
	WireTransfer    InternationalBankTransactionType = "wireTransfer"
	PerfectMoney    InternationalBankTransactionType = "perfectMoney"
	Neteller        InternationalBankTransactionType = "neteller"
	AdvCash         InternationalBankTransactionType = "advCash"
	Payeer          InternationalBankTransactionType = "payeer"
	Skrill          InternationalBankTransactionType = "skrill"
	Simplex         InternationalBankTransactionType = "simplex"
	SEPA            InternationalBankTransactionType = "sepa"
	Swift           InternationalBankTransactionType = "swift"
	RapidTransfer   InternationalBankTransactionType = "rapidTransfer"
	MisterTangoSEPA InternationalBankTransactionType = "misterTangoSepa"
	Qiwi            InternationalBankTransactionType = "qiwi"
	VisaMastercard  InternationalBankTransactionType = "visaMastercard"
	WebMoney        InternationalBankTransactionType = "webMoney"
	Capitalist      InternationalBankTransactionType = "capitalist"
	WesternUnion    InternationalBankTransactionType = "westernUnion"
	MoneyGram       InternationalBankTransactionType = "moneyGram"
	Contact         InternationalBankTransactionType = "contact"
)

// SubmitOrderResponse is what is returned after submitting an order to an exchange
type SubmitOrderResponse struct {
	IsOrderPlaced bool
	OrderID       string
}

// FeeBuilder is the type which holds all parameters required to calculate a fee for an exchange
type FeeBuilder struct {
	FeeType FeeType
	//Used for calculating crypto trading fees, deposits & withdrawals
	FirstCurrency  string
	SecondCurrency string
	Delimiter      string
	IsMaker        bool
	// Fiat currency used for bank deposits & withdrawals
	CurrencyItem        string
	BankTransactionType InternationalBankTransactionType
	// Used to multiply for fee calculations
	PurchasePrice float64
	Amount        float64
}

// OrderCancellation type requred when requesting to cancel an order
type OrderCancellation struct {
	AccountID     string
	OrderID       string
	CurrencyPair  pair.CurrencyPair
	WalletAddress string
	Side          OrderSide
}

// Definitions for each type of withdrawal method for a given exchange
const (
	// No withdraw
	NoAPIWithdrawalMethods                  uint32 = 0
	NoAPIWithdrawalMethodsText              string = "NONE, WEBSITE ONLY"
	AutoWithdrawCrypto                      uint32 = (1 << 0)
	AutoWithdrawCryptoWithAPIPermission     uint32 = (1 << 1)
	AutoWithdrawCryptoWithSetup             uint32 = (1 << 2)
	AutoWithdrawCryptoText                  string = "AUTO WITHDRAW CRYPTO"
	AutoWithdrawCryptoWithAPIPermissionText string = "AUTO WITHDRAW CRYPTO WITH API PERMISSION"
	AutoWithdrawCryptoWithSetupText         string = "AUTO WITHDRAW CRYPTO WITH SETUP"
	WithdrawCryptoWith2FA                   uint32 = (1 << 3)
	WithdrawCryptoWithSMS                   uint32 = (1 << 4)
	WithdrawCryptoWithEmail                 uint32 = (1 << 5)
	WithdrawCryptoWithWebsiteApproval       uint32 = (1 << 6)
	WithdrawCryptoWithAPIPermission         uint32 = (1 << 7)
	WithdrawCryptoWith2FAText               string = "WITHDRAW CRYPTO WITH 2FA"
	WithdrawCryptoWithSMSText               string = "WITHDRAW CRYPTO WITH SMS"
	WithdrawCryptoWithEmailText             string = "WITHDRAW CRYPTO WITH EMAIL"
	WithdrawCryptoWithWebsiteApprovalText   string = "WITHDRAW CRYPTO WITH WEBSITE APPROVAL"
	WithdrawCryptoWithAPIPermissionText     string = "WITHDRAW CRYPTO WITH API PERMISSION"
	AutoWithdrawFiat                        uint32 = (1 << 8)
	AutoWithdrawFiatWithAPIPermission       uint32 = (1 << 9)
	AutoWithdrawFiatWithSetup               uint32 = (1 << 10)
	AutoWithdrawFiatText                    string = "AUTO WITHDRAW FIAT"
	AutoWithdrawFiatWithAPIPermissionText   string = "AUTO WITHDRAW FIAT WITH API PERMISSION"
	AutoWithdrawFiatWithSetupText           string = "AUTO WITHDRAW FIAT WITH SETUP"
	WithdrawFiatWith2FA                     uint32 = (1 << 11)
	WithdrawFiatWithSMS                     uint32 = (1 << 12)
	WithdrawFiatWithEmail                   uint32 = (1 << 13)
	WithdrawFiatWithWebsiteApproval         uint32 = (1 << 14)
	WithdrawFiatWithAPIPermission           uint32 = (1 << 15)
	WithdrawFiatWith2FAText                 string = "WITHDRAW FIAT WITH 2FA"
	WithdrawFiatWithSMSText                 string = "WITHDRAW FIAT WITH SMS"
	WithdrawFiatWithEmailText               string = "WITHDRAW FIAT WITH EMAIL"
	WithdrawFiatWithWebsiteApprovalText     string = "WITHDRAW FIAT WITH WEBSITE APPROVAL"
	WithdrawFiatWithAPIPermissionText       string = "WITHDRAW FIAT WITH API PERMISSION"
	WithdrawCryptoViaWebsiteOnly            uint32 = (1 << 16)
	WithdrawFiatViaWebsiteOnly              uint32 = (1 << 17)
	WithdrawCryptoViaWebsiteOnlyText        string = "WITHDRAW CRYPTO VIA WEBSITE ONLY"
	WithdrawFiatViaWebsiteOnlyText          string = "WITHDRAW FIAT VIA WEBSITE ONLY"

	UnknownWithdrawalTypeText string = "UNKNOWN"
)

// AccountInfo is a Generic type to hold each exchange's holdings in
// all enabled currencies
type AccountInfo struct {
	ExchangeName string
	Currencies   []AccountCurrencyInfo
}

// AccountCurrencyInfo is a sub type to store currency name and value
type AccountCurrencyInfo struct {
	CurrencyName string
	TotalValue   float64
	Hold         float64
}

// TradeHistory holds exchange history data
type TradeHistory struct {
	Timestamp int64
	TID       int64
	Price     float64
	Amount    float64
	Exchange  string
	Type      string
}

// OrderDetail holds order detail data
type OrderDetail struct {
	Exchange      string
	ID            string
	BaseCurrency  string
	QuoteCurrency string
	OrderSide     string
	OrderType     string
	CreationTime  int64
	Status        string
	Price         float64
	Amount        float64
	OpenVolume    float64
}

// FundHistory holds exchange funding history data
type FundHistory struct {
	ExchangeName      string
	Status            string
	TransferID        int64
	Description       string
	Timestamp         int64
	Currency          string
	Amount            float64
	Fee               float64
	TransferType      string
	CryptoToAddress   string
	CryptoFromAddress string
	CryptoTxID        string
	BankTo            string
	BankFrom          string
}

// Base stores the individual exchange information
type Base struct {
	Name                                       string
	Enabled                                    bool
	Verbose                                    bool
	RESTPollingDelay                           time.Duration
	AuthenticatedAPISupport                    bool
	APIWithdrawPermissions                     uint32
	APIAuthPEMKeySupport                       bool
	APISecret, APIKey, APIAuthPEMKey, ClientID string
	Nonce                                      nonce.Nonce
	TakerFee, MakerFee, Fee                    float64
	BaseCurrencies                             []string
	AvailablePairs                             []string
	EnabledPairs                               []string
	AssetTypes                                 []string
	PairsLastUpdated                           int64
	SupportsAutoPairUpdating                   bool
	SupportsRESTTickerBatching                 bool
	HTTPTimeout                                time.Duration
	HTTPUserAgent                              string
	WebsocketURL                               string
	APIUrl                                     string
	APIUrlDefault                              string
	APIUrlSecondary                            string
	APIUrlSecondaryDefault                     string
	RequestCurrencyPairFormat                  config.CurrencyPairFormatConfig
	ConfigCurrencyPairFormat                   config.CurrencyPairFormatConfig
	Websocket                                  *Websocket
	*request.Requester
}

// IBotExchange enforces standard functions for all exchanges supported in
// GoCryptoTrader
type IBotExchange interface {
	Setup(exch config.ExchangeConfig)
	Start(wg *sync.WaitGroup)
	SetDefaults()
	GetName() string
	IsEnabled() bool
	SetEnabled(bool)
	GetTickerPrice(currency pair.CurrencyPair, assetType string) (ticker.Price, error)
	UpdateTicker(currency pair.CurrencyPair, assetType string) (ticker.Price, error)
	GetOrderbookEx(currency pair.CurrencyPair, assetType string) (orderbook.Base, error)
	UpdateOrderbook(currency pair.CurrencyPair, assetType string) (orderbook.Base, error)
	GetEnabledCurrencies() []pair.CurrencyPair
	GetAvailableCurrencies() []pair.CurrencyPair
	GetAssetTypes() []string
	GetAccountInfo() (AccountInfo, error)
	GetAuthenticatedAPISupport() bool
	SetCurrencies(pairs []pair.CurrencyPair, enabledPairs bool) error
	GetExchangeHistory(pair.CurrencyPair, string) ([]TradeHistory, error)
	SupportsAutoPairUpdates() bool
	GetLastPairsUpdateTime() int64
	SupportsRESTTickerBatchUpdates() bool

	GetWithdrawPermissions() uint32
	FormatWithdrawPermissions() string
	SupportsWithdrawPermissions(permissions uint32) bool

	GetFundingHistory() ([]FundHistory, error)
	SubmitOrder(p pair.CurrencyPair, side OrderSide, orderType OrderType, amount, price float64, clientID string) (SubmitOrderResponse, error)
	ModifyOrder(orderID int64, modify ModifyOrder) (int64, error)
	CancelOrder(order OrderCancellation) error
	CancelAllOrders(orders OrderCancellation) error
	GetOrderInfo(orderID int64) (OrderDetail, error)
	GetDepositAddress(cryptocurrency pair.CurrencyItem) (string, error)

	WithdrawCryptocurrencyFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error)
	WithdrawFiatFunds(currency pair.CurrencyItem, amount float64) (string, error)

	GetWebsocket() (*Websocket, error)
}

// SupportsRESTTickerBatchUpdates returns whether or not the
// exhange supports REST batch ticker fetching
func (e *Base) SupportsRESTTickerBatchUpdates() bool {
	return e.SupportsRESTTickerBatching
}

// SetHTTPClientTimeout sets the timeout value for the exchanges
// HTTP Client
func (e *Base) SetHTTPClientTimeout(t time.Duration) {
	if e.Requester == nil {
		e.Requester = request.New(e.Name,
			request.NewRateLimit(time.Second, 0),
			request.NewRateLimit(time.Second, 0),
			new(http.Client))
	}
	e.Requester.HTTPClient.Timeout = t
}

// SetHTTPClient sets exchanges HTTP client
func (e *Base) SetHTTPClient(h *http.Client) {
	if e.Requester == nil {
		e.Requester = request.New(e.Name,
			request.NewRateLimit(time.Second, 0),
			request.NewRateLimit(time.Second, 0),
			new(http.Client))
	}
	e.Requester.HTTPClient = h
}

// GetHTTPClient gets the exchanges HTTP client
func (e *Base) GetHTTPClient() *http.Client {
	if e.Requester == nil {
		e.Requester = request.New(e.Name,
			request.NewRateLimit(time.Second, 0),
			request.NewRateLimit(time.Second, 0),
			new(http.Client))
	}
	return e.Requester.HTTPClient
}

// SetHTTPClientUserAgent sets the exchanges HTTP user agent
func (e *Base) SetHTTPClientUserAgent(ua string) {
	if e.Requester == nil {
		e.Requester = request.New(e.Name,
			request.NewRateLimit(time.Second, 0),
			request.NewRateLimit(time.Second, 0),
			new(http.Client))
	}
	e.Requester.UserAgent = ua
	e.HTTPUserAgent = ua
}

// GetHTTPClientUserAgent gets the exchanges HTTP user agent
func (e *Base) GetHTTPClientUserAgent() string {
	return e.HTTPUserAgent
}

// SetClientProxyAddress sets a proxy address for REST and websocket requests
func (e *Base) SetClientProxyAddress(addr string) error {
	if addr != "" {
		proxy, err := url.Parse(addr)
		if err != nil {
			return fmt.Errorf("exchange.go - setting proxy address error %s",
				err)
		}

		err = e.Requester.SetProxy(proxy)
		if err != nil {
			return fmt.Errorf("exchange.go - setting proxy address error %s",
				err)
		}

		if e.Websocket != nil {
			err = e.Websocket.SetProxyAddress(addr)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SetAutoPairDefaults sets the default values for whether or not the exchange
// supports auto pair updating or not
func (e *Base) SetAutoPairDefaults() error {
	cfg := config.GetConfig()
	exch, err := cfg.GetExchangeConfig(e.Name)
	if err != nil {
		return err
	}

	update := false
	if e.SupportsAutoPairUpdating {
		if !exch.SupportsAutoPairUpdates {
			exch.SupportsAutoPairUpdates = true
			exch.PairsLastUpdated = 0
			update = true
		}
	} else {
		if exch.PairsLastUpdated == 0 {
			exch.PairsLastUpdated = time.Now().Unix()
			e.PairsLastUpdated = exch.PairsLastUpdated
			update = true
		}
	}

	if update {
		return cfg.UpdateExchangeConfig(exch)
	}
	return nil
}

// SupportsAutoPairUpdates returns whether or not the exchange supports
// auto currency pair updating
func (e *Base) SupportsAutoPairUpdates() bool {
	return e.SupportsAutoPairUpdating
}

// GetLastPairsUpdateTime returns the unix timestamp of when the exchanges
// currency pairs were last updated
func (e *Base) GetLastPairsUpdateTime() int64 {
	return e.PairsLastUpdated
}

// SetAssetTypes checks the exchange asset types (whether it supports SPOT,
// Binary or Futures) and sets it to a default setting if it doesn't exist
func (e *Base) SetAssetTypes() error {
	cfg := config.GetConfig()
	exch, err := cfg.GetExchangeConfig(e.Name)
	if err != nil {
		return err
	}

	update := false
	if exch.AssetTypes == "" {
		exch.AssetTypes = common.JoinStrings(e.AssetTypes, ",")
		update = true
	} else {
		exch.AssetTypes = common.JoinStrings(e.AssetTypes, ",")
		update = true
	}

	if update {
		return cfg.UpdateExchangeConfig(exch)
	}

	return nil
}

// GetAssetTypes returns the available asset types for an individual exchange
func (e *Base) GetAssetTypes() []string {
	return e.AssetTypes
}

// GetExchangeAssetTypes returns the asset types the exchange supports (SPOT,
// binary, futures)
func GetExchangeAssetTypes(exchName string) ([]string, error) {
	cfg := config.GetConfig()
	exch, err := cfg.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	return common.SplitStrings(exch.AssetTypes, ","), nil
}

// GetClientBankAccounts returns banking details associated with
// a client for withdrawal purposes
func (e *Base) GetClientBankAccounts(exchangeName, withdrawalCurrency string) (config.BankAccount, error) {
	cfg := config.GetConfig()
	return cfg.GetClientBankAccounts(exchangeName, withdrawalCurrency)
}

// GetExchangeBankAccounts returns banking details associated with an
// exchange for funding purposes
func (e *Base) GetExchangeBankAccounts(exchangeName, depositCurrency string) (config.BankAccount, error) {
	cfg := config.GetConfig()
	return cfg.GetExchangeBankAccounts(exchangeName, depositCurrency)
}

// CompareCurrencyPairFormats checks and returns whether or not the two supplied
// config currency pairs match
func CompareCurrencyPairFormats(pair1 config.CurrencyPairFormatConfig, pair2 *config.CurrencyPairFormatConfig) bool {
	if pair1.Delimiter != pair2.Delimiter ||
		pair1.Uppercase != pair2.Uppercase ||
		pair1.Separator != pair2.Separator ||
		pair1.Index != pair2.Index {
		return false
	}
	return true
}

// SetCurrencyPairFormat checks the exchange request and config currency pair
// formats and sets it to a default setting if it doesn't exist
func (e *Base) SetCurrencyPairFormat() error {
	cfg := config.GetConfig()
	exch, err := cfg.GetExchangeConfig(e.Name)
	if err != nil {
		return err
	}

	update := false
	if exch.RequestCurrencyPairFormat == nil {
		exch.RequestCurrencyPairFormat = &config.CurrencyPairFormatConfig{
			Delimiter: e.RequestCurrencyPairFormat.Delimiter,
			Uppercase: e.RequestCurrencyPairFormat.Uppercase,
			Separator: e.RequestCurrencyPairFormat.Separator,
			Index:     e.RequestCurrencyPairFormat.Index,
		}
		update = true
	} else {
		if CompareCurrencyPairFormats(e.RequestCurrencyPairFormat,
			exch.RequestCurrencyPairFormat) {
			e.RequestCurrencyPairFormat = *exch.RequestCurrencyPairFormat
		} else {
			*exch.RequestCurrencyPairFormat = e.RequestCurrencyPairFormat
			update = true
		}
	}

	if exch.ConfigCurrencyPairFormat == nil {
		exch.ConfigCurrencyPairFormat = &config.CurrencyPairFormatConfig{
			Delimiter: e.ConfigCurrencyPairFormat.Delimiter,
			Uppercase: e.ConfigCurrencyPairFormat.Uppercase,
			Separator: e.ConfigCurrencyPairFormat.Separator,
			Index:     e.ConfigCurrencyPairFormat.Index,
		}
		update = true
	} else {
		if CompareCurrencyPairFormats(e.ConfigCurrencyPairFormat,
			exch.ConfigCurrencyPairFormat) {
			e.ConfigCurrencyPairFormat = *exch.ConfigCurrencyPairFormat
		} else {
			*exch.ConfigCurrencyPairFormat = e.ConfigCurrencyPairFormat
			update = true
		}
	}

	if update {
		return cfg.UpdateExchangeConfig(exch)
	}
	return nil
}

// GetAuthenticatedAPISupport returns whether the exchange supports
// authenticated API requests
func (e *Base) GetAuthenticatedAPISupport() bool {
	return e.AuthenticatedAPISupport
}

// GetName is a method that returns the name of the exchange base
func (e *Base) GetName() string {
	return e.Name
}

// GetEnabledCurrencies is a method that returns the enabled currency pairs of
// the exchange base
func (e *Base) GetEnabledCurrencies() []pair.CurrencyPair {
	return pair.FormatPairs(e.EnabledPairs,
		e.ConfigCurrencyPairFormat.Delimiter,
		e.ConfigCurrencyPairFormat.Index)
}

// GetAvailableCurrencies is a method that returns the available currency pairs
// of the exchange base
func (e *Base) GetAvailableCurrencies() []pair.CurrencyPair {
	return pair.FormatPairs(e.AvailablePairs,
		e.ConfigCurrencyPairFormat.Delimiter,
		e.ConfigCurrencyPairFormat.Index)
}

// SupportsCurrency returns true or not whether a currency pair exists in the
// exchange available currencies or not
func (e *Base) SupportsCurrency(p pair.CurrencyPair, enabledPairs bool) bool {
	if enabledPairs {
		return pair.Contains(e.GetEnabledCurrencies(), p, false)
	}
	return pair.Contains(e.GetAvailableCurrencies(), p, false)
}

// GetExchangeFormatCurrencySeperator returns whether or not a specific
// exchange contains a separator used for API requests
func GetExchangeFormatCurrencySeperator(exchName string) bool {
	cfg := config.GetConfig()
	exch, err := cfg.GetExchangeConfig(exchName)
	if err != nil {
		return false
	}

	if exch.RequestCurrencyPairFormat.Separator != "" {
		return true
	}
	return false
}

// GetAndFormatExchangeCurrencies returns a pair.CurrencyItem string containing
// the exchanges formatted currency pairs
func GetAndFormatExchangeCurrencies(exchName string, pairs []pair.CurrencyPair) (pair.CurrencyItem, error) {
	var currencyItems pair.CurrencyItem
	cfg := config.GetConfig()
	exch, err := cfg.GetExchangeConfig(exchName)
	if err != nil {
		return currencyItems, err
	}

	for x := range pairs {
		currencyItems += FormatExchangeCurrency(exchName, pairs[x])
		if x == len(pairs)-1 {
			continue
		}
		currencyItems += pair.CurrencyItem(exch.RequestCurrencyPairFormat.Separator)
	}
	return currencyItems, nil
}

// FormatExchangeCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
func FormatExchangeCurrency(exchName string, p pair.CurrencyPair) pair.CurrencyItem {
	cfg := config.GetConfig()
	exch, _ := cfg.GetExchangeConfig(exchName)

	return p.Display(exch.RequestCurrencyPairFormat.Delimiter,
		exch.RequestCurrencyPairFormat.Uppercase)
}

// FormatCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
func FormatCurrency(p pair.CurrencyPair) pair.CurrencyItem {
	cfg := config.GetConfig()
	return p.Display(cfg.Currency.CurrencyPairFormat.Delimiter,
		cfg.Currency.CurrencyPairFormat.Uppercase)
}

// SetEnabled is a method that sets if the exchange is enabled
func (e *Base) SetEnabled(enabled bool) {
	e.Enabled = enabled
}

// IsEnabled is a method that returns if the current exchange is enabled
func (e *Base) IsEnabled() bool {
	return e.Enabled
}

// SetAPIKeys is a method that sets the current API keys for the exchange
func (e *Base) SetAPIKeys(APIKey, APISecret, ClientID string, b64Decode bool) {
	if !e.AuthenticatedAPISupport {
		return
	}

	e.APIKey = APIKey
	e.ClientID = ClientID

	if b64Decode {
		result, err := common.Base64Decode(APISecret)
		if err != nil {
			e.AuthenticatedAPISupport = false
			log.Printf(warningBase64DecryptSecretKeyFailed, e.Name)
		}
		e.APISecret = string(result)
	} else {
		e.APISecret = APISecret
	}
}

// SetCurrencies sets the exchange currency pairs for either enabledPairs or
// availablePairs
func (e *Base) SetCurrencies(pairs []pair.CurrencyPair, enabledPairs bool) error {
	if len(pairs) == 0 {
		return fmt.Errorf("%s SetCurrencies error - pairs is empty", e.Name)
	}

	cfg := config.GetConfig()
	exchCfg, err := cfg.GetExchangeConfig(e.Name)
	if err != nil {
		return err
	}

	var pairsStr []string
	for x := range pairs {
		pairsStr = append(pairsStr, pairs[x].Display(exchCfg.ConfigCurrencyPairFormat.Delimiter,
			exchCfg.ConfigCurrencyPairFormat.Uppercase).String())
	}

	if enabledPairs {
		exchCfg.EnabledPairs = common.JoinStrings(pairsStr, ",")
		e.EnabledPairs = pairsStr
	} else {
		exchCfg.AvailablePairs = common.JoinStrings(pairsStr, ",")
		e.AvailablePairs = pairsStr
	}

	return cfg.UpdateExchangeConfig(exchCfg)
}

// UpdateCurrencies updates the exchange currency pairs for either enabledPairs or
// availablePairs
func (e *Base) UpdateCurrencies(exchangeProducts []string, enabled, force bool) error {
	if len(exchangeProducts) == 0 {
		return fmt.Errorf("%s UpdateCurrencies error - exchangeProducts is empty", e.Name)
	}

	exchangeProducts = common.SplitStrings(common.StringToUpper(common.JoinStrings(exchangeProducts, ",")), ",")
	var products []string

	for x := range exchangeProducts {
		if exchangeProducts[x] == "" {
			continue
		}
		products = append(products, exchangeProducts[x])
	}

	var newPairs, removedPairs []string
	var updateType string

	if enabled {
		newPairs, removedPairs = pair.FindPairDifferences(e.EnabledPairs, products)
		updateType = "enabled"
	} else {
		newPairs, removedPairs = pair.FindPairDifferences(e.AvailablePairs, products)
		updateType = "available"
	}

	if force || len(newPairs) > 0 || len(removedPairs) > 0 {
		cfg := config.GetConfig()
		exch, err := cfg.GetExchangeConfig(e.Name)
		if err != nil {
			return err
		}

		if force {
			log.Printf("%s forced update of %s pairs.", e.Name, updateType)
		} else {
			if len(newPairs) > 0 {
				log.Printf("%s Updating pairs - New: %s.\n", e.Name, newPairs)
			}
			if len(removedPairs) > 0 {
				log.Printf("%s Updating pairs - Removed: %s.\n", e.Name, removedPairs)
			}
		}

		if enabled {
			exch.EnabledPairs = common.JoinStrings(products, ",")
			e.EnabledPairs = products
		} else {
			exch.AvailablePairs = common.JoinStrings(products, ",")
			e.AvailablePairs = products
		}
		return cfg.UpdateExchangeConfig(exch)
	}
	return nil
}

// ModifyOrder is a an order modifyer
type ModifyOrder struct {
	OrderType
	OrderSide
	Price  float64
	Amount float64
}

// Format holds exchange formatting
type Format struct {
	ExchangeName string
	OrderType    map[string]string
	OrderSide    map[string]string
}

// Formatting contain a range of exchanges formatting
type Formatting []Format

// OrderType enforces a standard for Ordertypes across the code base
type OrderType string

// OrderType ...types
const (
	Limit  OrderType = "Limit"
	Market OrderType = "Market"
)

// ToString changes the ordertype to the exchange standard and returns a string
func (o OrderType) ToString() string {
	return fmt.Sprintf("%v", o)
}

// OrderSide enforces a standard for OrderSides across the code base
type OrderSide string

// OrderSide types
const (
	Buy  OrderSide = "Buy"
	Sell OrderSide = "Sell"
)

// ToString changes the ordertype to the exchange standard and returns a string
func (o OrderSide) ToString() string {
	return fmt.Sprintf("%v", o)
}

// SetAPIURL sets configuration API URL for an exchange
func (e *Base) SetAPIURL(ec config.ExchangeConfig) error {
	if ec.APIURL == "" || ec.APIURLSecondary == "" {
		return errors.New("SetAPIURL error variable zero value")
	}
	if ec.APIURL != config.APIURLNonDefaultMessage {
		e.APIUrl = ec.APIURL
	}
	if ec.APIURLSecondary != config.APIURLNonDefaultMessage {
		e.APIUrlSecondary = ec.APIURLSecondary
	}
	return nil
}

// GetAPIURL returns the set API URL
func (e *Base) GetAPIURL() string {
	return e.APIUrl
}

// GetSecondaryAPIURL returns the set Secondary API URL
func (e *Base) GetSecondaryAPIURL() string {
	return e.APIUrlSecondary
}

// GetAPIURLDefault returns exchange default URL
func (e *Base) GetAPIURLDefault() string {
	return e.APIUrlDefault
}

// GetAPIURLSecondaryDefault returns exchange default secondary URL
func (e *Base) GetAPIURLSecondaryDefault() string {
	return e.APIUrlSecondaryDefault
}

// GetWithdrawPermissions passes through the exchange's withdraw permissions
func (e *Base) GetWithdrawPermissions() uint32 {
	return e.APIWithdrawPermissions
}

// SupportsWithdrawPermissions compares the supplied permissions with the exchange's to verify they're supported
func (e *Base) SupportsWithdrawPermissions(permissions uint32) bool {
	exchangePermissions := e.GetWithdrawPermissions()
	if permissions&exchangePermissions == permissions {
		return true
	}
	return false
}

// FormatWithdrawPermissions will return each of the exchange's compatible withdrawal methods in readable form
func (e *Base) FormatWithdrawPermissions() string {
	services := []string{}
	for i := 0; i < 32; i++ {
		var check uint32 = 1 << uint32(i)
		if e.GetWithdrawPermissions()&check != 0 {
			switch check {
			case AutoWithdrawCrypto:
				services = append(services, AutoWithdrawCryptoText)
			case AutoWithdrawCryptoWithAPIPermission:
				services = append(services, AutoWithdrawCryptoWithAPIPermissionText)
			case AutoWithdrawCryptoWithSetup:
				services = append(services, AutoWithdrawCryptoWithSetupText)
			case WithdrawCryptoWith2FA:
				services = append(services, WithdrawCryptoWith2FAText)
			case WithdrawCryptoWithSMS:
				services = append(services, WithdrawCryptoWithSMSText)
			case WithdrawCryptoWithEmail:
				services = append(services, WithdrawCryptoWithEmailText)
			case WithdrawCryptoWithWebsiteApproval:
				services = append(services, WithdrawCryptoWithWebsiteApprovalText)
			case WithdrawCryptoWithAPIPermission:
				services = append(services, WithdrawCryptoWithAPIPermissionText)
			case AutoWithdrawFiat:
				services = append(services, AutoWithdrawFiatText)
			case AutoWithdrawFiatWithAPIPermission:
				services = append(services, AutoWithdrawFiatWithAPIPermissionText)
			case AutoWithdrawFiatWithSetup:
				services = append(services, AutoWithdrawFiatWithSetupText)
			case WithdrawFiatWith2FA:
				services = append(services, WithdrawFiatWith2FAText)
			case WithdrawFiatWithSMS:
				services = append(services, WithdrawFiatWithSMSText)
			case WithdrawFiatWithEmail:
				services = append(services, WithdrawFiatWithEmailText)
			case WithdrawFiatWithWebsiteApproval:
				services = append(services, WithdrawFiatWithWebsiteApprovalText)
			case WithdrawFiatWithAPIPermission:
				services = append(services, WithdrawFiatWithAPIPermissionText)
			case WithdrawCryptoViaWebsiteOnly:
				services = append(services, WithdrawCryptoViaWebsiteOnlyText)
			case WithdrawFiatViaWebsiteOnly:
				services = append(services, WithdrawFiatViaWebsiteOnlyText)
			default:
				services = append(services, fmt.Sprintf("%s[1<<%v]", UnknownWithdrawalTypeText, i))
			}
		}
	}
	if len(services) > 0 {
		return strings.Join(services, " & ")
	}

	return NoAPIWithdrawalMethodsText
}
