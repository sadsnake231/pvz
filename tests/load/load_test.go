package load_test

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

const (
	baseURL          = "http://localhost:9000"
	targetRPS        = 50
	testDuration     = 1 * time.Minute
	maxWorkers       = 200
	userCount        = 500
	orderIDPrefix    = "loadtest-order"
	healthCheckRetry = 5
)

var (
	orderCounter atomic.Int64
	users        = make([]string, 0, userCount)
	usersMu      sync.Mutex
	httpClient   = &http.Client{Timeout: 10 * time.Second}
)

func TestLoad(t *testing.T) {
	t.Run("Регистрация и логин", func(t *testing.T) {
		if err := registerAndLoginUsers(); err != nil {
			t.Fatalf("Ошибка подготовки пользователей: %v", err)
		}
	})

	t.Run("Нагрузочное тестирование", func(t *testing.T) {
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			runAttack("CreateOrders", createOrderTargeter(), targetRPS)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			runAttack("GetHistory", historyTargeter(), targetRPS)
		}()

		wg.Wait()
	})
}

func healthCheck() bool {
	for i := 0; i < healthCheckRetry; i++ {
		resp, err := httpClient.Get(baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			return true
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

func registerAndLoginUsers() error {
	var wg sync.WaitGroup
	errCh := make(chan error, userCount)

	wg.Add(userCount)
	for i := 0; i < userCount; i++ {
		go func(i int) {
			defer wg.Done()
			if err := registerUser(i); err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("ошибки при регистрации: %v", errs)
	}

	errCh = make(chan error, userCount)
	wg.Add(userCount)
	for i := 0; i < userCount; i++ {
		go func(i int) {
			defer wg.Done()
			if err := loginUser(i); err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)

	errs = nil
	for err := range errCh {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("ошибки при логине: %v", errs)
	}

	return nil
}

func registerUser(i int) error {
	email := fmt.Sprintf("loaduser%d@test.com", i)
	body := fmt.Sprintf(
		`{
		"email":"%s",
		"password":"%s"
		}`,
		email, generatePassword(i),
	)

	req, _ := http.NewRequest("POST", baseURL+"/users/signup", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("ошибка регистрации: %s", resp.Status)
	}
	return nil
}

func loginUser(i int) error {
	email := fmt.Sprintf("loaduser%d@test.com", i)
	body := fmt.Sprintf(
		`{
		"email":"%s",
		"password":"%s"
		}`,
		email, generatePassword(i),
	)

	req, _ := http.NewRequest("POST", baseURL+"/users/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка логина: %s", resp.Status)
	}

	for _, c := range resp.Cookies() {
		if c.Name == "jwt" {
			usersMu.Lock()
			users = append(users, c.String())
			usersMu.Unlock()
			break
		}
	}
	return nil
}

func createOrderTargeter() vegeta.Targeter {
	return func(t *vegeta.Target) error {
		if t == nil {
			return vegeta.ErrNilTarget
		}

		user := getRandomUser()
		if user == "" {
			return fmt.Errorf("нет доступных пользователей")
		}

		id := orderCounter.Add(1)
		*t = vegeta.Target{
			Method: "POST",
			URL:    baseURL + "/orders",
			Body:   []byte(generateOrderPayload(id)),
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"Cookie":       []string{user},
			},
		}
		return nil
	}
}

func historyTargeter() vegeta.Targeter {
	return func(t *vegeta.Target) error {
		if t == nil {
			return vegeta.ErrNilTarget
		}

		user := getRandomUser()
		if user == "" {
			return fmt.Errorf("нет доступных пользователей")
		}

		*t = vegeta.Target{
			Method: "GET",
			URL:    baseURL + "/reports/history/v2",
			Header: http.Header{
				"Cookie": []string{user},
			},
		}
		return nil
	}
}

func runAttack(name string, targeter vegeta.Targeter, rate int) {
	attacker := vegeta.NewAttacker(
		vegeta.Timeout(10*time.Second),
		vegeta.Workers(maxWorkers),
		vegeta.KeepAlive(false),
		vegeta.Connections(1000),
	)

	pacer := vegeta.ConstantPacer{Freq: rate, Per: time.Second}
	metrics := vegeta.Metrics{}

	var (
		mu       sync.Mutex
		progress = newProgressReporter(name, rate)
	)

	for res := range attacker.Attack(targeter, pacer, testDuration, name) {
		mu.Lock()
		metrics.Add(res)
		mu.Unlock()

		progress.tick()

		if res.Error != "" || res.Code >= 400 {
			log.Printf("[%s] Ошибка: %s URL: %s Code: %d\n",
				name, res.Error, res.URL, res.Code)
		}
	}

	metrics.Close()
	generateReport(name, &metrics)
	progress.stop()
}

func generateReport(name string, metrics *vegeta.Metrics) {
	txtReport := vegeta.NewTextReporter(metrics)
	txtFile, _ := os.Create(name + "_report.txt")
	defer txtFile.Close()
	txtReport.Report(txtFile)

	jsonReport := vegeta.NewJSONReporter(metrics)
	jsonFile, _ := os.Create(name + "_metrics.json")
	defer jsonFile.Close()
	jsonReport.Report(jsonFile)
}

func generatePassword(i int) string {
	return fmt.Sprintf("P@ssw0rd-%d", i)
}

func generateOrderPayload(id int64) string {
	return fmt.Sprintf(
		`{
			"id": "%s-%d",
			"recipient_id": "user-%d",
			"expiry": "2025-04-01",
			"base_price": 1000,
			"weight": 2.5,
			"packaging": "коробка+пленка"
		}`,
		orderIDPrefix,
		id,
		rand.Intn(userCount),
	)
}

func getRandomUser() string {
	usersMu.Lock()
	defer usersMu.Unlock()

	if len(users) == 0 {
		return ""
	}
	return users[rand.Intn(len(users))]
}

type progressReporter struct {
	name   string
	rate   int
	ticker *time.Ticker
	done   chan struct{}
	mu     sync.Mutex
	count  int64
	start  time.Time
}

func newProgressReporter(name string, rate int) *progressReporter {
	p := &progressReporter{
		name:   name,
		rate:   rate,
		ticker: time.NewTicker(5 * time.Second),
		done:   make(chan struct{}),
		start:  time.Now(),
	}

	go p.report()
	return p
}

func (p *progressReporter) tick() {
	atomic.AddInt64(&p.count, 1)
}

func (p *progressReporter) report() {
	for {
		select {
		case <-p.ticker.C:
			elapsed := time.Since(p.start).Seconds()
			current := atomic.LoadInt64(&p.count)
			rps := float64(current) / elapsed

			log.Printf("[%s] Прогресс: %d запросов (%.1f RPS)\n",
				p.name, current, rps)

		case <-p.done:
			p.ticker.Stop()
			return
		}
	}
}

func (p *progressReporter) stop() {
	close(p.done)
}
