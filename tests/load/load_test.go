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
	baseURL           = "http://localhost:9000"
	warmupDuration    = 30 * time.Second
	testDuration      = 3 * time.Minute
	coolDownDuration  = 10 * time.Second
	maxWorkers        = 96
	connections       = 200
	userCount         = 1000
	orderIDPrefix     = "loadtest-order"
	healthCheckRetry  = 5
	createOrderWeight = 60
	getHistoryWeight  = 40
	targetRPS         = 800
)

var (
	orderCounter atomic.Int64
	users        = make([]string, 0, userCount)
	usersMu      sync.Mutex
	httpClient   = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: connections,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 5 * time.Second,
	}
)

func TestLoad(t *testing.T) {
	if !healthCheck() {
		t.Fatal("Сервис недоступен")
	}

	t.Run("Подготовка пользователей", func(t *testing.T) {
		start := time.Now()
		if err := registerAndLoginUsers(); err != nil {
			t.Fatalf("Ошибка подготовки пользователей: %v", err)
		}
		t.Logf("Подготовлено %d пользователей за %v", userCount, time.Since(start))
	})

	t.Run("Прогрев системы", func(t *testing.T) {
		runAttack("Warmup", mixedTargeter(), targetRPS/4, warmupDuration, false)
	})

	t.Run("Основной тест", func(t *testing.T) {
		runAttack("MainTest", mixedTargeter(), targetRPS, testDuration, true)
	})

	t.Run("Завершение", func(t *testing.T) {
		time.Sleep(coolDownDuration)
	})
}

func mixedTargeter() vegeta.Targeter {
	return func(t *vegeta.Target) error {
		if rand.Intn(100) < createOrderWeight {
			return createOrderTargeter()(t)
		}
		return historyTargeter()(t)
	}
}

func runAttack(name string, targeter vegeta.Targeter, rate int, duration time.Duration, detailed bool) {
	attacker := vegeta.NewAttacker(
		vegeta.Timeout(5*time.Second),
		vegeta.Workers(maxWorkers),
		vegeta.Connections(connections),
		vegeta.KeepAlive(true),
	)

	ratePacer := vegeta.Rate{Freq: rate, Per: time.Second}
	metrics := &vegeta.Metrics{}

	for res := range attacker.Attack(targeter, ratePacer, duration, name) {
		metrics.Add(res)
		if detailed && (res.Error != "" || res.Code >= 400) {
			log.Printf("[%s] Ошибка: %s URL: %s Code: %d\n",
				name, res.Error, res.URL, res.Code)
		}
	}

	metrics.Close()
	generateReport(name, metrics)

	if detailed {
		log.Printf("[%s] Итоги:\nЗапросов: %d\nRPS: %.1f\nУспешных: %.2f%%\nLatency 50/95/99: %v/%v/%v\n",
			name,
			metrics.Requests,
			metrics.Rate,
			metrics.Success*100,
			metrics.Latencies.P50,
			metrics.Latencies.P95,
			metrics.Latencies.P99)
	}
}

func healthCheck() bool {
	for range healthCheckRetry {
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
	for i := range userCount {
		go func(i int) {
			defer wg.Done()
			if err := registerUser(i); err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)

	if errs := collectErrors(errCh); len(errs) > 0 {
		return fmt.Errorf("ошибки регистрации: %v", errs)
	}

	errCh = make(chan error, userCount)
	wg.Add(userCount)
	for i := range userCount {
		go func(i int) {
			defer wg.Done()
			if err := loginUser(i); err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)

	if errs := collectErrors(errCh); len(errs) > 0 {
		return fmt.Errorf("ошибки логина: %v", errs)
	}

	return nil
}

func collectErrors(errCh <-chan error) []error {
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	return errs
}

func registerUser(i int) error {
	email := fmt.Sprintf("loaduser%d@test.com", i)
	body := fmt.Sprintf(`{
	"email":"%s",
	"password":"%s"
	}`, email, generatePassword(i))

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
	body := fmt.Sprintf(`{
	"email":"%s",
	"password":"%s"
	}`, email, generatePassword(i))

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

func generateReport(name string, metrics *vegeta.Metrics) {
	txtFile, _ := os.Create(name + "_report.txt")
	defer txtFile.Close()
	vegeta.NewTextReporter(metrics).Report(txtFile)

	jsonFile, _ := os.Create(name + "_metrics.json")
	defer jsonFile.Close()
	vegeta.NewJSONReporter(metrics).Report(jsonFile)
}

func generatePassword(i int) string {
	return fmt.Sprintf("P@ssw0rd-%d", i)
}

func generateOrderPayload(id int64) string {
	return fmt.Sprintf(
		`{
		"id":"%s-%d",
		"recipient_id":"user-%d",
		"expiry":"2025-04-01",
		"base_price":1000,
		"weight":2.5,
		"packaging":"коробка+пленка"
		}`,
		orderIDPrefix, id, rand.Intn(userCount),
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
	ticker *time.Ticker
	done   chan struct{}
	start  time.Time
	count  atomic.Int64
}

func newProgressReporter(name string) *progressReporter {
	p := &progressReporter{
		name:   name,
		ticker: time.NewTicker(5 * time.Second),
		done:   make(chan struct{}),
		start:  time.Now(),
	}
	go p.report()
	return p
}

func (p *progressReporter) tick() {
	p.count.Add(1)
}

func (p *progressReporter) report() {
	for {
		select {
		case <-p.ticker.C:
			elapsed := time.Since(p.start).Seconds()
			log.Printf("[%s] Прогресс: %d запросов (%.1f RPS)",
				p.name, p.count.Load(), float64(p.count.Load())/elapsed)
		case <-p.done:
			p.ticker.Stop()
			return
		}
	}
}

func (p *progressReporter) stop() {
	close(p.done)
}
