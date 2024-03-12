package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
)

const (
	// Use simple os.Getenv with predefined variables.
	// A more proper approach would be using something like Viper/Cobra.
	CFG_ENV_API_KEY  = "API_KEY"
	CFG_ENV_CURRENCY = "CURRENCY"
	CFG_ENV_MARKET   = "MARKET"

	DATA_API_ENDPOINT = "https://www.alphavantage.co/query?function=DIGITAL_CURRENCY_DAILY&symbol=%v&market=%v&apikey=%v"
)

var (
	// Since the free API key is rate limited to 25 requests per hour, refresh on specific intervals.
	// A more proper approach would be aynchronously refreshing if the data is determined to be stale
	// when processing a request, but it is out of the time box for the task.
	DEFAULT_REFRESH_INTERVAL = time.Hour * 2
	DEFAULT_RETRY_INTERVAL   = time.Second * 5
)

type DataAPI struct {
	Http   *fiber.App
	client *fiber.Client

	apikey   string
	currency string
	market   string

	data map[string]interface{}
	mu   sync.RWMutex
}

func (d *DataAPI) LivenessProbeHandler(c *fiber.Ctx) error {
	c.SendStatus(fiber.StatusOK)
	return c.Send(nil)
}

func (d *DataAPI) ReadinessProbeHandler(c *fiber.Ctx) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.data == nil {
		c.SendStatus(fiber.StatusServiceUnavailable)
	} else {
		c.SendStatus(fiber.StatusOK)
	}

	return c.Send(nil)
}

func (d *DataAPI) GetDataHandler(c *fiber.Ctx) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.data == nil {
		// In the more advanced case we could implement an asynchronous refresh here using channels
		c.SendStatus(fiber.StatusInternalServerError)
		return c.Send(nil)
	} else {
		// Necessary repocessing of data for the internal use / formatting of the output may go here
		// or be implemented in the fetcher goroutine (refreshData)
		return c.JSON(d.data)
	}
}

func (d *DataAPI) refreshData() {
	var refreshInterval time.Duration

	for {
		if refreshInterval > 0 {
			log.Printf("Sleeping for %d seconds before refreshing the data",
				refreshInterval/time.Second)

			select {
			// Signals handling etc. in the more advanced case goes here
			case <-time.After(refreshInterval):
			}
		}

		refreshInterval = DEFAULT_RETRY_INTERVAL

		log.Printf("Refreshing the data")
		url := fmt.Sprintf(DATA_API_ENDPOINT, d.currency, d.market, d.apikey)
		agent := d.client.Get(url)

		status, body, errs := agent.Bytes()
		if status != fiber.StatusOK || len(errs) > 0 {
			log.Printf("Error fetching data: status %v: %#v", status, errs)
			continue
		}

		log.Printf("Parsing fetched data")
		data := make(map[string]interface{})
		if err := json.Unmarshal(body, &data); err != nil {
			log.Printf("Cannot parse fetched data: %v", err)
			continue
		}

		if errmsg, ok := data["Error Message"]; ok {
			log.Printf("Fetch API error: %v", errmsg)
			continue
		}

		// Necessary repocessing of data for the internal use / formatting of the output may go here

		d.mu.Lock()
		d.data = data
		d.mu.Unlock()

		log.Printf("Refresh succeeded")
		refreshInterval = DEFAULT_REFRESH_INTERVAL
	}
}

func (d *DataAPI) Run() {
	go d.refreshData()
	d.Http.Listen("0.0.0.0:8080")
}

func NewDataAPI() *DataAPI {

	apikey := os.Getenv(CFG_ENV_API_KEY)
	if len(apikey) == 0 {
		log.Fatal("No API key provided")
	}

	currency := os.Getenv(CFG_ENV_CURRENCY)
	if len(currency) == 0 {
		log.Fatal("No currency ticker provided")
	}

	market := os.Getenv(CFG_ENV_MARKET)
	if len(market) == 0 {
		log.Fatal("No market name provided")
	}

	dataAPI := &DataAPI{
		client: &fiber.Client{},

		apikey:   apikey,
		currency: currency,
		market:   market,
	}

	app := fiber.New(fiber.Config{
		// Faster JSON processing
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})

	app.Get("/", dataAPI.GetDataHandler)
	app.Get("/healthz/live", dataAPI.LivenessProbeHandler)
	app.Get("/healthz/ready", dataAPI.ReadinessProbeHandler)
	// An internal handler to force-refresh the data can be added unless the asynchronous fetch
	// is implemented as described above.

	dataAPI.Http = app
	return dataAPI
}

func main() {
	app := NewDataAPI()
	app.Run()
}
